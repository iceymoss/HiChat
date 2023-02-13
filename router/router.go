package router

import (
	"HiChat/middlewear"
	"HiChat/service"

	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()

	//静态资源
	router.Static("/asset", "asset/")
	router.LoadHTMLGlob("views/**/*")
	router.GET("/", service.GetIndex)
	router.GET("/index", service.GetIndex)
	router.GET("/register", service.GetRegister)
	router.GET("/toChat", service.ToChat)

	v1 := router.Group("v1")

	//用户模块
	user := v1.Group("user")
	{
		user.GET("/list", middlewear.JWY(), service.List)
		user.POST("/login_pw", service.LoginByNameAndPassWord)
		user.POST("/new", service.NewUser)
		user.DELETE("/delete", middlewear.JWY(), service.DeleteUser)
		user.POST("/updata", middlewear.JWY(), service.UpdataUser)
		user.GET("/ws", middlewear.JWY(), service.SendMsg)
		user.GET("/SendUserMsg", middlewear.JWY(), service.SendUserMsg)
	}

	//图片、语音模块
	upload := v1.Group("upload").Use(middlewear.JWY())
	{
		upload.POST("/image", service.Image)
	}

	//好友关系
	relation := v1.Group("relation").Use(middlewear.JWY())
	{
		relation.POST("/list", service.FriendList)
		relation.POST("/add", service.AddFriendByName)
		relation.POST("/new_group", service.NewGroup)
		relation.POST("/group_list", service.GroupList)
		relation.POST("/join_group", service.JoinGroup)
	}

	//聊天记录
	v1.POST("/user/redisMsg", service.RedisMsg).Use(middlewear.JWY())

	return router
}
