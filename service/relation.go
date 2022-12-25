package service

import (
	"HiChat/common"
	"HiChat/dao"
	"HiChat/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type user struct {
	Name     string
	Avatar   string
	Gender   string
	Phone    string
	Email    string
	Identity string
}

func FriendList(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Request.FormValue("userId"))
	users, err := dao.FriendList(uint(id))
	if err != nil {
		zap.S().Info("获取好友列表失败", err)
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "好友为空",
		})
		return
	}

	infos := make([]user, 0)

	for _, v := range *users {
		info := user{
			Name:     v.Name,
			Avatar:   v.Avatar,
			Gender:   v.Gender,
			Phone:    v.Phone,
			Email:    v.Email,
			Identity: v.Identity,
		}
		infos = append(infos, info)
	}
	common.RespOKList(ctx.Writer, users, len(infos))
}

//AddFriendByID 通过加好友
func AddFriendByID(ctx *gin.Context) {
	user := ctx.PostForm("userId")
	userId, err := strconv.Atoi(user)
	if err != nil {
		zap.S().Info("类型转换失败", err)
		return
	}

	tar := ctx.PostForm("targetName")
	target, err := strconv.Atoi(tar)
	if err != nil {
		code, err := dao.AddFriendByName(uint(userId), tar)
		if err != nil {
			HandleErr(code, ctx, err)
			return
		}

	} else {
		code, err := dao.AddFriend(uint(userId), uint(target))
		if err != nil {
			HandleErr(code, ctx, err)
			return
		}
	}
	ctx.JSON(200, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "添加好友成功",
	})
}

func HandleErr(code int, ctx *gin.Context, err error) {
	switch code {
	case -1:
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": err.Error(),
		})
	case 0:
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "该好友已经存在",
		})
	case -2:
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "不能添加自己",
		})

	}
}

func NewGroup(ctx *gin.Context) {
	owner := ctx.PostForm("ownerId")
	ownerId, err := strconv.Atoi(owner)
	if err != nil {
		zap.S().Info("owner类型转换失败", err)
		return
	}

	ty := ctx.PostForm("cate")
	Type, err := strconv.Atoi(ty)
	if err != nil {
		zap.S().Info("ty类型转换失败", err)
		return
	}

	img := ctx.PostForm("icon")
	name := ctx.PostForm("name")
	desc := ctx.PostForm("desc")

	community := models.Community{}
	if ownerId == 0 {
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "您未登录",
		})
		return
	}

	if name == "" {
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "群名称不能为空",
		})
		return
	}

	if img != "" {
		community.Image = img
	}
	if desc != "" {
		community.Desc = desc
	}

	community.Name = name
	community.Type = Type
	community.OwnerId = uint(ownerId)

	code, err := dao.CreateCommunity(community)
	if err != nil {
		HandleErr(code, ctx, err)
		return
	}

	ctx.JSON(200, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "键群成功",
	})
}
