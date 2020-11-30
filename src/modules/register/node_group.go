package register

import (
	"encoding/json"
	"errors"
	"ays/src/models"
	"ays/src/modules/consul"
	"ays/src/modules/logger"
)

const (
	NODE_GROUP_INFO_PRE = "node_group_"
	NODE_GROUP_LIST = "list_node_group"
)

// 添加执行器（节点组）（调度器使用）
func NodeGroupAdd(nodeGroup models.NodeGroup) (bool, error) {
	// 不允许alias重复
	exist, err := nodeGroup.Exists()
	if exist || err != nil {
		return false, err
	}

	_, err = nodeGroup.Create()
	if err != nil {
		return false, err
	}

	return NodeGroupAddConsul(nodeGroup)
}

// consul中添加执行器（节点组）
func NodeGroupAddConsul(nodeGroup models.NodeGroup) (bool, error) {
	nodeGroupListAdd(nodeGroup.NodeGroupAlias)
	key := generateGroupInfoKey(nodeGroup.NodeGroupAlias)
	return consul.KvSetObj(key, nodeGroup)
}

// 更新执行器（节点组）（调度器使用）
func NodeGroupUpdate(nodeGroup models.NodeGroup) (bool, error) {
	_, err := nodeGroup.Update()

	if err != nil {
		return false, err
	} else {
		key := generateGroupInfoKey(nodeGroup.NodeGroupAlias)
		return consul.KvSetObj(key, nodeGroup)
	}
}

// 删除执行器组
func NodeGroupDel(nodeGroup models.NodeGroup) (bool, error) {
	nodeList := consul.FindByMeta(map[string]string{"node_group_alias":nodeGroup.NodeGroupAlias})
	if len(nodeList) > 0 {
		return false, errors.New("执行器组中还有活跃的节点，不能删除")
	}

	key := generateGroupInfoKey(nodeGroup.NodeGroupAlias)
	res, err := consul.KvDel(key)
	logger.IfError(err)
	if !res {
		return res, err
	}

	nodeGroupListDel(nodeGroup.NodeGroupAlias)
	_, err = nodeGroup.Delete()
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}
/*
type NodeGroupView struct {
	node_group_alias string
	node_group_name  string
	node_group_rank  int
	node_reg_type    int
	user_id          int
	created_at       time.Time
	updated_at       time.Time
	node_list        []NodeView
}
*/

type NodeGroupViewList []map[string]interface{}

// 执行器组列表（api）返回总数+viewlist
func NodeGroupViewListGet(pre_page int, page int, groupAlias, groupName string) (int64, NodeGroupViewList) {
        groupViewList := NodeGroupViewList{}
        //total, nodeGroupList := NodeGroupList(pre_page, page)
        group := models.NodeGroup{}
        total, err := group.TotalNum(groupAlias, groupName)
        logger.IfError(err)
        nodeGroupList := group.GetList(pre_page, page, groupAlias, groupName)

        for _, group := range nodeGroupList {

                groupViewList = append(groupViewList, map[string]interface{}{
                        "id": group.Id,
                        "node_group_alias": group.NodeGroupAlias,
                        "node_group_name": group.NodeGroupName,
                        "node_group_rank": group.NodeGroupRank,
                        "node_reg_type": group.NodeRegType,
                        "user_id": group.UserId,
                        "created_at": group.CreatedAt,
                        "updated_at": group.UpdatedAt,
                        "node_list": NodeFindByGroupMysql(group.NodeGroupAlias),
                })
        }

        return total, groupViewList
}



// 执行器组列表，返回总数+model list
func NodeGroupList(pre_page int, page int) (int, []*models.NodeGroup) {
	var nodeGroupList []*models.NodeGroup
	nodeGroupList = []*models.NodeGroup{}
	groupNameList := nodeGroupListGet()
	listLen := len(groupNameList)
	maxIndex := listLen - 1
	startIndex := page * pre_page
	endIndex := startIndex + pre_page
	if listLen == 0 || startIndex > maxIndex {
		return listLen, nodeGroupList
	}
	if endIndex > maxIndex {
		endIndex = maxIndex
	}

	groupNameList = groupNameList[startIndex:endIndex]
	var tmpGroup *models.NodeGroup
	for _,groupAlias := range groupNameList {
		tmpGroup = NodeGroupGetByAlias(groupAlias)
		if tmpGroup != nil {
			nodeGroupList = append(nodeGroupList, tmpGroup)
		}
	}

	return listLen, nodeGroupList
}

// 查询nodegroup
func NodeGroupGetByAlias(alias string) *models.NodeGroup {
	var nodeGroup models.NodeGroup
	key := generateGroupInfoKey(alias)
	groupJson := consul.KvGet(key)

	if groupJson == "" {
		return nil 	
	}

	err := json.Unmarshal([]byte(groupJson), &nodeGroup)
	logger.IfError(err)
	if err != nil {
		return nil
	}
	return &nodeGroup
}

// 生成group信息键值
func generateGroupInfoKey(node_group_alias string) string {
	return NODE_GROUP_INFO_PRE + node_group_alias
}

// 组名称列表
func nodeGroupListGet() []string {
	var groupList []string
	groupList = []string{}
	groupListJson := consul.KvGet(NODE_GROUP_LIST)
	if groupListJson != "" {
		err := json.Unmarshal([]byte(groupListJson), &groupList)
		logger.IfError(err)
	}

	return groupList
}

// 组名称列表添加
func nodeGroupListAdd(node_group_alias string) (bool, error) {
	groupList := nodeGroupListGet()

	// 是否存在
	if checkInArray(node_group_alias, groupList) {
		return true, nil
	}

	groupList = append(groupList, node_group_alias)
	return consul.KvSetObj(NODE_GROUP_LIST, groupList)
}

// 组名称列表删除
func nodeGroupListDel(node_group_alias string) (bool, error) {
	groupList := nodeGroupListGet()

	for index, val := range groupList {
		if val == node_group_alias {
			groupList = append(groupList[:index], groupList[index+1:]...)
		}
	}

	return consul.KvSetObj(NODE_GROUP_LIST, groupList)
}

// 检测元素是否在数组中[]string
func checkInArray(item string, list []string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}

	return false
}

func NodeGroupCount() int64{
	node_group_model := new(models.NodeGroup)
	total, _ := node_group_model.CountNodeGroup()
	return total
}
