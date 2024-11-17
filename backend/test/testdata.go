package test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"perema/controllers"
	"perema/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InjectTestData(db *gorm.DB) {

	contacts := []models.Contact{
		{
			Firstname: "Jeff",
			Lastname:  "Winger",
			Gender:    "Male",
			Email:     "jeff.winger@greendale.edu",
			Phone:     "123-456-7890",
			Nickname:  "The Winger",
			Birthday:  &models.Date{Time: time.Date(1975, time.November, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		{
			Firstname: "Britta",
			Lastname:  "Perry",
			Gender:    "Female",
			Email:     "britta.perry@greendale.edu",
			Phone:     "123-456-7891",
			Nickname:  "Buzzkill",
		},
		{
			Firstname: "Abed",
			Lastname:  "Nadir",
			Gender:    "Male",
			Email:     "abed.nadir@greendale.edu",
			Phone:     "123-456-7892",
			Nickname:  "Batman",
		},
		{
			Firstname: "Troy",
			Lastname:  "Barnes",
			Gender:    "Male",
			Email:     "troy.barnes@greendale.edu",
			Phone:     "123-456-7893",
			Nickname:  "T-Bone",
		},
		{
			Firstname: "Annie",
			Lastname:  "Edison",
			Gender:    "Female",
			Email:     "annie.edison@greendale.edu",
			Phone:     "123-456-7894",
			Nickname:  "Annie Adderall",
			Birthday:  &models.Date{Time: time.Date(1990, time.December, 19, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		{
			Firstname: "Shirley",
			Lastname:  "Bennett",
			Gender:    "Female",
			Email:     "shirley.bennett@greendale.edu",
			Phone:     "123-456-7895",
			Nickname:  "The Baker",
		},
		{
			Firstname: "Pierce",
			Lastname:  "Hawthorne",
			Gender:    "Male",
			Email:     "pierce.hawthorne@greendale.edu",
			Phone:     "123-456-7896",
			Nickname:  "The Magnificent",
			Birthday:  &models.Date{Time: time.Date(0001, time.June, 20, 0, 0, 0, 0, time.UTC), Valid: false},
		},
	}

	// Loop through the test data and call the controller method
	for _, contact := range contacts {
		// Convert contact to JSON
		jsonData, _ := json.Marshal(contact)

		// Create a new HTTP request using httptest
		req, _ := http.NewRequest("POST", "/contacts", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// Create a mock Gin context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("db", db) // Inject the database instance

		// Call the CreateContact controller
		controllers.CreateContact(c)

		// Check for any errors
		if w.Code != http.StatusOK {
			log.Printf("failed to create contact: %s", w.Body.String())
		}
	}

}
