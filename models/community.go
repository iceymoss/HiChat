package models

type Community struct {
	Model
	Name    string //群名称
	OwnerId uint   //群拥有者
	Type    int    //群类型
	Image   string //头像
	Desc    string //描述
}
