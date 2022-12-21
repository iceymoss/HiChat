package service

import (
	"HiChat/dao"
	"net/http"

	"github.com/gin-gonic/gin"
)

//List 获取用户列表
// @Summary List 获取用户列表
// @Description 用户列表
// @Tags 测试
// @Accept json
// @Router /list [get]
func List(ctx *gin.Context) {
	list := dao.GetUserList()
	ctx.JSON(http.StatusOK, list)
}
