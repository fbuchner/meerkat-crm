package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mycorrhizal/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func findGraphNode(nodes []models.GraphNode, id string) *models.GraphNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func findGraphEdge(edges []models.GraphEdge, id string) *models.GraphEdge {
	for i := range edges {
		if edges[i].ID == id {
			return &edges[i]
		}
	}
	return nil
}

func TestGetGraph_EmptyData(t *testing.T) {
	db, router := setupRouter()
	router.GET("/graph", GetGraph)

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.GraphResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Nodes)
	assert.Empty(t, resp.Edges)

	// db is returned so future tests in this file can reuse the seeded user via db.First.
	_ = db
}

func TestGetGraph_ContactsRelationshipsAndActivities(t *testing.T) {
	db, router := setupRouter()
	router.GET("/graph", GetGraph)

	var user models.User
	require.NoError(t, db.First(&user).Error)

	contactA := models.Contact{UserID: user.ID, Firstname: "Alice", Lastname: "Anderson", Circles: []string{"Family"}}
	contactB := models.Contact{UserID: user.ID, Firstname: "Bob", Lastname: "Brown"}
	// Archived contact must be excluded entirely.
	contactC := models.Contact{UserID: user.ID, Firstname: "Carol", Lastname: "Carter", Archived: true}
	require.NoError(t, db.Create(&contactA).Error)
	require.NoError(t, db.Create(&contactB).Error)
	require.NoError(t, db.Create(&contactC).Error)

	// A "friend_of" relationship edge linking B -> A.
	edge := models.RelationshipEdge{
		UserID:      user.ID,
		SourceID:    contactB.VCardUID,
		TargetID:    contactA.VCardUID,
		Type:        "friend_of",
		Source:      models.RelationshipSourceUserConfirmed,
		Confidence:  1.0,
		Status:      models.RelationshipStatusConfirmed,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&edge).Error)

	// An activity with 2+ contacts must produce an activity node + star edges.
	multiActivity := models.Activity{UserID: user.ID, Title: "Team Meeting", Date: time.Now()}
	require.NoError(t, db.Create(&multiActivity).Error)
	require.NoError(t, db.Model(&multiActivity).Association("Contacts").Append(&contactA, &contactB))

	// An activity with only 1 contact must NOT produce a node or edges.
	soloActivity := models.Activity{UserID: user.ID, Title: "Solo Errand", Date: time.Now()}
	require.NoError(t, db.Create(&soloActivity).Error)
	require.NoError(t, db.Model(&soloActivity).Association("Contacts").Append(&contactA))

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.GraphResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// Nodes: contactA, contactB, and the multi-contact activity. contactC
	// (archived) and soloActivity (only 1 contact) must be absent.
	assert.Len(t, resp.Nodes, 3)

	nodeA := findGraphNode(resp.Nodes, "c-1")
	require.NotNil(t, nodeA)
	assert.Equal(t, "contact", nodeA.Type)
	assert.Equal(t, "Alice Anderson", nodeA.Label)
	assert.Equal(t, []string{"Family"}, nodeA.Circles)

	nodeB := findGraphNode(resp.Nodes, "c-2")
	require.NotNil(t, nodeB)
	assert.Equal(t, "Bob Brown", nodeB.Label)

	assert.Nil(t, findGraphNode(resp.Nodes, "c-3"), "archived contact must be excluded")

	activityNodeID := "a-1"
	activityNode := findGraphNode(resp.Nodes, activityNodeID)
	require.NotNil(t, activityNode, "multi-contact activity must produce a node")
	assert.Equal(t, "activity", activityNode.Type)
	assert.Equal(t, "Team Meeting", activityNode.Label)

	assert.Nil(t, findGraphNode(resp.Nodes, "a-2"), "single-contact activity must not produce a node")

	// Edges: 1 relationship edge + 2 activity-star edges.
	assert.Len(t, resp.Edges, 3)

	relEdge := findGraphEdge(resp.Edges, edge.ID)
	require.NotNil(t, relEdge)
	assert.Equal(t, "c-2", relEdge.Source)
	assert.Equal(t, "c-1", relEdge.Target)
	assert.Equal(t, "relationship", relEdge.Type)
	assert.Equal(t, "friend_of", relEdge.Label)

	edgeToA := findGraphEdge(resp.Edges, "ae-1-1")
	require.NotNil(t, edgeToA)
	assert.Equal(t, activityNodeID, edgeToA.Source)
	assert.Equal(t, "c-1", edgeToA.Target)
	assert.Equal(t, "activity", edgeToA.Type)

	edgeToB := findGraphEdge(resp.Edges, "ae-1-2")
	require.NotNil(t, edgeToB)
	assert.Equal(t, activityNodeID, edgeToB.Source)
	assert.Equal(t, "c-2", edgeToB.Target)
}

// TestGetGraph_ExcludesSuggestedRelationshipEdges pins down the WP2 behavior
// change: a "suggested" RelationshipEdge (e.g. household-inferred, awaiting
// user review) must never appear in the graph -- only "confirmed" edges are
// graphed, per RelationshipEdge.Status's own doc comment.
func TestGetGraph_ExcludesSuggestedRelationshipEdges(t *testing.T) {
	db, router := setupRouter()
	router.GET("/graph", GetGraph)

	var user models.User
	require.NoError(t, db.First(&user).Error)

	contactA := models.Contact{UserID: user.ID, Firstname: "Alice"}
	contactB := models.Contact{UserID: user.ID, Firstname: "Bob"}
	require.NoError(t, db.Create(&contactA).Error)
	require.NoError(t, db.Create(&contactB).Error)

	suggested := models.RelationshipEdge{
		UserID:      user.ID,
		SourceID:    contactA.VCardUID,
		TargetID:    contactB.VCardUID,
		Type:        "roommate_of",
		Source:      models.RelationshipSourceHouseholdInferred,
		Confidence:  0.7,
		Status:      models.RelationshipStatusSuggested,
		Sensitivity: models.RelationshipSensitivityNormal,
	}
	require.NoError(t, db.Create(&suggested).Error)

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.GraphResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Nodes, 2, "both contacts still appear as nodes")
	assert.Empty(t, resp.Edges, "a suggested edge must not appear in the graph")
}

func TestGetGraph_BlankNameFallsBackToUnknown(t *testing.T) {
	db, router := setupRouter()
	router.GET("/graph", GetGraph)

	var user models.User
	require.NoError(t, db.First(&user).Error)

	contact := models.Contact{UserID: user.ID, Firstname: "", Lastname: ""}
	require.NoError(t, db.Create(&contact).Error)

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.GraphResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Nodes, 1)
	assert.Equal(t, "Unknown", resp.Nodes[0].Label)
}

func TestGetGraph_OnlyReturnsCallingUsersData(t *testing.T) {
	db, router := setupRouter()
	router.GET("/graph", GetGraph)

	// A contact belonging to a different user must never appear.
	otherUser := models.User{Username: "other", Password: "password123", Email: "other@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)
	otherContact := models.Contact{UserID: otherUser.ID, Firstname: "Not", Lastname: "Mine"}
	require.NoError(t, db.Create(&otherContact).Error)

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.GraphResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.Nodes)
}

func TestGetGraph_Unauthorized(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Contact{}, &models.RelationshipEdge{}, &models.Activity{}))

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		// Deliberately no "userID" set, simulating a request that reached
		// this handler without passing through auth middleware.
		c.Next()
	})
	router.GET("/graph", GetGraph)

	req, _ := http.NewRequest("GET", "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
