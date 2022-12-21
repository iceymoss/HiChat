package main

import (
	"HiChat/models"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:Qq/2013XiaoKUang@tcp(127.0.0.1:3306)/hi_chat?charset=utf8mb4&parseTime=True&loc=Local"
	//注意：pass为MySQL数据库的管理员密码，dbname为要连接的数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&models.UserBasic{})
	if err != nil {
		panic(err)
	}

	user := models.UserBasic{}
	user.Name = "iceymoss"

	t := time.Now()
	user.LoginTime = &t

	if tx := db.Create(&user); tx.RowsAffected == 0 {
		log.Fatal("新建用户失败", err)
	}

	db.Model(&user).Where(1).Update("pass_word", "123456")

}
