package routes

import (
	"meerkat/carddav"
	"meerkat/config"
	"meerkat/controllers"
	"meerkat/middleware"
	"meerkat/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {

	// Health check endpoint (no versioning, standard practice)
	router.GET("/health", controllers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required, strict rate limiting)
		v1.POST("/register", middleware.AuthRateLimitMiddleware(), middleware.ValidateJSONMiddleware(&models.UserRegistrationInput{}), controllers.RegisterUser)
		v1.POST("/login", middleware.AuthRateLimitMiddleware(), func(c *gin.Context) {
			controllers.LoginUser(c, cfg)
		})
		v1.POST("/logout", func(c *gin.Context) {
			controllers.LogoutUser(c, cfg)
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
			protected.PATCH("/users/language", controllers.UpdateLanguage)
		protected.PATCH("/users/date-format", controllers.UpdateDateFormat)
			protected.GET("/users/custom-fields", controllers.GetCustomFieldNames)
			protected.PATCH("/users/custom-fields", middleware.ValidateJSONMiddleware(&models.CustomFieldNamesInput{}), controllers.UpdateCustomFieldNames)
			protected.GET("/users/me", controllers.GetCurrentUser)

			// Contact routes
			protected.GET("/contacts", controllers.GetContacts)
			protected.GET("/contacts/circles", controllers.GetCircles)
			protected.GET("/contacts/random", controllers.GetContactsRandom)
			protected.GET("/contacts/birthdays", controllers.GetUpcomingBirthdays)
			protected.POST("/contacts", middleware.ValidateJSONMiddleware(&models.ContactInput{}), controllers.CreateContact)
			protected.GET("/contacts/:id", controllers.GetContact)
			protected.PUT("/contacts/:id", middleware.ValidateJSONMiddleware(&models.ContactInput{}), controllers.UpdateContact)
			protected.DELETE("/contacts/:id", controllers.DeleteContact)
			protected.POST("/contacts/:id/archive", controllers.ArchiveContact)
			protected.POST("/contacts/:id/unarchive", controllers.UnarchiveContact)

			// Contact import routes (CSV)
			protected.POST("/contacts/import/upload", controllers.UploadCSVForImport)
			protected.POST("/contacts/import/preview", middleware.ValidateJSONMiddleware(&models.ImportPreviewRequest{}), controllers.PreviewImport)
			protected.POST("/contacts/import/confirm", middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{}), controllers.ConfirmImport)

			// Contact import routes (VCF)
			protected.POST("/contacts/import/vcf/upload", func(c *gin.Context) {
				controllers.UploadVCFForImport(c, cfg)
			})
			protected.POST("/contacts/import/vcf/confirm", middleware.ValidateJSONMiddleware(&models.ImportConfirmRequest{}), func(c *gin.Context) {
				controllers.ConfirmVCFImport(c, cfg)
			})

			// Relationship routes
			protected.GET("/contacts/:id/relationships", controllers.GetRelationships)
			protected.GET("/contacts/:id/incoming-relationships", controllers.GetIncomingRelationships)
			protected.POST("/contacts/:id/relationships", middleware.ValidateJSONMiddleware(&models.RelationshipInput{}), controllers.CreateRelationship)
			protected.PUT("/contacts/:id/relationships/:rid", middleware.ValidateJSONMiddleware(&models.RelationshipInput{}), controllers.UpdateRelationship)
			protected.DELETE("/contacts/:id/relationships/:rid", controllers.DeleteRelationship)

			// Profile picture routes
			protected.POST("/contacts/:id/profile_picture", func(c *gin.Context) {
				controllers.AddPhotoToContact(c, cfg)
			})
			protected.GET("/contacts/:id/profile_picture", func(c *gin.Context) {
				controllers.GetProfilePicture(c, cfg)
			})

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

			// Reminder completion routes (for timeline)
			protected.GET("/contacts/:id/reminder-completions", controllers.GetCompletionsForContact)
			protected.DELETE("/reminder-completions/:id", controllers.DeleteCompletion)

			// Export routes
			protected.GET("/export", controllers.ExportData)
			protected.GET("/export/vcf", func(c *gin.Context) {
				controllers.ExportContactsAsVCF(c, cfg.ProfilePhotoDir)
			})

			// Graph/Network visualization route
			protected.GET("/graph", controllers.GetGraph)
		}

		// Admin routes (admin authentication required)
		admin := v1.Group("/admin")
		admin.Use(middleware.APIRateLimitMiddleware())
		admin.Use(middleware.AuthMiddleware(cfg))
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/users", controllers.ListUsers)
			admin.GET("/users/:id", controllers.GetUser)
			admin.PATCH("/users/:id", middleware.ValidateJSONMiddleware(&models.AdminUserUpdateInput{}), controllers.UpdateUser)
			admin.DELETE("/users/:id", controllers.DeleteUser)
		}
	}

	// CardDAV routes (optional, enabled via CARDDAV_ENABLED)
	if cfg.CardDAVEnabled {
		registerCardDAVRoutes(router, cfg, db)
	}
}

// registerCardDAVRoutes sets up CardDAV endpoints for contact synchronization
func registerCardDAVRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {
	// Well-known discovery endpoint (no auth required for discovery)
	router.GET("/.well-known/carddav", carddav.WellKnownRedirect)

	// Create CardDAV handler
	handler := carddav.NewHandler(db, cfg.ProfilePhotoDir)

	// CardDAV routes with Basic Auth and rate limiting
	cardDAVGroup := router.Group("/carddav")
	cardDAVGroup.Use(func(c *gin.Context) {
		// Inject db into context for BasicAuthMiddleware
		c.Set("db", db)
		c.Next()
	})
	cardDAVGroup.Use(middleware.AuthRateLimitMiddleware())
	cardDAVGroup.Use(carddav.BasicAuthMiddleware())
	{
		// Handle all CardDAV methods on all paths
		// Note: Gin's Any() doesn't include WebDAV methods, so we add them explicitly
		ginHandler := handler.GinHandler()
		cardDAVGroup.Any("/*path", ginHandler)
		// WebDAV methods required for CardDAV
		cardDAVGroup.Handle("PROPFIND", "/*path", ginHandler)
		cardDAVGroup.Handle("REPORT", "/*path", ginHandler)
		cardDAVGroup.Handle("MKCOL", "/*path", ginHandler)
		cardDAVGroup.Handle("COPY", "/*path", ginHandler)
		cardDAVGroup.Handle("MOVE", "/*path", ginHandler)
	}
}
