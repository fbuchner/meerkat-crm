package routes

import (
	"perema/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	// Contact routes
	router.GET("/contacts", controllers.GetAllContacts)
	router.POST("/contacts", controllers.CreateContact)
	router.GET("/contacts/circles", controllers.GetCircles)
	router.GET("/contacts/:id", controllers.GetContact)
	router.PUT("/contacts/:id", controllers.UpdateContact)
	router.DELETE("/contacts/:id", controllers.DeleteContact)
	router.POST("/contacts/:id/relationships", controllers.AddRelationshipToContact)
	router.POST("/contacts/:id/profile_picture", controllers.AddPhotoToContact)
	router.GET("/contacts/:id/profile_picture.jpg", controllers.GetProfilePicture)

	// Note routes
	router.GET("/contacts/:id/notes", controllers.GetNotesForContact)
	router.POST("/contacts/:id/notes", controllers.CreateNote)
	router.GET("/notes/:id", controllers.GetNote)
	router.GET("/notes", controllers.GetUnassignedNotes)
	router.POST("/notes", controllers.CreateUnassignedNote)
	router.PUT("/notes/:id", controllers.UpdateNote)
	router.DELETE("/notes/:id", controllers.DeleteNote)

	// Activity routes
	router.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
	router.POST("/activities", controllers.CreateActivity)
	router.GET("/activities", controllers.GetActivities)
	router.GET("/activities/:id", controllers.GetActivity)
	router.PUT("/activities/:id", controllers.UpdateActivity)
	router.DELETE("/activities/:id", controllers.DeleteActivity)
	router.POST("/activities/:id/contacts/:contact_id", controllers.AddContactToActivity)
	router.DELETE("/activities/:id/contacts/:contact_id", controllers.RemoveContactFromActivity)
}
