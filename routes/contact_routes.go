package routes

import (
	"perema/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.POST("/contacts", controllers.CreateContact)
	router.GET("/contacts", controllers.GetAllContacts)
	router.GET("/contacts/:id", controllers.GetContact)
	router.PUT("/contacts/:id", controllers.UpdateContact)
	router.DELETE("/contacts/:id", controllers.DeleteContact)
}
