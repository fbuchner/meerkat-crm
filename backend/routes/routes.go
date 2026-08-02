package routes

import (
	"mycorrhizal/carddav"
	"mycorrhizal/config"
	"mycorrhizal/controllers"
	"mycorrhizal/middleware"
	"mycorrhizal/models"
	"mycorrhizal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB, oidcProvider *services.OIDCProvider) {

	// Health check endpoint (no versioning, standard practice)
	router.GET("/health", controllers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// OIDC routes (config always public; login/callback only when OIDC is enabled)
		v1.GET("/auth/oidc/config", controllers.OIDCConfigHandler(cfg))
		if cfg.OIDC.Enabled && oidcProvider != nil {
			v1.GET("/auth/oidc/login", middleware.AuthRateLimitMiddleware(), controllers.OIDCLoginHandler(oidcProvider, cfg))
			v1.GET("/auth/oidc/callback", middleware.AuthRateLimitMiddleware(), controllers.OIDCCallbackHandler(oidcProvider, cfg))
		}

		// Public routes (no authentication required, strict rate limiting)
		v1.POST("/register", middleware.AuthRateLimitMiddleware(), middleware.ValidateJSONMiddleware(&models.UserRegistrationInput{}), controllers.RegisterUser(cfg))
		v1.POST("/login", middleware.AuthRateLimitMiddleware(), func(c *gin.Context) {
			controllers.LoginUser(c, cfg)
		})
		v1.POST("/logout", func(c *gin.Context) {
			controllers.LogoutUser(c, cfg, oidcProvider)
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
			protected.GET("/users/enabled-contact-fields", controllers.GetEnabledContactFields)
			protected.PATCH("/users/enabled-contact-fields", middleware.ValidateJSONMiddleware(&models.EnabledContactFieldsInput{}), controllers.UpdateEnabledContactFields)
			protected.GET("/users/me", controllers.GetCurrentUser)

			// Contact routes
			protected.GET("/contacts", controllers.GetContacts)
			protected.GET("/contacts/circles", controllers.GetCircles)
			protected.GET("/contacts/random", controllers.GetContactsRandom)
			protected.GET("/contacts/birthdays", controllers.GetUpcomingBirthdays)
			protected.POST("/contacts/merge/preview", middleware.ValidateJSONMiddleware(&models.ContactMergeRequest{}), controllers.PreviewContactMerge)
			protected.POST("/contacts/merge", middleware.ValidateJSONMiddleware(&models.ContactMergeRequest{}), controllers.CommitContactMerge)
			protected.POST("/contacts", middleware.ValidateJSONMiddleware(&models.ContactRecordInput{}), controllers.CreateContact)
			protected.GET("/contacts/:id", controllers.GetContact)
			protected.PUT("/contacts/:id", middleware.ValidateJSONMiddleware(&models.ContactRecordInput{}), controllers.UpdateContact)
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

			// Contact import routes (JSContact JSON) — WP-71 Gap 4 extension.
			// Confirmation deliberately reuses /contacts/import/vcf/confirm
			// (see UploadJSContactForImport's doc comment): the session it
			// creates is format-agnostic once parsed into []VCFContactData.
			protected.POST("/contacts/import/jscontact/upload", controllers.UploadJSContactForImport)

			// RelationshipEdge routes (graph-model relationship API; replaces the
			// legacy /contacts/:id/relationships stack, removed in §3d WP5)
			protected.POST("/relationship-edges", middleware.ValidateJSONMiddleware(&models.RelationshipEdgeInput{}), controllers.CreateRelationshipEdge)
			protected.GET("/relationship-edges", controllers.ListRelationshipEdges)
			protected.GET("/relationship-edges/:id", controllers.GetRelationshipEdge)
			protected.PUT("/relationship-edges/:id", middleware.ValidateJSONMiddleware(&models.RelationshipEdgeInput{}), controllers.UpdateRelationshipEdge)
			protected.DELETE("/relationship-edges/:id", controllers.DeleteRelationshipEdge)
			protected.PATCH("/relationship-edges/:id/accept", controllers.AcceptRelationshipEdge)

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

			// Circle routes (WP-84c)
			protected.POST("/circles", middleware.ValidateJSONMiddleware(&models.CircleInput{}), controllers.CreateCircle)
			protected.GET("/circles", controllers.ListCircles)
			protected.GET("/circles/:id", controllers.GetCircle)
			protected.PUT("/circles/:id", middleware.ValidateJSONMiddleware(&models.CircleInput{}), controllers.UpdateCircle)
			protected.DELETE("/circles/:id", controllers.DeleteCircle)
			protected.POST("/circles/:id/members", middleware.ValidateJSONMiddleware(&models.CircleMemberInput{}), controllers.AddCircleMember)
			protected.DELETE("/circles/:id/members/:vcard_uid", controllers.RemoveCircleMember)

			// Household routes (T1 — docs/fork-plan/tickets/09-T1-households.md)
			protected.POST("/households", middleware.ValidateJSONMiddleware(&models.HouseholdInput{}), controllers.CreateHousehold)
			protected.GET("/households", controllers.ListHouseholds)
			protected.GET("/households/:id", controllers.GetHousehold)
			protected.PUT("/households/:id", middleware.ValidateJSONMiddleware(&models.HouseholdInput{}), controllers.UpdateHousehold)
			protected.DELETE("/households/:id", controllers.DeleteHousehold)
			protected.POST("/households/:id/members", middleware.ValidateJSONMiddleware(&models.HouseholdMemberInput{}), controllers.AddHouseholdMember)
			protected.DELETE("/households/:id/members/:vcard_uid", controllers.RemoveHouseholdMember)
			protected.PATCH("/households/:id/members/:vcard_uid", controllers.UpdateHouseholdMember)
			protected.POST("/households/:id/suggest-relationships", controllers.SuggestHouseholdRelationships)

			// Tag routes (WP-84c)
			protected.POST("/tags", middleware.ValidateJSONMiddleware(&models.TagInput{}), controllers.CreateTag)
			protected.GET("/tags", controllers.ListTags)
			protected.GET("/tags/:id", controllers.GetTag)
			protected.PUT("/tags/:id", middleware.ValidateJSONMiddleware(&models.TagInput{}), controllers.UpdateTag)
			protected.DELETE("/tags/:id", controllers.DeleteTag)
			protected.POST("/tags/:id/contacts", middleware.ValidateJSONMiddleware(&models.ContactTagInput{}), controllers.AddContactTag)
			protected.DELETE("/tags/:id/contacts/:vcard_uid", controllers.RemoveContactTag)

			// LifeEvent routes (WP-84c)
			protected.POST("/life-events", middleware.ValidateJSONMiddleware(&models.LifeEventInput{}), controllers.CreateLifeEvent)
			protected.GET("/life-events", controllers.ListLifeEvents)
			protected.GET("/life-events/:id", controllers.GetLifeEvent)
			protected.PUT("/life-events/:id", middleware.ValidateJSONMiddleware(&models.LifeEventInput{}), controllers.UpdateLifeEvent)
			protected.DELETE("/life-events/:id", controllers.DeleteLifeEvent)

			// Preference routes (T20a — docs/fork-plan/tickets/10-T20a-preferences.md)
			protected.POST("/preferences", middleware.ValidateJSONMiddleware(&models.PreferenceInput{}), controllers.CreatePreference)
			protected.GET("/preferences", controllers.ListPreferences)
			protected.GET("/preferences/:id", controllers.GetPreference)
			protected.PUT("/preferences/:id", middleware.ValidateJSONMiddleware(&models.PreferenceInput{}), controllers.UpdatePreference)
			protected.DELETE("/preferences/:id", controllers.DeletePreference)

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
			protected.GET("/export/jscontact", controllers.ExportContactsAsJSContact)

			// Graph/Network visualization route
			protected.GET("/graph", controllers.GetGraph)

			// API token routes
			protected.GET("/api-tokens", controllers.ListApiTokens)
			protected.POST("/api-tokens", middleware.ValidateJSONMiddleware(&models.ApiTokenInput{}), controllers.CreateApiToken)
			protected.DELETE("/api-tokens/:id", controllers.RevokeApiToken)

			// Webhook routes
			protected.GET("/webhooks", controllers.ListWebhooks)
			protected.POST("/webhooks", middleware.ValidateJSONMiddleware(&models.WebhookInput{}), controllers.CreateWebhook)
			protected.GET("/webhooks/:id", controllers.GetWebhook)
			protected.PUT("/webhooks/:id", middleware.ValidateJSONMiddleware(&models.WebhookInput{}), controllers.UpdateWebhook)
			protected.DELETE("/webhooks/:id", controllers.DeleteWebhook)
			protected.POST("/webhooks/:id/test", controllers.TestWebhook)
			protected.GET("/webhooks/:id/deliveries", controllers.GetWebhookDeliveries)

			// Calendar subscription routes (CalDAV/iCS activity import)
			protected.GET("/calendars", controllers.ListCalendarSubscriptions)
			protected.POST("/calendars", middleware.ValidateJSONMiddleware(&models.CalendarSubscriptionInput{}), controllers.CreateCalendarSubscription)
			protected.PUT("/calendars/:id", middleware.ValidateJSONMiddleware(&models.CalendarSubscriptionInput{}), controllers.UpdateCalendarSubscription)
			protected.DELETE("/calendars/:id", controllers.DeleteCalendarSubscription)
			protected.POST("/calendars/:id/sync", controllers.SyncCalendarSubscription)

			// Contact subscription routes (CardDAV client: sync contacts in
			// from an external address book, WP-73b)
			protected.GET("/contact-subscriptions", controllers.ListContactSubscriptions)
			protected.POST("/contact-subscriptions", middleware.ValidateJSONMiddleware(&models.ContactSubscriptionInput{}), controllers.CreateContactSubscription)
			protected.PUT("/contact-subscriptions/:id", middleware.ValidateJSONMiddleware(&models.ContactSubscriptionInput{}), controllers.UpdateContactSubscription)
			protected.DELETE("/contact-subscriptions/:id", controllers.DeleteContactSubscription)
			protected.POST("/contact-subscriptions/:id/sync", controllers.SyncContactSubscription)
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
			admin.POST("/trigger-reminders", func(c *gin.Context) {
				controllers.TriggerReminders(c, *cfg)
			})
			admin.POST("/trigger-purge", func(c *gin.Context) {
				controllers.TriggerPurge(c, *cfg)
			})
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

	handler := carddav.NewHandler(db, cfg.ProfilePhotoDir)

	cardDAVGroup := router.Group("/carddav")
	cardDAVGroup.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	cardDAVGroup.Use(middleware.CardDAVRateLimitMiddleware())
	cardDAVGroup.Use(carddav.BasicAuthMiddleware())
	{
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
