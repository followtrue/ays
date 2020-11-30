package register

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/app"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/consul"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/queue"
	"os"
	"strconv"
	"strings"
	"time"
)

const NODE_LIST_NAME  = "ays_node_list"
const MANAGER_ALIVE_SIGN_PRE = "manager_alive_"

// 生成节点名称(因节点名称会带pid，所以只有节点调用才有效，而且是节点的slave调用)
func GenerateNodeName(ip string, port int) string {
	pid := os.Getpid()
	return fmt.Sprintf("node_%d_%s_%d", pid,strings.Replace(ip, ".", "-", -1), port)
}

// 手动新增节点（API用）
func NodeAdd(ip string, port int, nodeGroupAlias string) error {
	node := new(models.Node)
	exists, err := node.Find(ip, port)
	if exists {
		err := errors.New("节点已存在，不可新增")
		logger.Error(err.Error())
		return err
	}

	// 数据库新增节点记录
	node = new(models.Node)
	node.NodeGroupAlias = nodeGroupAlias
	node.NodeIp = ip
	node.NodePort = port
	node.NodeHealth = api.HealthMaint
	_, err = node.Create()
	if err != nil {
		logger.Error(err.Error())
		return err
	} else {
		return nil
	}
}
// 自动注册节点（执行器节点用）
func NodeRegister(ip string, port int, nodeGroupAlias string) (bool, error) {
	// consul注册节点
	nodeName := GenerateNodeName(ip, port)
	err := consul.Reg(nodeGroupAlias, nodeName, ip, port)
	logger.IfError(err)

	// 更改注册中心node列表，从而启动watch
	NodeListAppend(ip, port)

	// 如果执行器组中有任务需要队列，则加入
	queue.AddQueue(ip, port, nodeGroupAlias)

	return true, err
}

// 注销节点（执行器节点用）
func NodeDelReg(ip string, port int) bool {
	// consul删除节点
	name := GenerateNodeName(ip, port)
	// 执行器列表删除节点
	NodeListPop(ip, port)
	queue.DelQueue(ip, port)
	err := consul.DelReg(name)
	logger.IfError(err)
	if err != nil {
		return false
	} else {
		return true
	}
}

// 删除节点(API用)
func NodeDel(node *models.Node) bool {
	// consul注销
	res := NodeDelReg(node.NodeIp, node.NodePort)

	// 数据库删除
	if res {
		effect, err := node.Delete()
		logger.IfError(err)
		return effect > 0
	}

	return res
}

// 查找节点(API用)
func NodeFind(ip string, port int) *models.Node {
	node := new(models.Node)
	has, err := node.Find(ip, port)
	logger.IfError(err)

	if has {
		return node
	} else {
		return nil
	}
}

type NodeView struct {
	Name string
	Ip string
	Port int
	NodeGroupAlias string
}

// 查找可用节点（调度中心用）
func NodeFindByGroup(node_group_alias string) []NodeView {
	nodeViewList := make([]NodeView, 0)
	services := consul.FindByMeta(map[string]string{"node_group_alias":node_group_alias})

	for _,service := range services {
		nodeViewList = append(nodeViewList, NodeView{
			Name: service.Service,
			Ip: service.Address,
			Port: service.Port,
			NodeGroupAlias: node_group_alias,
		})
	}

	return nodeViewList
}

type NodeViewAPI struct {
	Id int
	Name string
	Ip string
	Port int
	NodeGroupAlias string
	NodeHealth string
}

// 从数据库获取节点列表
func NodeFindByGroupMysql(node_group_alias string) []NodeViewAPI {
	nodeViewList := make([]NodeViewAPI, 0)
	nodeModel := models.Node{}
	nodeList, err := nodeModel.GetNodesByNodeGroupAlias(node_group_alias)
	logger.IfError(err)
	if err != nil {
		return nodeViewList
	}

	for _, node := range nodeList {
		nodeViewList = append(nodeViewList, NodeViewAPI{
			Id: node.Id,
			Name: node.NodeId,
			Ip: node.NodeIp,
			Port: node.NodePort,
			NodeGroupAlias: node_group_alias,
			NodeHealth: node.NodeHealth,
		})
	}

	return nodeViewList
}

// 监控
type Node struct {
	Name string
	Ip string
	Port int
	IsWatch int // 监听节点状态的manager pid，没有时是0
}

type NodeList []Node

// 获取节点列表存入nodeList中
func (nodeList *NodeList) Get() {
	// kv中获取节点列表
	nodeListKey := NODE_LIST_NAME
	listJson := consul.KvGet(nodeListKey)
	json.Unmarshal([]byte(listJson), &nodeList)
}

// 节点列表存入kv
func (nodeList *NodeList) Save() {
	nodeListKey := NODE_LIST_NAME
	consul.KvSetObj(nodeListKey, nodeList)
}

// 节点列表追加新节点(执行器节点调用)
func NodeListAppend(ip string, port int) {
	// kv中获取节点列表
	nodeList := NodeList{}
	nodeList.Get()

	// 已存在的不再追加
	for _, node := range nodeList {
		if node.Ip == ip && node.Port == port {
			return
		}
	}

	// 追加节点
	nodeList = append(nodeList, Node{
		Name: GenerateNodeName(ip, port),
		Ip: ip,
		Port: port,
		IsWatch: 0,
	})
	nodeList.Save()
}

// 节点列表删除节点（执行器节点退出时调用、调度中心删除节点时调用）
func NodeListPop(ip string, port int) {
	// kv中获取节点列表
	nodeList := NodeList{}
	nodeList.Get()

	// 列表中去除
	for index, node := range nodeList {
		if node.Ip == ip && node.Port == port {
			nodeList = append(nodeList[:index], nodeList[index+1:]...)
		}
	}

	// 存入kv
	nodeList.Save()
}

// 记录本manager节点监听的node节点name
var localWatchList map[string]int

// 添加记录本manager节点监听的node节点name
func localWatchListAdd(name string)  {
	if localWatchList == nil {
		localWatchList = map[string]int{}
	}
	localWatchList[name] = 1
}

// 本节点退出时，标记退出监听
func UnwatchLocalWatchList() {
	nodeList := NodeList{}
	nodeList.Get()
	hasChange := false

	for index, node := range nodeList {
		if _, ok := localWatchList[node.Name]; ok {
			nodeList[index].IsWatch = 0
			hasChange = true
		}
	}

	if hasChange {
		nodeList.Save()
	}
}

// 监听节点列表变更(调度中心留给consul调用)
// 说明：consul增加watch_node_list.json变更的脚本后，每次有变更要调用此方法
// 使用中遇到consul未通知的情况，改为定时扫描
func WatchNodeList()  {
	managerAliveSign() // 标记manager自身存活
	nodeList := NodeList{}
	nodeList.Get()
	hasChange := false

	// 新增的添加watch
	tmpList := nodeList
	for index, node := range tmpList {
		if node.IsWatch == 0 || !managerAliveCheck(node.IsWatch) {
			go func(nodeName string) {
				fmt.Println(fmt.Sprintf("--in--node_name:%s ----", nodeName))
				NodeWatch(nodeName)
			}(node.Name)
			tmpList[index].IsWatch = os.Getpid()
			hasChange = true
			localWatchListAdd(node.Name)
		}
	}

	if hasChange {
		nodeList = tmpList
		nodeList.Save()
	}
}

// manager存活标记设置（标记内容为时间戳,标记频率需小于30秒）
func managerAliveSign() {
	managerAliveKey := fmt.Sprintf("%s%d", MANAGER_ALIVE_SIGN_PRE, os.Getpid())
	consul.KvSet(managerAliveKey, fmt.Sprintf("%d", time.Now().Unix()))
}

// manager存活检测
func managerAliveCheck(pid int) bool {
	// pid == self
	if pid == os.Getpid() {
		return true
	}

	// 检测最后存活标记时间距现在是否大于30秒
	now := time.Now().Unix()
	managerAliveKey := fmt.Sprintf("%s%d", MANAGER_ALIVE_SIGN_PRE, pid)
	lastSignStr := consul.KvGet(managerAliveKey)
	lastSignSecond, err := strconv.ParseInt(lastSignStr, 10, 64)
	if err != nil || lastSignSecond + 30 < now {
		return false
	} else {
		return true
	}
}

// 执行节点变更监听(会一直阻塞，直到收到退出channel)
func NodeWatch(name string) {
	quit := make(chan string)

	go func() {
		// node注销时其watch输出quit
		quit <- consul.Watch(name, "", app.Config.CONSUL.WatchRate, app.Config.CONSUL.WatchTimeOutTimes, OnNodeChange)
	}()

	select {
	case <-quit:
		// node注销时自动结束watch
		logger.Warn(fmt.Sprintf("NAME:%s quit watch", name))
	}
}

// 服务状态变更回调方法
func OnNodeChange(data *consul.ServiceData) {
	// 查找节点组
	group_alias := "unknown_group"

	if _, ok := data.Meta["node_group_alias"]; ok {
		nodeGroup := new(models.NodeGroup)
		has, err := nodeGroup.Find(data.Meta["node_group_alias"])
		logger.IfError(err)
		if has {
			group_alias = nodeGroup.NodeGroupAlias
		} else {
			return // 忽略无效数据
		}
	}

	// 将节点信息同步到数据库
	node := new(models.Node)
	node.NodeGroupAlias = group_alias
	node.NodeId = data.ID
	node.NodeIp = data.Address
	node.NodePort = data.Port
	node.NodeHealth = data.Health

	_,err := node.CreateOrUpdate()
	logger.IfError(err)
}