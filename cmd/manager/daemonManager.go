package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spkinger/daemon"
	"ays/src/models"
	"ays/src/modules/app"
	"ays/src/modules/graceful"
	"ays/src/modules/load_data"
	"ays/src/modules/logger"
	"ays/src/modules/register"
	"ays/src/modules/timer"
	"ays/src/modules/tools"
	"ays/src/routers"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const MANAGER_NAME = "aysmanager"
const MANAGER_VERSION = "v1.2.11"

var (
	slaveList []*exec.Cmd
	sockFdPub *os.File
	managerWillExit bool
)

func main() {
	managerWillExit = false
	service       := daemonService()
	apd           := kingpin.New(MANAGER_NAME, "ays manager")
	install       := apd.Command("install", "install manager before manager start")
	installIp     := install.Flag("ip", "manager listen ip").Short('p').Default("127.0.0.1").String()
	installPort   := install.Flag("port", "manager listen port").Short('P').Default("18080").String()
	remove        := apd.Command("remove", "remove manager")
	update        := apd.Command("update", "update manager's config")
	updateIp      := update.Flag("ip", "manager listen ip").Short('p').Default("127.0.0.1").String()
	updatePort    := update.Flag("port", "manager listen port").Short('P').Default("18080").String()
	start         := apd.Command("start", "manager start daemon")
	run           := apd.Command("run", "manager run")
	runIp         := run.Flag("ip", "manager master set ip").Short('p').Default("127.0.0.1").String()
	runPort       := run.Flag("port", "manager master set port").Short('P').Default("18080").String()
	runSlave      := apd.Command("run_slave", "master run slave")
	runTest       := apd.Command("debug", "master run test")
	runTestIp     := runTest.Flag("ip", "manager test set ip").Short('p').Default("127.0.0.1").String()
	runTestPort   := runTest.Flag("port", "manager test set port").Short('P').Default("18080").String()
	stop          := apd.Command("stop", "manager stop")
	restart       := apd.Command("restart", "manager restart")
	status        := apd.Command("status", "manager stop")
	version       := apd.Command("version", "manager version")
	loadData      := apd.Command("loaddata", "load data from database")

	// 异常崩溃处理
	defer exceptionHandle()

	switch kingpin.MustParse(apd.Parse(os.Args[1:])) {
	// 初始化manager配置,更新manager配置
	case install.FullCommand():
		service.SetUser("www")
		service.SetGroup("www")
		msg, err := service.Install("run", "-p", *installIp, "-P", *installPort)
		printAndExit(err)
		fmt.Println(msg)
	// 更新manager
	case update.FullCommand():
		msg, err := service.Remove()
		printAndExit(err)
		fmt.Println(msg)

		service.SetUser("www")
		service.SetGroup("www")
		msg, err = service.Install("run", "-p", *updateIp, "-P", *updatePort)
		printAndExit(err)
		fmt.Println(msg)
	// 卸载
	case remove.FullCommand():
		msg, err := service.Remove()
		printAndExit(err)
		fmt.Println(msg)
	// 执行manager master
	case run.FullCommand():
		var err error
		// 初始化master
		masterInit()
		// 注册主进程退出执行的内容
		OnMasterProcessExit()
		// 让子进程共用sock fd
		port, _   := strconv.Atoi(*runPort)
		_, sockFdPub, err = tools.PubSockFd(*runIp, port)
		printAndExit(err)
		bootSlave(sockFdPub)
		select {}
	// 执行manager slave
	case runSlave.FullCommand():
		runManagerSlave()
	case runTest.FullCommand():
		port, _ := strconv.Atoi(*runTestPort)
		runManagerTest(*runTestIp, port)
	// 启动manager
	case start.FullCommand():
		msg, err := service.Start()
		printAndExit(err)
		fmt.Println(msg)
	// 停止manager
	case stop.FullCommand():
		msg, err := service.Stop()
		printAndExit(err)
		fmt.Println(msg)
	// 重启
	case restart.FullCommand():
		msg, err := service.Stop()
		printAndExit(err)
		fmt.Println(msg)
		msg, err = service.Start()
		printAndExit(err)
		fmt.Println(msg)
	case status.FullCommand():
		msg, err := service.Status()
		printAndExit(err)
		fmt.Println(msg)
	case version.FullCommand():
		println("ays manager version:"+MANAGER_VERSION)
	case loadData.FullCommand():
		loadDataBase()
	}
}

// 运行实体
func runManagerSlave() {
	app.InitEnv()
	app.InitDb()
	app.InitConsul()
	timer.InitCron()
	app.InitLog()
	OnSlaveProcessStart()
	OnSlaveProcessExit()

	file := os.NewFile(3, "")
	routers.Serve(file)

	// 防止http server退出时进程结束
	select{}
}

// 测试执行
func runManagerTest(ip string, port int) {
	app.InitEnv()
	app.Config.ENV = "local"
	app.InitDb()
	app.InitConsul()
	timer.InitCron()
	app.InitLog()
	OnSlaveProcessStart()
	OnSlaveProcessExit()

	_, sockFd, err := tools.PubSockFd(ip, port)
	printAndExit(err)
	routers.Serve(sockFd)

	defer app.ReleaseConsul()
	// 防止http server退出时进程结束
	select{}
}

// 返回daemon控制器
func daemonService() daemon.Daemon {
	service, err := daemon.New(MANAGER_NAME, "Dispatch center")
	printAndExit(err)
	return service
}

func errPrint(err error) {
	if err != nil {
		fmt.Println(err.Error())
	}
}

func printAndExit(err error)  {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// 启动时执行的任务
func OnSlaveProcessStart() {
	// 定时扫描node列表,对未watch的node进行监听
	go func() {
		for {
			if managerWillExit {
				logger.Error(fmt.Sprintf("node watch circle closed pid:%d ---------------", timer.GetManagerId()))
				break
			}
			register.WatchNodeList()
			time.Sleep(time.Duration(20)*time.Second)
		}
	}()

	// 定时循环任务列表，并更新定时任务
	go func() {
		for {
			if managerWillExit {
				logger.Error(fmt.Sprintf("timer job circle closed pid:%d ---------------", timer.GetManagerId()))
				break
			}
			CheckAndBootTimerJob()
			time.Sleep(time.Duration(60)*time.Second)
		}
	}()
}

// 注册进程退出时执行的内容
func OnSlaveProcessExit()  {
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case <-interrupt:
				beforeExit()
				os.Exit(0)
			}
		}
	}()
}

// 注册主进程退出执行内容
func OnMasterProcessExit() {
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR2)
		for {
			sig := <-interrupt

			switch sig {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				// 退出子进程
				killSlave(slaveList)
				// 退出主进程
				os.Exit(0)
			case syscall.SIGUSR2:
				logger.Error("in reload------------------------")
				// 关闭旧的子进程
				slaves := slaveList
				slaveList = []*exec.Cmd{}
				// 启动新的子进程
				go func() {
					bootSlave(sockFdPub)
				}()
				go func() {
					killSlave(slaves)
				}()
			}
		}
	}()
}

// 进程退出时执行的内容
func beforeExit()  {
	logger.Error("manager slave begin exit------------------------")
	// 标记定时任务不再循环拉取
	managerWillExit = true
	// 退出http服务
	routers.StopServe(context.Background())
	// 标记本manager退出监听node节点
	register.UnwatchLocalWatchList()
	//// 释放掉定时任务
	//timer.LocalDelAllTimer()
	// 释放consul的session
	app.ReleaseConsul()
	// 等待未执行完成的任务
	logger.Error("manager on exiting wait running------------------------")
	graceful.WaitRunningJob()
	logger.Error("manager on exiting has finished------------------------")
}

// 运行master进程
// master主要用于控制真正的业务实体slave的平滑重启--reload
// restart时master和salve一起重启
func bootSlave(sockFd *os.File) {
	logger.Error(fmt.Sprintf("manager file path: %s", os.Args[0]))
	cmd := exec.Command(os.Args[0], "run_slave")
	cmd.ExtraFiles = []*os.File{sockFd}
	err := cmd.Start()
	if err != nil {
		logger.Error("graceful start err:")
		logger.Error(err.Error())
	}

	// 存储入子进程列表
	slaveList = append(slaveList, cmd)

	// 等待子进程退出
	err = cmd.Wait()
	if err != nil {
		logger.Error("manager slave graceful wait err:", err)
		logger.Error("restart manager slave ----------")
		bootSlave(sockFd)
	}
}

func masterInit()  {
	app.InitEnv()
	app.InitLog()
}

// 主进程杀掉子进程
func killSlave(slaves []*exec.Cmd) {
	for _, cmd :=range slaves {
		logger.Error("master for kill child:", cmd.Process.Pid)
		cmd.Process.Signal(os.Interrupt)
	}
}

// 检测并启动定时任务
func CheckAndBootTimerJob() {
	editTimes := 0
	// 更新进程中的定时任务列表
	timer.GetTimerJobList()

	// 遍历任务，将定时任务写入定时任务列表并启动，并更新变动的任务
	var job models.Job
	jobs := register.JobListGet()
	activeJobAliasList := map[string]int{} // 开启的任务的代称列表
	for _, jobJson := range jobs {
		err := json.Unmarshal([]byte(jobJson), &job)
		if err != nil {
			errPrint(err)
			continue
		}

		if job.IsTiming() && job.Status == 1 {
			activeJobAliasList[job.JobAlias] = 1
			editTimes = editTimes + timer.UpdateTimerJobList(job)
		}
	}

	// 检测是否有需要停止的定时
	editTimes = editTimes + timer.CheckStopTimerJob(activeJobAliasList)

	// 定时任务列表写入consul
	if editTimes > 0 {
		timer.SaveTimerJobList()
	}
}

func loadDataBase() {
	app.InitEnv()
	app.InitDb()
	app.InitConsul()
	app.InitLog()
	load_data.LoadData()
}

func exceptionHandle() {
	if r := recover(); r != nil {
		logger.Error(fmt.Sprintf("ays manager crashed exit pid:%d", timer.GetManagerId()), r)
		beforeExit()
	}
}