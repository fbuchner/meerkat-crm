package main

import (
	"fmt"
	"log"
	"perema/config"
	"perema/models"
	"perema/routes"
	"perema/services"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
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

	log.Println("Loading migrations...")
	if err := db.AutoMigrate(&models.Contact{}, &models.Activity{}, &models.Note{}, models.Relationship{}, models.Reminder{}, models.User{}); err != nil {
		log.Fatalf("failed to migrate database schema: %v", err)
	}

	log.Println("Running scheduler...")
	// Schedule the reminder task daily
	if !cfg.UseSendgrid {
		log.Printf("WARN: No Mails to be sent since Sendgrid configuration is not set")
	}
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At(cfg.ReminderTime).Do(func() {
		if err := services.SendBirthdayReminders(db); err != nil {
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
