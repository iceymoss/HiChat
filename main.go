package main

import (
	_ "HiChat/docs"
	"HiChat/global"
	"HiChat/initialize"
	"HiChat/models"
	"HiChat/router"
	"fmt"
	"time"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	router := router.Router()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	go router.Run(fmt.Sprintf(":%d", global.ServiceConfig.Port))

	time.Sleep(120 * time.Second)
	models.WriteDB("msg_7_8")
}
