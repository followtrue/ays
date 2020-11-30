package models

import (
	"time"
)

type Node struct {
	Id                int       `xorm:"not null pk autoincr INT(11)"`
	NodeGroupAlias    string    `xorm:"not null comment('节点组alias') CHAR(20)"`
	NodeId            string    `xorm:"not null comment('执行器节点在consul中的ID') CHAR(32)"`
	NodeIp            string    `xorm:"not null default '' comment('执行器节点IP') CHAR(15)"`
	NodePort          int       `xorm:"not null default 0 comment('执行器节点端口') INT(11)"`
	NodeHealth        string    `xorm:"not null default '' comment('执行器节点健康状态：passing-通过检查，warning-警告，critical-危急，maintenance-维护') VARCHAR(20)"`
	CreatedAt         time.Time `xorm:"created DATETIME"`
	UpdatedAt         time.Time `xorm:"updated DATETIME"`
}

const (
	NodeHealth_PASSING = "passing"
	NodeHealth_WARNING = "warning"
	NodeHealth_CRITICAL = "critical"
	NodeHealth_MAINTENANCE = "maintenance"
	NodeHealth_OFFLINE = "offline"
)

// 新增
func (node *Node) Create() (insertId int, err error) {
	_, err = DB.Insert(node)
	if err == nil {
		insertId = node.Id
	}

	return insertId, err
}

// 不存在新增，存在更新
func (node *Node) CreateOrUpdate() (int, error) {
	var id int
	var isExists bool
	var err error

	if node.NodeIp != "" && node.NodePort != 0 {
		isExists, err = DB.Table(node).Where("node_ip = ?", node.NodeIp).And("node_port = ?", node.NodePort).Exist()
	} else if node.NodeId != "" {
		isExists, err = DB.Table(node).Where("node_id = ?", node.NodeId).Exist()
	} else {
		return 0, nil
	}
	if err != nil {
		return id, err
	}

	if isExists {
		_, err = node.Update()
		id = node.Id
	} else {
		id, err = node.Create()
	}

	return id, err
}

// 更新所有指定字段，未填的强制设置为空
func (node *Node) UpdateBean(id int) (int64, error) {
	return DB.ID(id).Cols("node_group_id, node_id, node_ip").Update(node)
}

// 更新有值字段
func (node *Node) UpdateMap(id int, data CommonMap) (int64, error) {
	return DB.Table(node).ID(id).Update(data)
}

// 更新
func (node *Node) Update() (int64, error) {
	if node.NodeHealth == NodeHealth_OFFLINE {
		return DB.Table(node).Where("node_id = ?", node.NodeId).Update(CommonMap{
			"node_health": NodeHealth_OFFLINE,
		})
	} else {
		return DB.Where("node_ip = ?", node.NodeIp).And("node_port = ?", node.NodePort).
			Cols("node_group_alias, node_id, node_ip, node_port, node_health").Update(node)
	}
}

// 查找节点
func (node *Node) Find(ip string, port int) (bool, error) {
	return DB.Table(node).Where("node_ip = ?", ip).And("node_port = ?", port).Get(node)

}

// 删除
func (node *Node) Delete() (int64, error) {
	return DB.Delete(node)
}

// 获取node节点
func (node *Node) GetNodesByNodeGroupAlias(NodeGroupAlia string) ([]Node, error) {
	nodes := make([]Node, 0)
	err := DB.Where("node_group_alias=?", NodeGroupAlia).Find(&nodes)
	return nodes, err
}