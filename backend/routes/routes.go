package routes

import (
	"perema/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {

	// Routes from contact controller
	router.GET("/contacts", controllers.GetContacts)
	router.POST("/contacts", controllers.CreateContact)
	router.GET("/contacts/:id", controllers.GetContact)
	router.PUT("/contacts/:id", controllers.UpdateContact)
	router.DELETE("/contacts/:id", controllers.DeleteContact)
	router.GET("/contacts/circles", controllers.GetCircles)

	// Routes from relationship controller
	router.GET("/contacts/:id/relationships", controllers.GetRelationships)
	router.POST("/contacts/:id/relationships", controllers.CreateRelationship)
	router.PUT("/contacts/:id/relationships/:rid", controllers.UpdateRelationship)
	router.DELETE("/contacts/:id/relationships/:rid", controllers.DeleteRelationship)

	// Routes from profile picture controller
	router.POST("/contacts/:id/profile_picture", controllers.AddPhotoToContact)
	router.GET("/contacts/:id/profile_picture.jpg", controllers.GetProfilePicture)

	// Routes from note controller
	router.GET("/contacts/:id/notes", controllers.GetNotesForContact)
	router.POST("/contacts/:id/notes", controllers.CreateNote)
	router.GET("/notes/:id", controllers.GetNote)
	router.GET("/notes", controllers.GetUnassignedNotes)
	router.POST("/notes", controllers.CreateUnassignedNote)
	router.PUT("/notes/:id", controllers.UpdateNote)
	router.DELETE("/notes/:id", controllers.DeleteNote)

	// Routes from activity controller
	router.GET("/contacts/:id/activities", controllers.GetActivitiesForContact)
	router.POST("/activities", controllers.CreateActivity)
	router.GET("/activities", controllers.GetActivities)
	router.GET("/activities/:id", controllers.GetActivity)
	router.PUT("/activities/:id", controllers.UpdateActivity)
	router.DELETE("/activities/:id", controllers.DeleteActivity)

	// Routes from reminder controller
	router.GET("/contacts/:id/reminders", controllers.GetRemindersForContact)
	router.POST("/contacts/:id/reminders", controllers.CreateReminder)
	router.GET("/reminders/:id", controllers.GetReminder)
	router.PUT("/reminders/:id", controllers.UpdateReminder)
	router.DELETE("/reminders/:id", controllers.DeleteReminder)
}
