package graceful

import "sync"

var (
	waitJob sync.WaitGroup
)

// 添加一个任务等待（每当有一个新任务进入执行时执行）
func AddJob() {
	waitJob.Add(1)
}

// 减去一个任务等待（任务执行完执行）
func FinishJob() {
	waitJob.Done()
}

// 等待当前任务执行完成
func WaitRunningJob() {
	waitJob.Wait()
}
