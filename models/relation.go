package models

type Relation struct {
	Model
	OwnerId  uint   //谁的关系信息
	TargetID uint   //对应的谁
	Type     int    //关系类型：0 1 2
	Desc     string //描述
}

func (r *Relation) RelTableName() string {
	return "relation"
}
