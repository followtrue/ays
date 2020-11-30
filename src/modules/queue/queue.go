package queue

import (
	"encoding/json"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/consul"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/rpc/client"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/rpc/proto"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"math/rand"
)

const (
	QUEUE_LIST_NAME = "ays_queue_list"
	QUEUE_JOB_LIST_NAME = "ays_queue_job_list"
)

type Queue struct {
	Ip string
	Port int
}

type QueueList map[string]Queue // [队列topic]Queue
type QueueJobItem struct {
	JobList []string // 任务alias列表
}
type QueueJobList map[string]QueueJobItem // [node group alias]QueueJobItem

// 返回队列名称数组
func (queueList QueueList) ToQueueNameList() []string {
	list := make([]string, 0)
	for queueName := range queueList {
		list = append(list, queueName)
	}

	return list
}

// queue列表添加队列
func AddQueue(ip string, port int, groupAlias string) {
	if !QueueJobEmpty(groupAlias) {
		queueName := tools.GetQueueName(ip, port)
		if !QueueExists(queueName) {
			// 队列列表添加队列
			queueListAppend(queueName, Queue{
				Ip: ip,
				Port: port,
			})
		}
	}
}

// queue列表删除队列
func DelQueue(ip string, port int) {
	queueName := tools.GetQueueName(ip, port)
	if QueueExists(queueName) {
		// 队列列表删除队列
		queueListPop(queueName)
	}
}

// 添加队列并启动监听
func AddJob(queueList QueueList, jobAlias string, groupAlias string) error {
	// 队列任务列表追加队列下的任务
	QueueJobAppend(queueList, jobAlias, groupAlias)

	// 队列不存在时添加队列并监听
	for queueName, queue := range queueList {
		err := queueListAppend(queueName, queue)
		if err != nil {
			return err
		}

		// rpc通知node监听队列
		go func() {
			client.Exec(queue.Ip, queue.Port, &rpc.JobRequest{
				Id: int64(rand.Int()),
				Type: int32(2),
				Command: queueName,
			})
		}()
	}

	return nil
}

// 判断是否还有在使用队列的job，没有才删除
func DelJob(queueList QueueList, jobAlias string, groupAlias string) error {
	QueueJobDel(jobAlias, groupAlias)

	// 没有job使用队列，直接删除队列
	if QueueJobEmpty(groupAlias) {
		for queueName := range queueList {
			queueListPop(queueName)
		}
	}

	return nil
}

// 队列列表追加队列
func queueListAppend(key string, queue Queue) error {
	if QueueExists(key) {
		return nil
	}

	queueList := GetQueueList()
	queueList[key] = queue

	_, err := consul.KvSetObj(QUEUE_LIST_NAME, queueList)
	return err
}

// 队列列表弹出队列
func queueListPop(key string) error {
	if !QueueExists(key) {
		return nil
	}

	queueList := GetQueueList()
	delete(queueList, key)

	_, err := consul.KvSetObj(QUEUE_LIST_NAME, queueList)
	return err
}

// 获取队列列表
func GetQueueList() QueueList {
	queueList := QueueList{}
	listJson := consul.KvGet(QUEUE_LIST_NAME)
	if listJson == "" {
		return queueList
	}

	err := json.Unmarshal([]byte(listJson), &queueList)
	logger.IfError(err)

	return queueList
}

// 判断队列是否存在
func QueueExists(queueName string) bool {
	queueList := GetQueueList()

	_, ok := queueList[queueName]
	return ok
}

// 添加job
func QueueJobAppend(queueList QueueList, jobAlias string, groupAlias string) {
	list := GetQueueJobList()
	queueJobItem := list[groupAlias]

	if !QueueJobEmpty(groupAlias) {
		if !CheckJobInQueueJob(jobAlias, queueJobItem.JobList) {
			queueJobItem.JobList = append(queueJobItem.JobList, jobAlias)
			list[groupAlias] = queueJobItem
		}
	} else {
		list[groupAlias] = QueueJobItem{
			JobList: []string{jobAlias},
		}
	}

	consul.KvSetObj(QUEUE_JOB_LIST_NAME, list)
}

// 删除job
func QueueJobDel(jobAlias string, groupAlias string) {
	list := GetQueueJobList()
	queueJobItem := list[groupAlias]

	if !QueueJobEmpty(groupAlias) {
		if CheckJobInQueueJob(jobAlias, queueJobItem.JobList) {
			for index, alias := range queueJobItem.JobList {
				if alias == jobAlias {
					queueJobItem.JobList = append(queueJobItem.JobList[:index], queueJobItem.JobList[index+1:]...)
					if len(queueJobItem.JobList) == 0 {
						delete(list, groupAlias)
						consul.KvSetObj(QUEUE_JOB_LIST_NAME, list)
					}
					return
				}
			}
		}
	}
}

// 获取队列job列表
func GetQueueJobList() QueueJobList {
	queueJobList := QueueJobList{}
	listJson := consul.KvGet(QUEUE_JOB_LIST_NAME)
	if listJson == "" {
		return queueJobList
	}

	err := json.Unmarshal([]byte(listJson), &queueJobList)
	logger.IfError(err)
	return queueJobList
}

// 判断队列job是否还有job
func QueueJobEmpty(groupAlias string) bool {
	queueJobList := GetQueueJobList()
	queueJobItem, ok := queueJobList[groupAlias]
	if !ok {
		return true
	}
	return len(queueJobItem.JobList) == 0
}

// 检测job是否在queueJob中
func CheckJobInQueueJob(jobAlias string, jobList []string) bool {
	for _, alias := range jobList {
		if alias == jobAlias {
			return true
		}
	}

	return false
}

// 检测队列名称是否在queueJob中
func CheckQueueInQueueJob(queueName string, queueList []string) bool {
	for _, alias := range queueList {
		if alias == queueName {
			return true
		}
	}

	return false
}