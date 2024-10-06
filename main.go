package main

import (
	"perema/models"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"gorm.io/driver/sqlite" // or use the appropriate driver
	"gorm.io/gorm"
)

func main() {
	s := gocron.NewScheduler(time.UTC)

	db, err := gorm.Open(sqlite.Open("contacts.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Schedule the birthday reminder task daily
	s.Every(1).Day().At("08:00").Do(sendBirthdayReminders, db)

	// Start the scheduler
	s.StartBlocking()

	// Migrate the schema
	db.AutoMigrate(&models.Contact{})

	r := gin.Default()

	// Add routes here

	r.Run() // listen and serve on 0.0.0.0:8080

}

func sendBirthdayReminders(db *gorm.DB) {
	var contacts []models.Contact
	db.Where("birthday = ?", time.Now().Format("2006-01-02")).Find(&contacts)

	for _, contact := range contacts {
		sendEmail(contact.Email, "Happy Birthday!", "Happy birthday, "+contact.Name+"!")
	}
}

// We are using Twillio Sendgrid to send e-mails. Thef free tier allows for up to 100 mails per day.
func sendEmail(to, subject, body string) {

}
