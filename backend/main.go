package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"perema/config"
	"perema/models"
	"perema/routes"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	log.Println("Loading server...")

	log.Println("Loading configuration...")
	cfg := config.LoadConfig()

	log.Println("Loading database...")
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database")
	}
	// Migrate the schema
	log.Println("Loading migrations...")
	if err := db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Relationship{}, models.Reminder{}); err != nil {
		log.Fatalf("failed to migrate database schema: %v", err)
	}

	log.Println("Running scheduler...")
	// Schedule the reminder task daily
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At(cfg.ReminderTime).Do(func() {
		if err := sendBirthdayReminders(db); err != nil {
			log.Printf("Error sending birthday reminders: %v", err)
		}
	})
	go s.StartBlocking()

	r := gin.Default()

	// Enable CORS for all origins, methods, and headers
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.SetTrustedProxies(cfg.TrustedProxies)

	// Inject db into context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Register all routes from routes.go
	routes.RegisterRoutes(r, cfg)

	log.Printf("Starting server on Port :%s...", cfg.Port)
	if err := r.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func sendBirthdayReminders(db *gorm.DB) error {
	var contacts []models.Contact
	if err := db.Where("birthday = ?", time.Now().Format("2006-01-02")).Find(&contacts).Error; err != nil {
		return fmt.Errorf("failed to query contacts: %w", err)
	}

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

		if err := sendBirthdayMail(nickname, contact.Firstname+" "+contact.Lastname, age); err != nil {
			return fmt.Errorf("failed to send email for %s: %w", contact.Firstname, err)
		}
	}
	return nil
}

// We are using Twillio Sendgrid to send e-mails. The free tier allows for up to 100 mails per day.
func sendBirthdayMail(birthday_person_nick, birthday_person, birthday_age string) error {
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
		return err
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

	return nil
}

// TODO: use this middleware to protect routes
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
