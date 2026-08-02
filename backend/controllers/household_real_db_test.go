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

// TestHouseholdSuggestions_RealMigratedSchema is the real-DB check for T1
// (docs/fork-plan/tickets/09-T1-households.md): every other controller test
// in this package uses AutoMigrate against :memory: sqlite, which derives
// its schema from the same Go struct tags the application code uses — it
// cannot catch a GORM column-tag mismatch against the real migration SQL
// (this fork's own recurring bug class, e.g. ContactSyncLink.ETag, and the
// ticket's explicit trap about HouseholdMember.MemberVCardUID). This test
// runs the household trigger routes against a database.InitDB-migrated real
// file database, creating a household with mixed roles plus a pet, asserting
// the expected suggested edges exist, re-running idempotently, then closing
// the review loop: accept one suggestion through the real route and confirm
// it becomes `confirmed` (and appears in the graph), reject another by
// deleting it.
func TestHouseholdSuggestions_RealMigratedSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "household-real.db")
	db, err := database.InitDB(dbPath)
	require.NoError(t, err)

	user := models.User{Username: "realdbtester", Password: "password123!A", Email: "realdb@example.com"}
	require.NoError(t, db.Create(&user).Error)

	adult1 := createHouseholdTestContact(t, db, user.ID, "Alice", "")
	adult2 := createHouseholdTestContact(t, db, user.ID, "Bob", "")
	child := createHouseholdTestContact(t, db, user.ID, "Charlie", "")
	pet := createHouseholdTestContact(t, db, user.ID, "Fluffy", "pet")

	household := models.Household{UserID: user.ID, Name: "Smith Family", Type: models.HouseholdTypeFamilyUnit}
	require.NoError(t, db.Create(&household).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: adult1.VCardUID, Role: models.HouseholdRoleHead}).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: adult2.VCardUID, Role: models.HouseholdRoleHead}).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: child.VCardUID, Role: models.HouseholdRoleChild}).Error)
	require.NoError(t, db.Create(&models.HouseholdMember{HouseholdID: household.ID, UserID: user.ID, MemberVCardUID: pet.VCardUID, Role: models.HouseholdRolePet}).Error)

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Next()
	})
	router.POST("/households/:id/suggest-relationships", SuggestHouseholdRelationships)
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

	edgeTypeCounts := func() map[string]int {
		var edges []models.RelationshipEdge
		require.NoError(t, db.Where("user_id = ?", user.ID).Find(&edges).Error)
		counts := map[string]int{}
		for _, e := range edges {
			counts[e.Type]++
		}
		return counts
	}

	// First run: 2 adults + 1 child + 1 pet -> 1 spouse_of + 2 parent_of +
	// 3 owned_by (the child counts as a human for owned_by, per §91.4).
	first := doJSON("POST", "/households/"+household.ID+"/suggest-relationships", nil)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	var firstBody struct {
		Message string                    `json:"message"`
		Total   int                       `json:"total"`
		Edges   []models.RelationshipEdge `json:"suggested_edges"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstBody))
	assert.Equal(t, 6, firstBody.Total, "1 spouse_of + 2 parent_of + 3 owned_by")
	assert.Len(t, firstBody.Edges, 6)

	counts := edgeTypeCounts()
	assert.Equal(t, 1, counts["spouse_of"])
	assert.Equal(t, 2, counts["parent_of"])
	assert.Equal(t, 3, counts["owned_by"])

	// Every generated edge is a pending suggestion with household provenance.
	var petEdges []models.RelationshipEdge
	require.NoError(t, db.Where("type = ?", "owned_by").Find(&petEdges).Error)
	for _, e := range petEdges {
		assert.Equal(t, pet.VCardUID, e.SourceID, "the pet must be the SOURCE of each owned_by edge")
		assert.Equal(t, models.RelationshipStatusSuggested, e.Status)
		assert.Equal(t, models.RelationshipSourceHouseholdInferred, e.Source)
	}

	// Idempotency: a second run must not duplicate any edge.
	second := doJSON("POST", "/households/"+household.ID+"/suggest-relationships", nil)
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	var secondBody struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondBody))
	assert.Equal(t, 0, secondBody.Total, "re-running the trigger must not duplicate edges")

	var totalEdges int64
	require.NoError(t, db.Model(&models.RelationshipEdge{}).Where("user_id = ?", user.ID).Count(&totalEdges).Error)
	assert.EqualValues(t, 6, totalEdges)

	// Close the review loop: pick a spouse_of suggestion, reject another edge
	// via DELETE, then accept the spouse_of through the real route and confirm
	// it flips to confirmed and shows up in the graph.
	var spouseEdge models.RelationshipEdge
	require.NoError(t, db.Where("type = ?", "spouse_of").First(&spouseEdge).Error)
	var rejectEdge models.RelationshipEdge
	require.NoError(t, db.Where("type = ?", "parent_of").First(&rejectEdge).Error)

	graphBefore := doJSON("GET", "/graph", nil)
	require.Equal(t, http.StatusOK, graphBefore.Code)
	var graphBeforeBody models.GraphResponse
	require.NoError(t, json.Unmarshal(graphBefore.Body.Bytes(), &graphBeforeBody))
	assert.Len(t, graphBeforeBody.Edges, 0, "no suggested edge may appear in the graph")

	rejectResp := doJSON("DELETE", "/relationship-edges/"+rejectEdge.ID, nil)
	require.Equal(t, http.StatusOK, rejectResp.Code, rejectResp.Body.String())
	var rejectCount int64
	require.NoError(t, db.Model(&models.RelationshipEdge{}).Where("id = ?", rejectEdge.ID).Count(&rejectCount).Error)
	assert.Zero(t, rejectCount, "rejecting a suggestion deletes the edge")

	acceptResp := doJSON("PATCH", "/relationship-edges/"+spouseEdge.ID+"/accept", nil)
	require.Equal(t, http.StatusOK, acceptResp.Code, acceptResp.Body.String())
	var accepted models.RelationshipEdge
	require.NoError(t, db.First(&accepted, "id = ?", spouseEdge.ID).Error)
	assert.Equal(t, models.RelationshipStatusConfirmed, accepted.Status)
	assert.Equal(t, models.RelationshipSourceHouseholdInferred, accepted.Source, "provenance preserved through accept")

	graphAfter := doJSON("GET", "/graph", nil)
	require.Equal(t, http.StatusOK, graphAfter.Code)
	var graphAfterBody models.GraphResponse
	require.NoError(t, json.Unmarshal(graphAfter.Body.Bytes(), &graphAfterBody))
	assert.Len(t, graphAfterBody.Edges, 1, "the accepted edge now appears in the graph")
}
