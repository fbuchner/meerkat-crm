package routes

import (
	"meerkat/config"
	"meerkat/controllers"
	"meerkat/middleware"
	"meerkat/models"

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
		v1.POST("/password-reset/request", middleware.AuthRateLimitMiddleware(), middleware.ValidateJSONMiddleware(&models.PasswordResetRequestInput{}), func(c *gin.Context) {
			controllers.RequestPasswordReset(c, cfg)
		})
		v1.POST("/password-reset/confirm", middleware.AuthRateLimitMiddleware(), middleware.ValidateJSONMiddleware(&models.PasswordResetConfirmInput{}), controllers.ConfirmPasswordReset)

		// Protected routes (authentication required, general rate limiting)
		protected := v1.Group("/")
		protected.Use(middleware.APIRateLimitMiddleware())
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			protected.POST("/users/change-password", middleware.ValidateJSONMiddleware(&models.ChangePasswordInput{}), controllers.ChangePassword)

			// Contact routes
			protected.GET("/contacts", controllers.GetContacts)
			protected.GET("/contacts/circles", controllers.GetCircles)
			protected.GET("/contacts/random", controllers.GetContactsRandom)
			protected.GET("/contacts/birthdays", controllers.GetUpcomingBirthdays)
			protected.POST("/contacts", middleware.ValidateJSONMiddleware(&models.ContactInput{}), controllers.CreateContact)
			protected.GET("/contacts/:id", controllers.GetContact)
			protected.PUT("/contacts/:id", middleware.ValidateJSONMiddleware(&models.ContactInput{}), controllers.UpdateContact)
			protected.DELETE("/contacts/:id", controllers.DeleteContact)

			// Contact import routes
			protected.POST("/contacts/import/upload", controllers.UploadCSVForImport)
			protected.POST("/contacts/import/preview", middleware.ValidateJSONMiddleware(&models.ImportPreviewRequest{}), controllers.PreviewImport)
			protected.POST("/contacts/import/confirm", middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{}), controllers.ConfirmImport)

			// Relationship routes
			protected.GET("/contacts/:id/relationships", controllers.GetRelationships)
			protected.GET("/contacts/:id/incoming-relationships", controllers.GetIncomingRelationships)
			protected.POST("/contacts/:id/relationships", middleware.ValidateJSONMiddleware(&models.RelationshipInput{}), controllers.CreateRelationship)
			protected.PUT("/contacts/:id/relationships/:rid", middleware.ValidateJSONMiddleware(&models.RelationshipInput{}), controllers.UpdateRelationship)
			protected.DELETE("/contacts/:id/relationships/:rid", controllers.DeleteRelationship)

			// Profile picture routes
			protected.POST("/contacts/:id/profile_picture", controllers.AddPhotoToContact)
			protected.GET("/contacts/:id/profile_picture", controllers.GetProfilePicture)

			// Image proxy route (for fetching images from external URLs)
			protected.GET("/proxy/image", controllers.ProxyImage)

			// Note routes
			protected.GET("/contacts/:id/notes", controllers.GetNotesForContact)
			protected.POST("/contacts/:id/notes", middleware.ValidateJSONMiddleware(&models.NoteInput{}), controllers.CreateNote)
			protected.GET("/notes/:id", controllers.GetNote)
			protected.GET("/notes", controllers.GetUnassignedNotes)
			protected.POST("/notes", middleware.ValidateJSONMiddleware(&models.NoteInput{}), controllers.CreateUnassignedNote)
			protected.PUT("/notes/:id", middleware.ValidateJSONMiddleware(&models.NoteInput{}), controllers.UpdateNote)
			protected.DELETE("/notes/:id", controllers.DeleteNote)

			// Activity routes
			protected.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
			protected.POST("/activities", middleware.ValidateJSONMiddleware(&models.ActivityInput{}), controllers.CreateActivity)
			protected.GET("/activities", controllers.GetActivities)
			protected.GET("/activities/:id", controllers.GetActivity)
			protected.PUT("/activities/:id", middleware.ValidateJSONMiddleware(&models.ActivityInput{}), controllers.UpdateActivity)
			protected.DELETE("/activities/:id", controllers.DeleteActivity)

			// Reminder routes
			protected.GET("/reminders", controllers.GetAllReminders)
			protected.GET("/reminders/upcoming", controllers.GetUpcomingReminders)
			protected.GET("/contacts/:id/reminders", controllers.GetRemindersForContact)
			protected.POST("/contacts/:id/reminders", middleware.ValidateJSONMiddleware(&models.Reminder{}), controllers.CreateReminder)
			protected.GET("/reminders/:id", controllers.GetReminder)
			protected.PUT("/reminders/:id", middleware.ValidateJSONMiddleware(&models.Reminder{}), controllers.UpdateReminder)
			protected.POST("/reminders/:id/complete", controllers.CompleteReminder)
			protected.DELETE("/reminders/:id", controllers.DeleteReminder)

			// Export routes
			protected.GET("/export", controllers.ExportData)

			// Graph/Network visualization route
			protected.GET("/graph", controllers.GetGraph)
		}
	}
}
