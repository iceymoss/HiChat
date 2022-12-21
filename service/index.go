package service

import (
	"HiChat/global"
	"HiChat/models"

	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//GetIndex 首页
// @Summary index 获取首页
// @Description 首页
// @Tags 测试
// @Accept json
// @Router /index [get]
func GetIndex(ctx *gin.Context) {
	var user models.UserBasic
	if tx := global.DB.Find(&user); tx.RowsAffected == 0 {
		zap.S().Info("未查询到用户数据")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"userID": user.ID,
		"name":   user.Name,
	})
}
