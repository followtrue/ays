package timer

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/consul"
	jobModules "gitlab.keda-digital.com/kedadigital/ays/src/modules/job"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gopkg.in/robfig/cron.v2"
	"os"
)

const (
	TIMER_LIST_NAME = "ays_timer_list"
	LOCK_TIMER_LIST = "ays_timer_list_lock"
)

var (
	timer *cron.Cron
	timerJobList map[string]TimerJob
)

type TimerJob struct {
	ManagerId int
	UpdateTime string
	TimerId cron.EntryID
}

// 初始化计划任务模块
func InitCron() {
	timer = cron.New()
	timer.Start()
	timerJobList = map[string]TimerJob{} // 定时任务列表
}

// 删除本manager启动的所有timer记录，正在执行的会等待执行完成
//func LocalDelAllTimer() {
//	var pair *api.KVPair
//	var ok bool
//	for {
//		if pair, ok = GetTimerLock(); ok {
//			fmt.Println(fmt.Sprintf("???release all timer job pid:%d", GetManagerId()))
//			timerJobList = map[string]TimerJob{}
//			SaveTimerJobList()
//			break
//		}
//	}
//
//	ReleaseTimerLock(pair)
//}

// 新增Or更新定时任务
func UpdateTimerJobList(job models.Job) int {
	// 在列表中的检测是否需要更新
	tmpList := timerJobList
	for jobAlias, timerJob := range tmpList {
		if jobAlias == job.JobAlias {
			if timerJob.UpdateTime != job.UpdatedAt.String() {
				fmt.Println(fmt.Sprintf("====update timer job alias:%v time:%v job time:%v", jobAlias, timerJob.UpdateTime, job.UpdatedAt.String()))
				combineStopTimerJob(jobAlias, timerJob.TimerId)
				combineStartTimerJob(job)
				return 1
			//} else {
			//	fmt.Println(fmt.Sprintf("====no need add timer job alias:%v pid:%d", jobAlias, GetManagerId()))
			} else {
				return 0
			}
		}
	}

	// 不在列表中的新增
	combineStartTimerJob(job)
	fmt.Println(fmt.Sprintf("====add timer job alias:%v job time:%v", job.JobAlias, job.UpdatedAt.String()))
	return 1
}

// 复合启动定时
func combineStartTimerJob(job models.Job) {
	timerId, err := startTimerJob(job)
	logger.IfError(err)
	if err == nil {
		addJob(timerId, job)
		fmt.Println(fmt.Sprintf("++++add timing job:%s success timer_id:%v pid:%d", job.JobAlias, timerId, GetManagerId()))
	} else {
		fmt.Println(fmt.Sprintf("----add timing job:%s failed reason:%s pid:%d", job.JobAlias, err.Error(), GetManagerId()))
	}
}

// 复合停止定时任务
func combineStopTimerJob(jobAlias string, timerId cron.EntryID)  {
	stopTimerJob(timerId)
	delJob(jobAlias)
	fmt.Println(fmt.Sprintf("????del timing job:%s timer_id:%v", jobAlias, timerId))
}

// 与开启的job列表对比，关闭不存在的定时任务
func CheckStopTimerJob(jobAliasList map[string]int) int {
	editTimes := 0
	tmpList := timerJobList
	for jobAlias, timerJob := range tmpList {
		if _, ok := jobAliasList[jobAlias]; !ok {
			// 仅关闭本进程运行的定时任务
			if timerJob.ManagerId == GetManagerId() {
				stopTimerJob(timerJob.TimerId)
				delJob(jobAlias)
				editTimes = editTimes + 1
			}
		}
	}
	return editTimes
}

// 启动定时
func startTimerJob(job models.Job) (cron.EntryID, error) {
	return timer.AddFunc(job.Timing, func() {
		jobModules.Dispatch("", &job)
	})
}

// 停止定时
func stopTimerJob(entryID cron.EntryID) {
	timer.Remove(entryID)
}

// 本进程ID
func GetManagerId() int {
	return os.Getpid()
}

// 添加timer记录
func addJob(timerId cron.EntryID, job models.Job) {
	timerJobList[job.JobAlias] = TimerJob{
		ManagerId: GetManagerId(),
		UpdateTime: job.UpdatedAt.String(),
		TimerId: timerId,
	}
}

// 删除timer记录
func delJob(jobAlias string) {
	delete(timerJobList, jobAlias)
}

// 获取定时任务列表
func GetTimerJobList() map[string]TimerJob {
	// 获取consul中记录的定时任务列表
	var tmpList map[string]TimerJob
	jsonList := consul.KvGet(TIMER_LIST_NAME)
	if jsonList == "" {
		jsonList = "{}"
	}
	json.Unmarshal([]byte(jsonList), &tmpList)

	// 对比真实定时任务列表，剔除不存在的任务
	timerJobList = checkTimerActive(tmpList)
	fmt.Println(fmt.Sprintf("timer list len: %d", len(timerJobList)))
	return timerJobList
}

// 删除不在运行的任务，删除consul列表没记录的任务
func checkTimerActive(timerJobList map[string]TimerJob) map[string]TimerJob {
	tmpList := timerJobList
	entryIdsActive := getEntryIds() // 有效的、正在运行的EntryID列表
	entryIds := map[cron.EntryID]cron.EntryID{} // timerJobList中的EntryID列表

	// 删除无效的定时任务
	for jobAlias, timerJob := range tmpList {
		entryIds[timerJob.TimerId] = timerJob.TimerId
		if _, ok := entryIdsActive[timerJob.TimerId]; !ok {
			delete(timerJobList, jobAlias)
		}
	}

	// 删除正在运行，但是不在timerJobList中的任务
	for entryID := range entryIdsActive {
		if _, ok := entryIds[entryID]; !ok {
			timer.Remove(entryID)
		}
	}

	return timerJobList
}

// 获取当前定时任务列表的EntryID列表
func getEntryIds() map[cron.EntryID]cron.EntryID {
	entryIDs := map[cron.EntryID]cron.EntryID{}
	entries := timer.Entries()
	for _, entry := range entries {
		entryIDs[entry.ID] = entry.ID
	}

	return entryIDs
}

// 修改定时任务列表
func SaveTimerJobList() {
	fmt.Println(fmt.Sprintf("job list len:%v", len(timerJobList)))
	res, err := consul.KvSetObj(TIMER_LIST_NAME, timerJobList)
	fmt.Println(fmt.Sprintf("????save timing list res:%v", res))
	if err != nil {
		fmt.Println(fmt.Sprintf("----save timing list err:%s", err.Error()))
	}
}

// 获取锁
func GetTimerLock() (*api.KVPair, bool) {
	pair := consul.GetKvPair(LOCK_TIMER_LIST)
	if pair == nil {
		pair = consul.CreateKvPair(LOCK_TIMER_LIST, "1")
	}
	pair.Session = consul.ConsulSession
	res := consul.LockKVPair(pair)
	fmt.Println(fmt.Sprintf("????add lock res:%v", res))
	return pair, res
}

// 释放锁
func ReleaseTimerLock(pair *api.KVPair) bool {
	if pair == nil {
		return true
	}

	res := consul.ReleaseKVPair(pair)
	fmt.Println(fmt.Sprintf("????del lock res:%v", res))
	return res
}