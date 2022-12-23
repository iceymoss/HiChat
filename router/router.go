package router

import (
	"HiChat/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()

	v1 := router.Group("v1")
	user := v1.Group("user")
	{
		user.GET("/index", service.GetIndex)
		user.GET("/list", service.List)
		user.POST("/login_pw", service.LoginByNameAndPassWord)
		user.POST("/new", service.NewUser)
		user.DELETE("/delete", service.DeleteUser)
		user.PUT("/updata", service.UpdateUser)
		user.GET("/ws", service.SendMsg)
		user.GET("/SendUserMsg", service.SendUserMsg)
	}

	return router
}
