package dao

import (
	"HiChat/global"
	"HiChat/models"

	"go.uber.org/zap"
)

func GetUserList() []*models.UserBasic {
	var list []*models.UserBasic
	if tx := global.DB.Find(&list); tx.RowsAffected == 0 {
		zap.S().Info("获取用户失败")
		return nil
	}
	return list
}
