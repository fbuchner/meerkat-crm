package controllers

import (
	"bytes"
	"encoding/json"
	"mycorrhizal/database"
	"mycorrhizal/models"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelationshipEdgeAndGraph_RealMigratedSchema is the real-DB check for
// §3d WP1+WP2 (docs/fork-plan/95-backlog-and-priorities.md): every other
// test in this package uses AutoMigrate against :memory: sqlite, which
// derives its schema from the same Go struct tags the application code
// uses -- it cannot catch a GORM column-tag mismatch against the real
// migration SQL (this fork's own recurring bug class, e.g.
// ContactSyncLink.ETag). This test runs the new RelationshipEdge CRUD
// routes and the rewired GetGraph against a database.InitDB-migrated real
// file database instead, exercising create (both an existing-contact and a
// thin-contact endpoint) -> list (both filters) -> update -> accept ->
// delete, plus a GET /graph pass confirming only confirmed edges appear.
func TestRelationshipEdgeAndGraph_RealMigratedSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "relationship-edge-real.db")
	db, err := database.InitDB(dbPath)
	require.NoError(t, err)

	user := models.User{Username: "realdbtester", Password: "password123!A", Email: "realdb@example.com"}
	require.NoError(t, db.Create(&user).Error)
	alice := models.Contact{UserID: user.ID, Firstname: "Alice"}
	bob := models.Contact{UserID: user.ID, Firstname: "Bob"}
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&bob).Error)

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Next()
	})
	router.POST("/relationship-edges", withValidated(func() any { return &models.RelationshipEdgeInput{} }), CreateRelationshipEdge)
	router.GET("/relationship-edges", ListRelationshipEdges)
	router.PUT("/relationship-edges/:id", withValidated(func() any { return &models.RelationshipEdgeInput{} }), UpdateRelationshipEdge)
	router.PATCH("/relationship-edges/:id/accept", AcceptRelationshipEdge)
	router.DELETE("/relationship-edges/:id", DeleteRelationshipEdge)
	router.GET("/graph", GetGraph)

	doJSON := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req, _ := http.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// Create: existing-contact endpoints, a registered relation type.
	createResp := doJSON("POST", "/relationship-edges", models.RelationshipEdgeInput{
		SourceID: alice.VCardUID, TargetID: bob.VCardUID, Type: "parent_of",
	})
	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var created struct {
		RelationshipEdge models.RelationshipEdge `json:"relationship_edge"`
	}
	require.NoError(t, json.Unmarshal(createResp.Body.Bytes(), &created))
	edgeID := created.RelationshipEdge.ID
	require.NotEmpty(t, edgeID)
	assert.True(t, created.RelationshipEdge.Directional)

	// Create: thin-contact target.
	thinResp := doJSON("POST", "/relationship-edges", models.RelationshipEdgeInput{
		SourceID: alice.VCardUID, Type: "sibling_of",
		TargetThin: &models.ThinContactInput{Name: "Real DB Sibling"},
	})
	require.Equal(t, http.StatusCreated, thinResp.Code, thinResp.Body.String())
	var thinContact models.Contact
	require.NoError(t, db.Where("user_id = ? AND firstname = ?", user.ID, "Real DB Sibling").First(&thinContact).Error)

	// List, filtered by contact (both directions).
	listResp := doJSON("GET", "/relationship-edges?contact_id="+alice.VCardUID, nil)
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())
	var listed struct {
		RelationshipEdges []models.RelationshipEdge `json:"relationship_edges"`
		Total             int64                     `json:"total"`
	}
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listed))
	assert.EqualValues(t, 2, listed.Total)

	// List, filtered by status (nothing suggested yet).
	suggestedResp := doJSON("GET", "/relationship-edges?status=suggested", nil)
	require.Equal(t, http.StatusOK, suggestedResp.Code)
	var suggestedListed struct {
		RelationshipEdges []models.RelationshipEdge `json:"relationship_edges"`
	}
	require.NoError(t, json.Unmarshal(suggestedResp.Body.Bytes(), &suggestedListed))
	assert.Empty(t, suggestedListed.RelationshipEdges)

	// Update.
	updateResp := doJSON("PUT", "/relationship-edges/"+edgeID, models.RelationshipEdgeInput{
		SourceID: alice.VCardUID, TargetID: bob.VCardUID, Type: "roommate_of", Sensitivity: "private",
	})
	require.Equal(t, http.StatusOK, updateResp.Code, updateResp.Body.String())
	var updatedEdge models.RelationshipEdge
	require.NoError(t, db.First(&updatedEdge, "id = ?", edgeID).Error)
	assert.Equal(t, "roommate_of", updatedEdge.Type)
	assert.Equal(t, "private", updatedEdge.Sensitivity)

	// Seed a household-inferred suggestion directly (mirrors
	// household_service.go's shape), then accept it through the route.
	suggestion := models.RelationshipEdge{
		UserID: user.ID, SourceID: alice.VCardUID, TargetID: thinContact.VCardUID, Type: "gets_along_with",
		Source: models.RelationshipSourceHouseholdInferred, Confidence: 0.6, Status: models.RelationshipStatusSuggested,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&suggestion).Error)

	// GET /graph before acceptance: the suggestion must not appear as an
	// edge, but both confirmed edges (parent_of promoted to roommate_of, and
	// sibling_of) must.
	graphResp := doJSON("GET", "/graph", nil)
	require.Equal(t, http.StatusOK, graphResp.Code, graphResp.Body.String())
	var graph models.GraphResponse
	require.NoError(t, json.Unmarshal(graphResp.Body.Bytes(), &graph))
	assert.Len(t, graph.Nodes, 3, "alice, bob, and the thin sibling contact")
	assert.Len(t, graph.Edges, 2, "the two confirmed edges, not the pending suggestion")
	for _, e := range graph.Edges {
		assert.NotEqual(t, "gets_along_with", e.Label, "suggested edge must not appear in the graph")
	}

	acceptResp := doJSON("PATCH", "/relationship-edges/"+suggestion.ID+"/accept", nil)
	require.Equal(t, http.StatusOK, acceptResp.Code, acceptResp.Body.String())
	var accepted models.RelationshipEdge
	require.NoError(t, db.First(&accepted, "id = ?", suggestion.ID).Error)
	assert.Equal(t, models.RelationshipStatusConfirmed, accepted.Status)
	assert.Equal(t, models.RelationshipSourceHouseholdInferred, accepted.Source, "provenance preserved through accept")

	// GET /graph after acceptance: now 3 edges.
	graphResp2 := doJSON("GET", "/graph", nil)
	require.Equal(t, http.StatusOK, graphResp2.Code)
	var graph2 models.GraphResponse
	require.NoError(t, json.Unmarshal(graphResp2.Body.Bytes(), &graph2))
	assert.Len(t, graph2.Edges, 3, "the newly-accepted edge now appears too")

	// Delete.
	deleteResp := doJSON("DELETE", "/relationship-edges/"+edgeID, nil)
	require.Equal(t, http.StatusOK, deleteResp.Code, deleteResp.Body.String())
	var countAfterDelete int64
	require.NoError(t, db.Model(&models.RelationshipEdge{}).Where("id = ?", edgeID).Count(&countAfterDelete).Error)
	assert.EqualValues(t, 0, countAfterDelete)
}
