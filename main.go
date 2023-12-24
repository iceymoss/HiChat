package main

import (
	_ "HiChat/docs"
	"HiChat/global"
	"HiChat/initialize"
	"HiChat/models"
	"HiChat/router"
	"fmt"
	//swaggerFiles "github.com/swaggo/files"
	//ginSwagger "github.com/swaggo/gin-swagger"
)

// @title 这是一个测试文档
// @version 1.0
// @description HiCha聊天服务
// @host 127.0.0.1:8000
// @BasePath /v1
func main() {
	//初始化日志
	initialize.InitLogger()
	//初始化配置
	initialize.InitConfig()
	//初始化数据库
	initialize.InitDB()
	initialize.InitRedis()

	//持久化数据
	go models.RecordPersistence()

	router := router.Router()
	//router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.Run(fmt.Sprintf(":%d", global.ServiceConfig.Port))
}
