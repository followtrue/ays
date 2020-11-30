package consul

import (
	"github.com/hashicorp/consul/api"
	"ays/src/modules/config"
	"ays/src/modules/logger"
	"os"
	"strconv"
)

var ConsulClient *api.Client
var ConsulSession string

func InitClient(config *config.Config) *api.Client {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = config.CONSUL.Address
	ConsulClient, err := api.NewClient(consulConfig)
	logger.IfError(err)

	return ConsulClient
}

// 程序退出时调用
func ReleaseClient() {
	DestroySession(ConsulSession)
}

// 创建当前进程持有的consul session
func CreateSession() string {
	pid := os.Getpid()
	se := &api.SessionEntry{
		Name: "ays_session_"+strconv.Itoa(pid),
	}
	sessionId, _, err := ConsulClient.Session().Create(se, &api.WriteOptions{})
	logger.IfError(err)
	return sessionId
}

// 销毁session
func DestroySession(sessionId string)  {
	ConsulClient.Session().Destroy(sessionId, &api.WriteOptions{})
}