package service

import (
	"HiChat/models"
	"html/template"
	"strconv"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

//GetIndex 首页
// @Summary index 获取首页
// @Description 首页
// @Tags 测试
// @Accept json
// @Router /index [get]
func GetIndex(ctx *gin.Context) {
	tem, err := template.ParseFiles("index.html", "views/chat/head.html")
	if err != nil {
		panic(err)
	}

	tem.Execute(ctx.Writer, "欢迎登陆HiChat系统")
}

func GetRegister(ctx *gin.Context) {
	tem, err := template.ParseFiles("views/user/register.html")
	if err != nil {
		panic(err)
	}

	tem.Execute(ctx.Writer, "欢迎注册HiChat账号")

}

func ToChat(ctx *gin.Context) {
	tem, err := template.ParseFiles("views/chat/index.html",
		"views/chat/head.html",
		"views/chat/foot.html",
		"views/chat/tabmenu.html",
		"views/chat/concat.html",
		"views/chat/group.html",
		"views/chat/profile.html",
		"views/chat/main.html",
		"views/chat/createcom.html",
		"views/chat/userinfo.html",
	)
	if err != nil {
		panic(err)
	}

	//获取参数
	user := models.UserBasic{}
	id := ctx.Query("userId")
	uid, err := strconv.Atoi(id)
	if err != nil {
		zap.S().Info("id转换失败", err)
		return
	}
	user.ID = uint(uid)

	user.Identity = ctx.Query("token")

	zap.S().Info("获取数据：", user)
	tem.Execute(ctx.Writer, "欢迎来到HiChat主页")
}
