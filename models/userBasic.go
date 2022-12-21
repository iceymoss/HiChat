package models

import (
	"gorm.io/gorm"

	"time"
)

type Model struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type UserBasic struct {
	Model
	Name          string
	PassWord      string
	Gender        string `gorm:"column:gender;default:male;type:varchar(6) comment 'male表示男， famale表示女'"`
	Phone         string
	Email         string
	Identity      string
	ClientIp      string
	ClientPort    string
	LoginTime     *time.Time `gorm:"column:login_time"`
	HeartBeatTime *time.Time `gorm:"column:heart_beat_time"`
	LoginOutTime  *time.Time `gorm:"column:login_out_time"`
	IsLoginOut    bool
	DeviceInfo    string
}

func (table *UserBasic) UserTableName() string {
	return "user_basic"
}
