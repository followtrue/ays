package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"strconv"
	"time"
)

// 服务数据
type ServiceData struct {
	ID string
	Name string
	Address string
	Port int
	Meta map[string]string // 存有node_group_alias
	Health string // 健康状态：passing-通过检查，warning-警告，critical-危急，maintenance-维护
}

type ServiceDataList []ServiceData
type ServiceChangeCallBack func(*ServiceData)

const WatchQuit = "watch_quit"

// 注册服务
func Reg(group string, name string, ip string, port int) error {
	check := new(api.AgentServiceCheck)
	check.Name = "check_"+name
	check.TCP = ip+":"+strconv.Itoa(port)
	check.Timeout = "3s"
	check.Interval = "10s"
	check.DeregisterCriticalServiceAfter = "30s" // check失败30秒后删除服务

	reg := new(api.AgentServiceRegistration)
	reg.Name = name
	reg.Address = ip
	reg.Meta = map[string]string{"node_group_alias":group}
	reg.Port = port
	reg.Check = check
	err := ConsulClient.Agent().ServiceRegister(reg)
	logger.IfPanic(err)

	return err
}

// 注销服务
func DelReg(name string) error {
	err := ConsulClient.Agent().ServiceDeregister(name)
	if err != nil {
		logger.Error("node:"+name+" deregister failed", err)
	}
	return err
}

// 监听服务变更
func Watch(name string, tag string, HeartbeatRate int, noHeartbeatMaxTimes int, fn ServiceChangeCallBack) string {
	offlineChannel := make(chan int)
	quitChannel := make(chan int)

	go func() {
		LastIndex := uint64(0)
		serviceLen := 0
		//zeroTimes := 0
		for {
			// 开始监听变更
			services, meta, err := ConsulClient.Health().Service(name, tag, false, &api.QueryOptions{})

			// 监听异常-退出
			logger.IfError(err)
			if err != nil {
				fmt.Println(fmt.Sprintf("----------service:%s watch err:%s-----------", name, err.Error()))
				quitChannel <- 1
				return
			}

			// 最后修改记录不同时，执行回调
			if meta.LastIndex != LastIndex {
				fmt.Println(fmt.Sprintf("-------------change:%s---index:%d--------------", name, int(meta.LastIndex)))
				LastIndex = meta.LastIndex
				for _, service := range services {
					fmt.Println(fmt.Sprintf("-----------node_service--ID:%s--status:%s--------------", service.Service.ID, service.Checks.AggregatedStatus()))
					fn(&ServiceData{
						ID: service.Service.ID,
						Name: service.Service.Service,
						Address: service.Service.Address,
						Port: service.Service.Port,
						Meta: service.Service.Meta,
						Health: service.Checks.AggregatedStatus(),
					})
				}
			}

			// 直接下线
			serviceLen = len(services)
			if serviceLen == 0 {
				offlineChannel <- 1
				return
			}
			// xxxxx可响应service数量为零超过noHeartbeatMaxTime时，认为下线，退出
			//if serviceLen == 0 {
			//	zeroTimes++
			//	if zeroTimes > noHeartbeatMaxTimes {
			//		offlineChannel <- 1
			//		return
			//	}
			//} else {
			//	zeroTimes = 0
			//}

			fmt.Println(fmt.Sprintf("++watch:%s++", name))
			time.Sleep(time.Duration(HeartbeatRate) * time.Second)
		}
	}()

	select {
	case <-offlineChannel:
		fmt.Println(fmt.Sprintf("----------service:%s watch time out-----------", name))
		fn(&ServiceData{
			ID: name,
			Name: "",
			Address: "",
			Port: 0,
			Meta: map[string]string{},
			Health: models.NodeHealth_OFFLINE,
		})
		return WatchQuit
	case <-quitChannel:
		fmt.Println(fmt.Sprintf("----------service:%s watch err quit----------", name))
		return WatchQuit
	}

	return WatchQuit
}

// 返回服务最后一次变更记录的索引号
func LastChangeIndex(name string, tag string) uint64 {
	_, metainfo, err := ConsulClient.Health().Service(name, tag, true, &api.QueryOptions{})
	logger.IfError(err)

	return metainfo.LastIndex
}

// 根据meta查找service
func FindByMeta(metas map[string]string) ([]*api.AgentService) {
	var returnServices []*api.AgentService
	services, err := ConsulClient.Agent().Services()
	if err != nil {
		logger.IfError(err)
		return returnServices
	}

	for metaKey, metaVal := range metas {
		for _, service := range services {
			if val, ok := service.Meta[metaKey]; ok {
				if val == metaVal {
					returnServices = append(returnServices, service)
				}
			}
		}
	}

	return returnServices
}

