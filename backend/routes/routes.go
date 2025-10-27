package routes

import (
	"perema/config"
	"perema/controllers"
	"perema/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, cfg *config.Config) {

	// Health check endpoint (no versioning, standard practice)
	router.GET("/health", controllers.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		v1.POST("/register", controllers.RegisterUser)
		v1.POST("/login", func(c *gin.Context) {
			controllers.LoginUser(c, cfg)
		})

		// Protected routes (authentication required)
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// Contact routes
			protected.GET("/contacts", controllers.GetContacts)
			protected.POST("/contacts", controllers.CreateContact)
			protected.GET("/contacts/:id", controllers.GetContact)
			protected.PUT("/contacts/:id", controllers.UpdateContact)
			protected.DELETE("/contacts/:id", controllers.DeleteContact)
			protected.GET("/contacts/circles", controllers.GetCircles)

			// Relationship routes
			protected.GET("/contacts/:id/relationships", controllers.GetRelationships)
			protected.POST("/contacts/:id/relationships", controllers.CreateRelationship)
			protected.PUT("/contacts/:id/relationships/:rid", controllers.UpdateRelationship)
			protected.DELETE("/contacts/:id/relationships/:rid", controllers.DeleteRelationship)

			// Profile picture routes
			protected.POST("/contacts/:id/profile_picture", controllers.AddPhotoToContact)
			protected.GET("/contacts/:id/profile_picture", controllers.GetProfilePicture)

			// Note routes
			protected.GET("/contacts/:id/notes", controllers.GetNotesForContact)
			protected.POST("/contacts/:id/notes", controllers.CreateNote)
			protected.GET("/notes/:id", controllers.GetNote)
			protected.GET("/notes", controllers.GetUnassignedNotes)
			protected.POST("/notes", controllers.CreateUnassignedNote)
			protected.PUT("/notes/:id", controllers.UpdateNote)
			protected.DELETE("/notes/:id", controllers.DeleteNote)

			// Activity routes
			protected.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
			protected.POST("/activities", controllers.CreateActivity)
			protected.GET("/activities", controllers.GetActivities)
			protected.GET("/activities/:id", controllers.GetActivity)
			protected.PUT("/activities/:id", controllers.UpdateActivity)
			protected.DELETE("/activities/:id", controllers.DeleteActivity)

			// Reminder routes
			protected.GET("/contacts/:id/reminders", controllers.GetRemindersForContact)
			protected.POST("/contacts/:id/reminders", controllers.CreateReminder)
			protected.GET("/reminders/:id", controllers.GetReminder)
			protected.PUT("/reminders/:id", controllers.UpdateReminder)
			protected.DELETE("/reminders/:id", controllers.DeleteReminder)
		}
	}
}
