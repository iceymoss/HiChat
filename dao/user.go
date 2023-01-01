package dao

import (
	"errors"
	"strconv"
	"time"

	"HiChat/common"
	"HiChat/global"
	"HiChat/models"

	"go.uber.org/zap"
)

func GetUserList() ([]*models.UserBasic, error) {
	var list []*models.UserBasic
	if tx := global.DB.Find(&list); tx.RowsAffected == 0 {
		return nil, errors.New("获取用户列表失败")
	}
	return list, nil
}

//查询用户:根据昵称，根据电话，根据邮件

//FindUserByNameAndPwd 昵称和密码查询
func FindUserByNameAndPwd(name string, password string) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where("name = ? and pass_word=?", name, password).First(&user); tx.RowsAffected == 0 {
		return nil, errors.New("未查询到记录")
	}
	//token加密
	t := strconv.Itoa(int(time.Now().Unix()))
	temp := common.Md5encoder(t)
	if tx := global.DB.Model(&user).Where("id = ?", user.ID).Update("identity", temp); tx.RowsAffected == 0 {
		return nil, errors.New("写入identity失败")
	}
	return &user, nil
}

func FindUserByName(name string) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where("name = ?", name).First(&user); tx.RowsAffected == 0 {
		return nil, errors.New("没有查询到记录")
	}
	return &user, nil
}

func FindUser(name string) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where("name = ?", name).First(&user); tx.RowsAffected == 1 {
		return nil, errors.New("没有查询到记录")
	}
	return &user, nil
}

func FindUserByPhone(phone string) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where("phone = ?", phone).First(&user); tx.RowsAffected == 0 {
		return nil, errors.New("未查询到记录")
	}
	return &user, nil
}

func FindUerByEmail(email string) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where("email = ?", email).First(&user); tx.RowsAffected == 0 {
		return nil, errors.New("未查询到记录")
	}
	return &user, nil
}

func FindUserID(ID uint) (*models.UserBasic, error) {
	user := models.UserBasic{}
	if tx := global.DB.Where(ID).First(&user); tx.RowsAffected == 0 {
		return nil, errors.New("未查询到记录")
	}
	return &user, nil
}

//CreateUser 新建用户
func CreateUser(user models.UserBasic) (*models.UserBasic, error) {
	tx := global.DB.Create(&user)
	if tx.RowsAffected == 0 {
		zap.S().Info("新建用户失败")
		return nil, errors.New("新增用户失败")
	}
	return &user, nil
}

func UpdateUser(user models.UserBasic) (*models.UserBasic, error) {
	tx := global.DB.Model(&user).Updates(models.UserBasic{
		Name:     user.Name,
		PassWord: user.PassWord,
		Gender:   user.Gender,
		Phone:    user.Phone,
		Email:    user.Email,
		Avatar:   user.Avatar,
		Salt:     user.Salt,
	})
	if tx.RowsAffected == 0 {
		zap.S().Info("更新用户失败")
		return nil, errors.New("更新用户失败")
	}
	return &user, nil
}

func DeleteUser(user models.UserBasic) error {
	if tx := global.DB.Delete(&user); tx.RowsAffected == 0 {
		zap.S().Info("删除失败")
		return errors.New("删除用户失败")
	}
	return nil
}
