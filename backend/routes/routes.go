package routes

import (
	"perema/config"
	"perema/controllers"
	"perema/middleware"
	"perema/models"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, cfg *config.Config) {

	// Health check endpoint (no versioning, standard practice)
	router.GET("/health", controllers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required, strict rate limiting)
		v1.POST("/register", middleware.AuthRateLimitMiddleware(), middleware.ValidateJSONMiddleware(&models.User{}), controllers.RegisterUser)
		v1.POST("/login", middleware.AuthRateLimitMiddleware(), func(c *gin.Context) {
			controllers.LoginUser(c, cfg)
		})
		v1.POST("/check-password-strength", middleware.AuthRateLimitMiddleware(), controllers.CheckPasswordStrength)

		// Protected routes (authentication required, general rate limiting)
		protected := v1.Group("/")
		protected.Use(middleware.APIRateLimitMiddleware())
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// Contact routes
			protected.GET("/contacts", controllers.GetContacts)
			protected.POST("/contacts", middleware.ValidateJSONMiddleware(&models.Contact{}), controllers.CreateContact)
			protected.GET("/contacts/:id", controllers.GetContact)
			protected.PUT("/contacts/:id", middleware.ValidateJSONMiddleware(&models.Contact{}), controllers.UpdateContact)
			protected.DELETE("/contacts/:id", controllers.DeleteContact)
			protected.GET("/contacts/circles", controllers.GetCircles)

			// Relationship routes
			protected.GET("/contacts/:id/relationships", controllers.GetRelationships)
			protected.POST("/contacts/:id/relationships", middleware.ValidateJSONMiddleware(&models.Relationship{}), controllers.CreateRelationship)
			protected.PUT("/contacts/:id/relationships/:rid", middleware.ValidateJSONMiddleware(&models.Relationship{}), controllers.UpdateRelationship)
			protected.DELETE("/contacts/:id/relationships/:rid", controllers.DeleteRelationship)

			// Profile picture routes
			protected.POST("/contacts/:id/profile_picture", controllers.AddPhotoToContact)
			protected.GET("/contacts/:id/profile_picture", controllers.GetProfilePicture)

			// Note routes
			protected.GET("/contacts/:id/notes", controllers.GetNotesForContact)
			protected.POST("/contacts/:id/notes", middleware.ValidateJSONMiddleware(&models.Note{}), controllers.CreateNote)
			protected.GET("/notes/:id", controllers.GetNote)
			protected.GET("/notes", controllers.GetUnassignedNotes)
			protected.POST("/notes", middleware.ValidateJSONMiddleware(&models.Note{}), controllers.CreateUnassignedNote)
			protected.PUT("/notes/:id", middleware.ValidateJSONMiddleware(&models.Note{}), controllers.UpdateNote)
			protected.DELETE("/notes/:id", controllers.DeleteNote)

			// Activity routes
			protected.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
			protected.POST("/activities", middleware.ValidateJSONMiddleware(&models.Activity{}), controllers.CreateActivity)
			protected.GET("/activities", controllers.GetActivities)
			protected.GET("/activities/:id", controllers.GetActivity)
			protected.PUT("/activities/:id", middleware.ValidateJSONMiddleware(&models.Activity{}), controllers.UpdateActivity)
			protected.DELETE("/activities/:id", controllers.DeleteActivity)

			// Reminder routes
			protected.GET("/contacts/:id/reminders", controllers.GetRemindersForContact)
			protected.POST("/contacts/:id/reminders", middleware.ValidateJSONMiddleware(&models.Reminder{}), controllers.CreateReminder)
			protected.GET("/reminders/:id", controllers.GetReminder)
			protected.PUT("/reminders/:id", middleware.ValidateJSONMiddleware(&models.Reminder{}), controllers.UpdateReminder)
			protected.DELETE("/reminders/:id", controllers.DeleteReminder)
		}
	}
}
