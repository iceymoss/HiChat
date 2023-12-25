package main

import (
	"HiChat/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:password@tcp(1.14.180.202:3306)/hi_chat?charset=utf8mb4&parseTime=True&loc=Local"
	//注意：pass为MySQL数据库的管理员密码，dbname为要连接的数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&models.Message{}, &models.GroupInfo{}, &models.Relation{}, &models.UserBasic{}, &models.Community{})
	if err != nil {
		panic(err)
	}
}
