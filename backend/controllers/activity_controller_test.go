package controllers

import (
	"bytes"
	"encoding/json"
	"meerkat/models"
	"net/http"
	"net/http/httptest"
	"strconv"

	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRouter() (*gorm.DB, *gin.Engine) {
	gin.SetMode(gin.ReleaseMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Relationship{}, models.Reminder{}, models.User{})

	user := models.User{Username: "tester", Password: "password123", Email: "tester@example.com"}
	if err := db.Create(&user).Error; err != nil {
		panic("failed to seed user")
	}

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("userID", user.ID)
		c.Next()
	})

	return db, router
}

func withValidated(factory func() any) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := factory()
		if payload != nil {
			if err := c.ShouldBindJSON(payload); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.Set("validated", payload)
		}
		c.Next()
	}
}

func TestCreateActivity(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.POST("/activities", withValidated(func() any { return &models.ActivityInput{} }), CreateActivity)

	contacts := []models.Contact{
		{
			UserID:    user.ID,
			Firstname: "John",
			Lastname:  "Doe",
		},
		{
			UserID:    user.ID,
			Firstname: "Jane",
			Lastname:  "Smith",
		},
	}

	db.Create(&contacts[0])
	db.Create(&contacts[1])

	activityPayload := models.ActivityInput{
		Title:       "Great activity",
		Description: "A fun get-together.",
		Location:    "Somewhere out there",
		Date:        time.Now().AddDate(0, 0, 1),
		ContactIDs:  []uint{contacts[0].ID, contacts[1].ID},
	}
	jsonValue, _ := json.Marshal(activityPayload)

	req, _ := http.NewRequest("POST", "/activities", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Activity created successfully", responseBody["message"])
}

func TestGetActivitiesForContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/contacts/:id/activities", GetActivitiesForContact)

	// Create a contact
	contact := models.Contact{
		UserID:    user.ID,
		Firstname: "John",
		Lastname:  "Doe",
	}
	db.Create(&contact)

	// Create some activities
	activity1 := models.Activity{
		UserID:      user.ID,
		Title:       "Activity One",
		Description: "First activity",
		Location:    "Location One",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	activity2 := models.Activity{
		UserID:      user.ID,
		Title:       "Activity Two",
		Description: "Second activity",
		Location:    "Location Two",
		Date:        time.Now().AddDate(0, 0, 2),
	}
	db.Create(&activity1)
	db.Create(&activity2)

	// Associate the contact with the activities
	db.Model(&activity1).Association("Contacts").Append(&contact)
	db.Model(&activity2).Association("Contacts").Append(&contact)

	// Make the request to get activities for the contact
	req, _ := http.NewRequest("GET", "/contacts/"+strconv.Itoa(int(contact.ID))+"/activities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["activities"], 2) // Should return both activities for the contact
}

func TestGetActivities(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/activities", GetActivities)

	// Create some activities
	activity1 := models.Activity{
		UserID:      user.ID,
		Title:       "Activity One",
		Description: "First activity",
		Location:    "Location One",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	activity2 := models.Activity{
		UserID:      user.ID,
		Title:       "Activity Two",
		Description: "Second activity",
		Location:    "Location Two",
		Date:        time.Now().AddDate(0, 0, 2),
	}
	db.Create(&activity1)
	db.Create(&activity2)

	// Make the request to get all activities
	req, _ := http.NewRequest("GET", "/activities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Len(t, responseBody["activities"], 2)
	assert.EqualValues(t, 2, responseBody["total"])
	assert.EqualValues(t, 1, responseBody["page"])
}

func TestGetActivitiesSearchByContact(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/activities", GetActivities)

	contact := models.Contact{UserID: user.ID, Firstname: "Search", Lastname: "Target"}
	db.Create(&contact)

	activityWithContact := models.Activity{
		UserID:      user.ID,
		Title:       "Targeted Activity",
		Description: "Includes contact",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	activityWithoutContact := models.Activity{
		UserID:      user.ID,
		Title:       "Other Activity",
		Description: "No matching contact",
		Date:        time.Now().AddDate(0, 0, 2),
	}
	db.Create(&activityWithContact)
	db.Create(&activityWithoutContact)
	db.Model(&activityWithContact).Association("Contacts").Append(&contact)

	req, _ := http.NewRequest("GET", "/activities?search=target", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &responseBody)

	activitiesRaw, ok := responseBody["activities"].([]any)
	if !ok {
		t.Fatalf("expected activities array in response")
	}
	assert.Len(t, activitiesRaw, 1)
	assert.EqualValues(t, 1, responseBody["total"])
}

func TestGetActivity(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.GET("/activities/:id", GetActivity)

	// Create an activity
	activity := models.Activity{
		UserID:      user.ID,
		Title:       "Activity One",
		Description: "First activity",
		Location:    "Location One",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	db.Create(&activity)

	// Make the request to get the activity by ID
	req, _ := http.NewRequest("GET", "/activities/"+strconv.Itoa(int(activity.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Activity
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, activity.Title, responseBody.Title)
}

func TestUpdateActivity(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.PUT("/activities/:id", withValidated(func() any { return &models.ActivityInput{} }), UpdateActivity)

	// Create an activity
	activity := models.Activity{
		UserID:      user.ID,
		Title:       "Activity One",
		Description: "First activity",
		Location:    "Location One",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	db.Create(&activity)

	// Update activity details
	activityUpdate := models.ActivityInput{
		Title:       "Updated Activity",
		Description: "Updated description",
		Location:    "Updated location",
		Date:        time.Now(),
		ContactIDs:  []uint{},
	}
	jsonValue, _ := json.Marshal(activityUpdate)

	// Make the request to update the activity
	req, _ := http.NewRequest("PUT", "/activities/"+strconv.Itoa(int(activity.ID)), bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody models.Activity
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, activityUpdate.Title, responseBody.Title)
}

func TestDeleteActivity(t *testing.T) {
	db, router := setupRouter()

	var user models.User
	db.First(&user)

	router.DELETE("/activities/:id", DeleteActivity)

	// Create an activity
	activity := models.Activity{
		UserID:      user.ID,
		Title:       "Activity One",
		Description: "First activity",
		Location:    "Location One",
		Date:        time.Now().AddDate(0, 0, 1),
	}
	db.Create(&activity)

	// Make the request to delete the activity
	req, _ := http.NewRequest("DELETE", "/activities/"+strconv.Itoa(int(activity.ID)), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responseBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.Equal(t, "Activity deleted", responseBody["message"])

	// Verify activity has been deleted
	var deletedActivity models.Activity
	result := db.First(&deletedActivity, activity.ID)
	assert.True(t, result.Error != nil) // This should return an error as it has been deleted
}
