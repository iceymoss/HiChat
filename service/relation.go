package service

import (
	"strconv"

	"HiChat/common"
	"HiChat/dao"
	"HiChat/middlewear"
	"HiChat/models"

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
	userID, _ := middlewear.AuthenticatedUserID(ctx)
	users, err := dao.FriendList(userID)
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

// AddFriendByName 通过加好友
func AddFriendByName(ctx *gin.Context) {
	userID, _ := middlewear.AuthenticatedUserID(ctx)

	tar := ctx.PostForm("targetName")
	target, err := strconv.Atoi(tar)
	if err != nil {
		code, err := dao.AddFriendByName(userID, tar)
		if err != nil {
			HandleErr(code, ctx, err)
			return
		}

	} else {
		code, err := dao.AddFriend(userID, uint(target))
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
	ownerID, _ := middlewear.AuthenticatedUserID(ctx)

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
	community.OwnerId = ownerID

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

func GroupList(ctx *gin.Context) {
	ownerID, _ := middlewear.AuthenticatedUserID(ctx)

	rsp, err := dao.GetCommunityList(ownerID)
	if err != nil {
		zap.S().Info("获取群列表失败", err)
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "你还没加入任何群聊",
		})
		return
	}

	common.RespOKList(ctx.Writer, rsp, len(*rsp))
}

func JoinGroup(ctx *gin.Context) {
	comInfo := ctx.PostForm("comId")
	if comInfo == "" {
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "群名称不能为空",
		})
		return
	}

	userID, _ := middlewear.AuthenticatedUserID(ctx)

	code, err := dao.JoinCommunity(userID, comInfo)
	if err != nil {
		HandleErr(code, ctx, err)
		return
	}

	ctx.JSON(200, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "加群成功",
	})
}

func RedisMsg(c *gin.Context) {
	authenticatedUserID, _ := middlewear.AuthenticatedUserID(c)
	userIdA := int(authenticatedUserID)
	userIdB, err := strconv.Atoi(c.PostForm("userIdB"))
	if err != nil || userIdB <= 0 {
		c.JSON(400, gin.H{"code": -1, "message": "用户ID不合法"})
		return
	}
	start, _ := strconv.Atoi(c.PostForm("start"))
	end, _ := strconv.Atoi(c.PostForm("end"))
	isRev, _ := strconv.ParseBool(c.PostForm("isRev"))
	res := models.RedisMsg(int64(userIdA), int64(userIdB), int64(start), int64(end), isRev)
	common.RespOKList(c.Writer, "ok", res)
}

func GroupRedisMsg(c *gin.Context) {
	userID, _ := middlewear.AuthenticatedUserID(c)
	groupID, err := strconv.ParseUint(c.PostForm("groupId"), 10, 64)
	if err != nil || groupID == 0 {
		c.JSON(400, gin.H{"code": -1, "message": "群ID不合法"})
		return
	}
	if !models.IsGroupMember(userID, uint(groupID)) {
		c.JSON(403, gin.H{"code": -1, "message": "您不是该群成员"})
		return
	}

	start, _ := strconv.ParseInt(c.PostForm("start"), 10, 64)
	end, _ := strconv.ParseInt(c.PostForm("end"), 10, 64)
	isRev, _ := strconv.ParseBool(c.PostForm("isRev"))
	res := models.GroupRedisMsg(uint(groupID), start, end, isRev)
	common.RespOKList(c.Writer, "ok", res)
}
