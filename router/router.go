package router

import (
	"HiChat/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()

	user := router.Group("user")
	{
		user.GET("/index", service.GetIndex)
		user.GET("/list", service.List)
	}

	return router
}
