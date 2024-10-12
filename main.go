package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"perema/controllers"
	"perema/models"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/golang-jwt/jwt/v4"
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
	db.AutoMigrate(&models.Activity{}, &models.Contact{}, &models.Note{})

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

	authorized := r.Group("/")
	r.Static("/static", "./frontend/dist")

	//TODO: check why secret is not being required
	authorized.Use(AuthMiddleware())
	{
		// Contact routes
		r.GET("/contacts", controllers.GetAllContacts)
		r.POST("/contacts", controllers.CreateContact)
		r.POST("/contacts/circles", controllers.CreateContact)
		r.GET("/contacts/:id", controllers.GetContact)
		r.PUT("/contacts/:id", controllers.UpdateContact)
		r.DELETE("/contacts/:id", controllers.DeleteContact)
		r.POST("/contacts/:id/relationships", controllers.AddRelationshipToContact)
		r.POST("/contacts/:id/profile_picture", controllers.AddProfilePictureToContact)

		// Note routes
		r.GET("/contacts/:id/notes", controllers.GetNotesForContact)
		r.POST("/contacts/:id/notes", controllers.CreateNote)
		r.GET("/notes/:id", controllers.GetNote)
		r.PUT("/notes/:id", controllers.UpdateNote)
		r.DELETE("/notes/:id", controllers.DeleteNote)

		// Activity routes
		r.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
		r.POST("/activities", controllers.CreateActivity)
		r.GET("/activities/:id", controllers.GetActivity)
		r.PUT("/activities/:id", controllers.UpdateActivity)
		r.DELETE("/activities/:id", controllers.DeleteActivity)
		r.POST("/activities/:id/contacts/:contact_id", controllers.AddContactToActivity)
		r.DELETE("/activities/:id/contacts/:contact_id", controllers.RemoveContactFromActivity)
	}

	jwt_debug, err := GenerateDebugJWT()
	if err != nil {
		panic("Unable to create JWT token based on your secret.")
	}
	log.Println("Use this token for testing: ", jwt_debug)

	log.Println("Server listening on Port 8080...")

	//TODO setup https
	// Start HTTP server to redirect to HTTPS
	//go func() {
	//	if err := http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
	//	})); err != nil {
	//		log.Fatalf("Failed to start HTTP server: %s", err)
	//	}
	//}()

	// Listen and serve on HTTPS
	//err = r.RunTLS(":8443", "./cert.pem", "./key.pem")
	//if err != nil {
	//	log.Fatalf("Failed to start HTTPS server: %s", err)
	//}

	r.Run(":8080")
}

func sendBirthdayReminders(db *gorm.DB) {
	var contacts []models.Contact
	db.Where("birthday = ?", time.Now().Format("2006-01-02")).Find(&contacts)

	for _, contact := range contacts {
		age := "unknown age"
		zeroTime := time.Time{}

		contactBirthday, validBirthday := contact.Birthday.ToTime()
		if validBirthday && !contactBirthday.Equal(zeroTime) {
			age = fmt.Sprintf("%d years old", time.Now().Year()-contact.Birthday.Time.Year())
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

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(os.Getenv("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateDebugJWT generates a JWT token for debugging purposes
func GenerateDebugJWT() (string, error) {
	claims := jwt.MapClaims{
		"authorized": true,
		"user":       "debug_user",
		"exp":        time.Now().Add(time.Hour * 96).Unix(), // Token expires in 96 hours
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
