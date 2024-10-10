package main

import (
	"fmt"
	"log"
	"os"
	"perema/controllers"
	"perema/models"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gorm.io/driver/sqlite" // or use the appropriate driver
	"gorm.io/gorm"
)

func main() {
	log.Println("Loading server...")

	s := gocron.NewScheduler(time.UTC)

	log.Println("Loading database...")

	// Open a connection to the SQLite database
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "perema.db" // Default path if environment variable is not set
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	log.Println("Loading migrations...")
	db.AutoMigrate(&models.Activity{}, &models.Contact{}, &models.Note{}, &models.Circle{})

	log.Println("Running scheduler...")
	// Schedule the birthday reminder task daily
	s.Every(1).Day().At("08:00").Do(sendBirthdayReminders, db)
	// Start the scheduler in a separate goroutine
	go s.StartBlocking()

	r := gin.Default()

	// Enable CORS for all origins, methods, and headers
	r.Use(cors.Default()) // Add CORS middleware

	// Inject db into context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Add routes here
	r.Static("/static", "./frontend/dist")

	r.GET("/contacts", controllers.GetAllContacts)
	r.POST("/contacts", controllers.CreateContact)
	r.GET("/contacts/:id", controllers.GetContact)
	r.PUT("/contacts/:id", controllers.UpdateContact)
	r.DELETE("/contacts/:id", controllers.DeleteContact)

	log.Println("Server listening on Port 8080...")
	r.Run() // listen and serve on 0.0.0.0:8080

}

func sendBirthdayReminders(db *gorm.DB) {
	var contacts []models.Contact
	db.Where("birthday = ?", time.Now().Format("2006-01-02")).Find(&contacts)

	for _, contact := range contacts {
		age := "unknown age"
		zeroTime := time.Time{}

		contactBirthday := contact.Birthday.ToTime().Format("2006")
		if contactBirthday != zeroTime.Format("2006") {
			age = fmt.Sprintf("%d years old", time.Now().Year()-contact.Birthday.ToTime().Year())
		}

		nickname := contact.Nickname
		if nickname == "" {
			nickname = contact.Firstname
		}
		sendBirthdayMail(nickname, contact.Firstname+" "+contact.Lastname, age)
	}
}

// We are using Twillio Sendgrid to send e-mails. The free tier allows for up to 100 mails per day.
func sendBirthdayMail(birthday_person_nick, birthday_person, birthday_age string) {
	toEmail := mail.NewEmail("", os.Getenv("SENDGRID_TO_EMAIL"))
	message := mail.NewV3Mail()
	message.SetTemplateID(os.Getenv("SENDGRID_BIRTHDAY_TEMPLATE_ID"))

	personalization := mail.NewPersonalization()
	personalization.AddTos(toEmail)

	personalization.SetDynamicTemplateData("birthday_person_nick", birthday_person_nick)
	personalization.SetDynamicTemplateData("birthday_person", birthday_person)
	personalization.SetDynamicTemplateData("birthday_age", birthday_age)

	message.AddPersonalizations(personalization)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
}
