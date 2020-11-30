package register

import (
	"encoding/json"
	"fmt"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/consul"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
)

type Queue struct {
	Name    string
	Ip      string
	Port    int
}

type QueueList map[string]Queue

const (
	QUEUE_LIST_NAME = "ays_topic_list"
	QUEUE_NAME_PRE = "ays_topic"
)

// 生成队列名称
func QueueName(ip string, port int) string {
	return fmt.Sprintf("%s_%s_%d", QUEUE_NAME_PRE, ip, port)
}

// 判断队列是否存在
func QueueExist(ip string, port int) bool {
	_, ok := QueueGet(ip, port)
	return ok
}

// 获取队列struct
func QueueGet(ip string, port int) (*Queue, bool) {
	var list QueueList
	name := QueueName(ip, port)
	listJson := consul.KvGet(QUEUE_LIST_NAME)
	err := json.Unmarshal([]byte(listJson), &list)
	logger.IfError(err)
	if err != nil {
		return nil, false
	}

	queue, ok := list[name]
	return &queue, ok
}