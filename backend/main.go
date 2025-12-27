package main

import (
	"context"
	"fmt"
	"meerkat/config"
	"meerkat/database"
	apperrors "meerkat/errors"
	"meerkat/logger"
	"meerkat/middleware"
	"meerkat/routes"
	"meerkat/services"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
)

func main() {
	// Initialize logger first
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	isPretty := os.Getenv("LOG_PRETTY")
	prettyLog := isPretty == "true" || isPretty == "1"

	// In development, use pretty logs by default
	if os.Getenv("GIN_MODE") != "release" {
		prettyLog = true
	}

	logger.InitLogger(logger.Config{
		Level:  logLevel,
		Pretty: prettyLog,
	})

	logger.Info().Msg("Loading server...")

	logger.Info().Msg("Loading configuration...")
	cfg := config.LoadConfig()

	logger.Info().Msg("Validating configuration...")
	cfg.ValidateOrPanic()

	logger.Info().Msg("Loading database and running migrations...")
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	logger.Info().Msg("Running scheduler...")
	// Schedule the reminder task daily
	if !cfg.UseResend {
		logger.Warn().Msg("No Mails to be sent since Resend configuration is not set")
	}
	s := gocron.NewScheduler(time.UTC)
	task := func() {
		// Use rate-limited version to prevent duplicate emails during rapid restarts
		if err := services.SendRemindersWithRateLimit(db, *cfg); err != nil {
			logger.Error().Err(err).Msg("Error sending reminders")
		}
	}
	s.Every(1).Day().At(cfg.ReminderTime).Do(task)
	go task() // Run initially once on startup (rate-limited to prevent duplicates)
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

	// Add logging middleware (after request ID)
	r.Use(middleware.LoggingMiddleware())

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

	logger.Info().
		Str("port", cfg.Port).
		Int("read_timeout", cfg.ReadTimeout).
		Int("write_timeout", cfg.WriteTimeout).
		Int("idle_timeout", cfg.IdleTimeout).
		Msg("Starting server")

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to run server")
		}
	}()

	logger.Info().Msg("Server is ready to handle requests")

	// Block until we receive a shutdown signal
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Stop the scheduler first to prevent new jobs from starting
	logger.Info().Msg("Stopping scheduler...")
	s.Stop()

	// Create a deadline to wait for active requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown of HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Close database connection
	logger.Info().Msg("Closing database connection...")
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			logger.Error().Err(err).Msg("Error closing database connection")
		}
	}

	logger.Info().Msg("Server exited gracefully")
}
