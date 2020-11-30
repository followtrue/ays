package load_data

import (
	"ays/src/models"
	"ays/src/modules/consul"
	"ays/src/modules/register"
)

func LoadData()  {
	clearConsul()
	loadnodeGroup()
	loadjob()
}

// 清理consul数据
func clearConsul() {
	// 删除 list_node_group\list_node_list
	consul.KvDel("list_node_group")
	consul.KvDel("list_node_list")
	// 删除 node_group*
	node_group_keys := consul.PreSearchKeys("node_group")
	for _,key := range node_group_keys {
		consul.KvDel(key)
	}
	// 删除 job*
	job_keys := consul.PreSearchKeys("job")
	for _,key := range job_keys {
		consul.KvDel(key)
	}
}

// 导入节点组
func loadnodeGroup() {
	pre_page := 10
	page := 1
	mod := models.NodeGroup{}
	for {
		node_group_list := mod.GetList(pre_page, page, "", "")
		if len(node_group_list) <= 0 {
			break
		}
		for _, node_group := range node_group_list {
			register.NodeGroupAddConsul(node_group)
		}
		page = page + 1
	}
}

// 导入任务
func loadjob() {
	pre_page := 10
	page := 1
	mod := models.Job{}
	for {
		job_list := mod.GetList(pre_page, page, "", "", "")
		if len(job_list) <= 0 {
			break
		}
		for _, job := range job_list {
			register.JobAddConsul(job)
		}
		page = page + 1
	}
}