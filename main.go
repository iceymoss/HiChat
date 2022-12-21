package main

import (
	_ "HiChat/docs"
	"HiChat/initialize"
	"HiChat/router"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title 这是一个测试文档
// @version 1.0
// @description HiCha聊天服务
// @host 127.0.0.1:8000
// @BasePath /user
func main() {
	//初始化日志
	initialize.InitLogger()
	//初始化配置
	initialize.InitConfig()
	//初始化数据库
	initialize.InitDB()

	router := router.Router()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.Run(":8000")
}
