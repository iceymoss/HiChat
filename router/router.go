package router

import (
	"HiChat/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()

	//静态资源
	router.Static("/asset", "asset/")
	router.LoadHTMLGlob("views/**/*")
	router.GET("/index", service.GetIndex)
	router.GET("/register", service.GetRegister)
	router.GET("/toChat", service.ToChat)

	v1 := router.Group("v1")
	user := v1.Group("user")
	{
		user.GET("/list", service.List)
		user.POST("/login_pw", service.LoginByNameAndPassWord)
		user.POST("/new", service.NewUser)
		user.DELETE("/delete", service.DeleteUser)
		user.PUT("/updata", service.UpdateUser)
		user.GET("/ws", service.SendMsg)
		user.GET("/SendUserMsg", service.SendUserMsg)
	}

	upload := v1.Group("upload")
	{
		upload.POST("/image", service.Image)
	}

	relation := v1.Group("relation")
	{
		relation.POST("/list", service.FriendList)
		relation.POST("/add", service.AddFriendByID)
		relation.POST("/new_group", service.NewGroup)
	}

	return router
}
