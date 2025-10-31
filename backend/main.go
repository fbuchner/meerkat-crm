package main

import (
	"fmt"
	"log"
	"net/http"
	"perema/config"
	apperrors "perema/errors"
	"perema/middleware"
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

	log.Println("Validating configuration...")
	cfg.ValidateOrPanic()

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
	task := func() {
		if err := services.SendReminders(db, *cfg); err != nil {
			log.Printf("Error sending birthday reminders: %v", err)
		}
	}
	s.Every(1).Day().At(cfg.ReminderTime).Do(task)
	go task() // Run initially once on startup
	go s.StartBlocking()

	r := gin.Default()

	// CORS configuration with preflight caching
	// MaxAge: Browsers cache preflight OPTIONS requests for 12 hours
	// This reduces redundant OPTIONS requests and improves performance
	//
	// Note: When AllowCredentials is true, AllowOrigins cannot be "*"
	// This is a security restriction in the CORS specification.
	// For development, we allow any origin dynamically via AllowOriginFunc.
	// For production, set FRONTEND_URL to specific origin(s) like "https://yourdomain.com"
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // Cache preflight for 12 hours
	}

	// Handle wildcard "*" for development: allow any origin
	if cfg.FrontendURL == "*" {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true // Allow all origins in development
		}
	} else {
		// Production: allow specific origin(s)
		corsConfig.AllowOrigins = []string{cfg.FrontendURL}
	}

	r.Use(cors.New(corsConfig))

	// Add request ID middleware for tracing
	r.Use(middleware.RequestIDMiddleware())

	// Add error handling middleware
	r.Use(apperrors.ErrorHandlerMiddleware())

	r.SetTrustedProxies(cfg.TrustedProxies)

	// Inject db into context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Register all routes from routes.go
	routes.RegisterRoutes(r, cfg)

	// Create HTTP server with timeout configuration
	// ReadTimeout: Maximum duration for reading the entire request (including body)
	// WriteTimeout: Maximum duration before timing out writes of the response
	// IdleTimeout: Maximum time to wait for the next request when keep-alives are enabled
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	}

	log.Printf("Starting server on Port :%s...", cfg.Port)
	log.Printf("HTTP Timeouts - Read: %ds, Write: %ds, Idle: %ds", cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
