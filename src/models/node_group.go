package models

import (
	"time"
)

const RegTypeAuto = 1
const RegTypeManual  = 2

type NodeGroup struct {
	Id             int       `xorm:"not null pk autoincr INT(11)"`
	NodeGroupAlias string    `xorm:"not null default '' comment('执行器代称') unique CHAR(20)"`
	NodeGroupName  string    `xorm:"not null default '' comment('执行器名称') VARCHAR(255)"`
	NodeGroupRank  int       `xorm:"not null default 1 comment('执行器排序') INT(11)"`
	NodeRegType    int       `xorm:"not null default 1 comment('注册方式：1.自动注册，2.手动注册') TINYINT(1)"`
	UserId         int       `xorm:"not null default 0 comment('最后操作者ID') INT(11)"`
	CreatedAt      time.Time `xorm:"created comment('添加时间') DATETIME"`
	UpdatedAt      time.Time `xorm:"updated comment('更新时间') DATETIME"`
}

// 根据alias判断是否存在
func (nodeGroup *NodeGroup) Exists() (isExists bool, err error) {
	return DB.Table(nodeGroup).Where("node_group_alias = ?", nodeGroup.NodeGroupAlias).Exist()
}

// 新增
func (nodeGroup *NodeGroup) Create() (insertId int, err error) {
	_, err = DB.Insert(nodeGroup)
	if err == nil {
		insertId = nodeGroup.Id
	}

	return insertId, err
}

// 更新所有指定字段，未填的强制设置为空
func (nodeGroup *NodeGroup) UpdateBean(id int) (int64, error) {
	return DB.ID(id).Cols("node_group_alias, node_group_name, node_group_rank, node_reg_type, user_id").Update(nodeGroup)
}

// 更新有值字段
func (nodeGroup *NodeGroup) UpdateMap(id int, data CommonMap) (int64, error) {
	return DB.Table(nodeGroup).ID(id).Update(data)
}

// 更新
func (nodeGroup *NodeGroup) Update() (int64, error) {
	return DB.Where("id = ?", nodeGroup.Id).Cols("node_group_alias, node_group_name, node_group_rank, node_reg_type, user_id").Update(nodeGroup)
}

// 删除
func (nodeGroup *NodeGroup) Delete() (int64, error) {
	return DB.Id(nodeGroup.Id).Delete(new(NodeGroup))
}

// 查找执行器组
func (nodeGroup *NodeGroup) Find(alias string) (bool, error) {
	return DB.Table(nodeGroup).Where("node_group_alias = ?", alias).Get(nodeGroup)
}


// 查询总数量
func (nodeGroup *NodeGroup) TotalNum(groupAlias, groupName string) (int64, error) {
        if groupAlias != "" {
                nodeGroup.NodeGroupAlias = groupAlias
        }

        if groupName != "" {
                nodeGroup.NodeGroupName = groupName
        }
        return DB.Count(nodeGroup)
}


// 获取job列表
func (nodeGroup *NodeGroup) GetList(pre_page int, page int, groupAlias, groupName string) []NodeGroup {
	jobs := make([]NodeGroup, 0)
	engine := DB.AllCols()

	if groupAlias != "" {
		engine = engine.Where("node_group_alias like ?", "%"+groupAlias+"%")
	}

	if groupName != "" {
		engine = engine.Where("node_group_name like ?", "%"+groupName+"%")
	}

	startIndex := pre_page * (page - 1)
	engine.OrderBy("id DESC,node_group_rank DESC").Limit(pre_page, startIndex).Find(&jobs)

	return jobs
}

func (nodeGroup *NodeGroup) CountNodeGroup() (total int64, err error){
	return DB.Count(nodeGroup)
}
