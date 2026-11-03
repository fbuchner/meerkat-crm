package controllers

import (
	"fmt"
	"meerkat/models"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nodeIDs(graph models.GraphResponse) []string {
	ids := make([]string, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		ids = append(ids, node.ID)
	}
	return ids
}

func TestGetGraph_OnlyReturnsTheUsersData(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)

	ada := models.Contact{UserID: user.ID, Firstname: "Ada", Lastname: "Lovelace", Circles: []string{"friends"}}
	bob := models.Contact{UserID: user.ID, Firstname: "Bob"}
	nameless := models.Contact{UserID: user.ID}
	archived := models.Contact{UserID: user.ID, Firstname: "Old", Archived: true}
	theirs := models.Contact{UserID: stranger.ID, Firstname: "Secret"}
	theirs2 := models.Contact{UserID: stranger.ID, Firstname: "Hidden"}
	for _, contact := range []*models.Contact{&ada, &bob, &nameless, &archived, &theirs, &theirs2} {
		require.NoError(t, db.Create(contact).Error)
	}

	link := models.Relationship{UserID: user.ID, ContactID: ada.ID, RelatedContactID: &bob.ID, Name: "Bob", Type: "Friend"}
	unlinked := models.Relationship{UserID: user.ID, ContactID: ada.ID, Name: "Offline", Type: "Friend"}
	theirLink := models.Relationship{UserID: stranger.ID, ContactID: theirs.ID, RelatedContactID: &theirs2.ID, Name: "x", Type: "Secret"}
	for _, relationship := range []*models.Relationship{&link, &unlinked, &theirLink} {
		require.NoError(t, db.Create(relationship).Error)
	}

	group := models.Activity{UserID: user.ID, Title: "Dinner", Date: time.Now(), Contacts: []models.Contact{ada, bob}}
	solo := models.Activity{UserID: user.ID, Title: "Solo", Date: time.Now(), Contacts: []models.Contact{ada}}
	theirActivity := models.Activity{UserID: stranger.ID, Title: "Their party", Date: time.Now(), Contacts: []models.Contact{theirs, theirs2}}
	for _, activity := range []*models.Activity{&group, &solo, &theirActivity} {
		require.NoError(t, db.Create(activity).Error)
	}

	router := routerForUser(db, user.ID)
	router.GET("/graph", GetGraph)
	w := doRequest(router, http.MethodGet, "/graph", nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.NotContains(t, w.Body.String(), "Secret")
	assert.NotContains(t, w.Body.String(), "Hidden")
	assert.NotContains(t, w.Body.String(), "Their party")

	graph := decodeJSON[models.GraphResponse](t, w)
	assert.ElementsMatch(t, []string{
		fmt.Sprintf("c-%d", ada.ID), fmt.Sprintf("c-%d", bob.ID), fmt.Sprintf("c-%d", nameless.ID),
		fmt.Sprintf("a-%d", group.ID),
	}, nodeIDs(graph), "archived contacts and single-contact activities are not nodes")

	labels := map[string]string{}
	for _, node := range graph.Nodes {
		labels[node.ID] = node.Label
	}
	assert.Equal(t, "Ada Lovelace", labels[fmt.Sprintf("c-%d", ada.ID)])
	assert.Equal(t, "Unknown", labels[fmt.Sprintf("c-%d", nameless.ID)])

	edges := map[string]models.GraphEdge{}
	for _, edge := range graph.Edges {
		edges[edge.ID] = edge
	}
	assert.Len(t, edges, 3, "one relationship edge and two activity edges")
	relationshipEdge := edges[fmt.Sprintf("r-%d", link.ID)]
	assert.Equal(t, fmt.Sprintf("c-%d", ada.ID), relationshipEdge.Source)
	assert.Equal(t, fmt.Sprintf("c-%d", bob.ID), relationshipEdge.Target)
	assert.Equal(t, "Friend", relationshipEdge.Label)
	activityEdge := edges[fmt.Sprintf("ae-%d-%d", group.ID, bob.ID)]
	assert.Equal(t, fmt.Sprintf("a-%d", group.ID), activityEdge.Source)
	assert.Equal(t, fmt.Sprintf("c-%d", bob.ID), activityEdge.Target)
}

func TestGetGraph_ActivityEdgesSkipAnotherUsersContact(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	stranger := newTestUser(t, db, "stranger", false)
	mine := newTestContact(t, db, user.ID, "Mine")
	theirs := newTestContact(t, db, stranger.ID, "Theirs")

	// A corrupt link to a foreign contact must neither leak that contact nor make the activity a group.
	activity := models.Activity{UserID: user.ID, Title: "Mixed", Date: time.Now()}
	require.NoError(t, db.Create(&activity).Error)
	for _, id := range []uint{mine.ID, theirs.ID} {
		require.NoError(t, db.Exec("INSERT INTO activity_contacts (activity_id, contact_id) VALUES (?, ?)", activity.ID, id).Error)
	}

	router := routerForUser(db, user.ID)
	router.GET("/graph", GetGraph)
	w := doRequest(router, http.MethodGet, "/graph", nil, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	graph := decodeJSON[models.GraphResponse](t, w)
	assert.Equal(t, []string{fmt.Sprintf("c-%d", mine.ID)}, nodeIDs(graph))
	assert.Empty(t, graph.Edges)
}

func TestGetGraph_EmptyForAUserWithoutData(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db, "owner", false)
	router := routerForUser(db, user.ID)
	router.GET("/graph", GetGraph)

	w := doRequest(router, http.MethodGet, "/graph", nil, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"nodes":[],"edges":[]}`, w.Body.String(), "empty lists, not null, so the frontend can iterate them")
}
