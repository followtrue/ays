package main

import (
	"fmt"
	"github.com/spkinger/daemon"
	"ays/src/modules/app"
	"ays/src/modules/graceful"
	"ays/src/modules/listen"
	"ays/src/modules/logger"
	"ays/src/modules/queue"
	"ays/src/modules/register"
	"ays/src/modules/rpc/server"
	"ays/src/modules/tools"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

const NODE_NAME = "aysnode"
const NODE_VERSION = "V1.1.11"

var (
	sockFdPub *os.File
	localGroupAlias string
	localIp string
	localPort string
	rpcServer *grpc.Server
	slaveList []*exec.Cmd
)

func main() {
	service       := daemonService()
	apd           := kingpin.New(NODE_NAME, "ays node")
	install       := apd.Command("install", "install node before node start")
	installAlias  := install.Flag("group_alias", "node group alias").Short('g').String()
	installIp     := install.Flag("ip", "node listen ip").Short('p').Default("0.0.0.0").String()
	installPort   := install.Flag("port", "node listen port").Short('P').Default("18181").String()
	update        := apd.Command("update", "update node's config")
	updateAlias   := update.Flag("group_alias", "node group alias").Short('g').String()
	updateIp      := update.Flag("ip", "node listen ip").Short('p').Default("0.0.0.0").String()
	updatePort    := update.Flag("port", "node listen port").Short('P').Default("18181").String()
	remove        := apd.Command("remove", "remove node")
	start         := apd.Command("start", "start daemon node")
	run           := apd.Command("run", "node run master")
	runAlias      := run.Flag("group_alias", "node group alias").Short('g').String()
	runIp         := run.Flag("ip", "node listen ip").Short('p').Default("0.0.0.0").String()
	runPort       := run.Flag("port", "node listen port").Short('P').Default("18181").String()
	runSlave      := apd.Command("run_slave", "node run slave")
	runSlaveAlias := runSlave.Flag("group_alias", "node group alias").Short('g').String()
	runSlaveIp    := runSlave.Flag("ip", "node listen ip").Short('p').Default("0.0.0.0").String()
	runSlavePort  := runSlave.Flag("port", "node listen port").Short('P').Default("18181").String()
	runTest       := apd.Command("debug", "master run test")
	runTestAlias := runTest.Flag("group_alias", "node group alias").Short('g').String()
	runTestIp     := runTest.Flag("ip", "node test set ip").Short('p').Default("127.0.0.1").String()
	runTestPort   := runTest.Flag("port", "node test set port").Short('P').Default("18181").String()
	stop          := apd.Command("stop", "node stop")
	restart       := apd.Command("restart", "node restart")
	status        := apd.Command("status", "node stop")
	version       := apd.Command("version", "node version")

	defer exceptionHandle()

	switch kingpin.MustParse(apd.Parse(os.Args[1:])) {
	// 初始化node配置,更新node配置
	case install.FullCommand():
		service.SetUser("www")
		service.SetGroup("www")
		msg, err := service.Install("run", "-g", *installAlias, "-p", *installIp, "-P", *installPort)
		printAndExit(err)
		fmt.Println(msg)
	// 更新
	case update.FullCommand():
		msg, err := service.Remove()
		printAndExit(err)
		fmt.Println(msg)

		service.SetUser("www")
		service.SetGroup("www")
		msg, err = service.Install("run", "-g", *updateAlias, "-p", *updateIp, "-P", *updatePort)
		printAndExit(err)
		fmt.Println(msg)
	// 卸载
	case remove.FullCommand():
		msg, err := service.Remove()
		printAndExit(err)
		fmt.Println(msg)
	// 运行master
	case run.FullCommand():
		var err error
		localGroupAlias = *runAlias
		localIp         = *runIp
		localPort       = *runPort
		// master初始化
		masterInit()
		// 注册主进程退出执行的内容
		OnMasterProcessExit()
		// 主进程中启动子进程
		port, _   := strconv.Atoi(localPort)
		_, sockFdPub, err = tools.PubSockFd(localIp, port)
		printAndExit(err)
		bootSlave(sockFdPub)
		select {}
	// 运行slave
	case runSlave.FullCommand():
		localGroupAlias = *runSlaveAlias
		localIp         = *runSlaveIp
		localPort       = *runSlavePort
		runNodeSlave(localGroupAlias, localIp, localPort)
	// 运行test
	case runTest.FullCommand():
		localGroupAlias = *runTestAlias
		localIp         = *runTestIp
		localPort       = *runTestPort
		runNodeTest(localGroupAlias, localIp, localPort)
	// 启动node
	case start.FullCommand():
		msg, err := service.Start()
		printAndExit(err)
		fmt.Println(msg)
	// 停止node
	case stop.FullCommand():
		msg, err := service.Stop()
		printAndExit(err)
		fmt.Println(msg)
	// 重启
	// 支持restart方法
	// 若使用reload方法，直接service aysnode reload,是通过信号方式实现
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
		println("ays node version:"+NODE_VERSION)
	}
}

func masterInit()  {
	app.InitEnv()
	app.InitLog()
}

// 运行master进程
// master主要用于控制真正的业务实体slave的平滑重启--reload
// restart时master和salve一起重启
func bootSlave(sockFd *os.File) {
	cmd := exec.Command(os.Args[0], "run_slave", "-g", localGroupAlias, "-p", localIp, "-P", localPort)
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
		logger.Error("restart node slave ----------")
		bootSlave(sockFd)
	}
}

// 运行实体（子进程）
func runNodeSlave(groupAlias string, ip string, port string) {
	app.InitEnv()
	app.InitDb()
	app.InitConsul()
	app.InitLog()
	intPort, err := strconv.Atoi(port)
	printAndExit(err)

	// 启动时注册node
	register.NodeRegister(ip, intPort, groupAlias)

	// 检测queue是否存在
	queueName := tools.GetQueueName(ip, intPort)
	if queue.QueueExists(queueName) {
		// mq队列监听启动
		go func() {
			listen.ListenQueue(queueName)
		}()
	}

	// 注册进程退出监听
	OnSlaveProcessExit(ip, intPort)

	// rpc 启动
	file := os.NewFile(3, "")
	listener, err := tools.GetFileListener(file)
	printAndExit(err)
	rpcServer = server.GetGrpcServer()
	server.Start(rpcServer, listener)
}

// 运行测试node
func runNodeTest(groupAlias string, ip string, port string) {
	app.InitEnv()
	app.Config.ENV = "local"
	app.InitDb()
	app.InitConsul()
	app.InitLog()
	intPort, err := strconv.Atoi(port)
	printAndExit(err)

	// 启动时注册node
	register.NodeRegister(ip, intPort, groupAlias)

	// 检测queue是否存在
	queueName := tools.GetQueueName(ip, intPort)
	if queue.QueueExists(queueName) {
		// mq队列监听启动
		go func() {
			listen.ListenQueue(queueName)
		}()
	}

	// 注册进程退出监听
	OnSlaveProcessExit(ip, intPort)

	// rpc 启动
	portInt, _ := strconv.Atoi(port)
	listener, _, err := tools.PubSockFd(ip, portInt)
	printAndExit(err)
	rpcServer = server.GetGrpcServer()
	server.Start(rpcServer, listener)
}

// 返回daemon控制器
func daemonService() daemon.Daemon {
	service, err := daemon.New(NODE_NAME, "Dispatch center")
	printAndExit(err)
	return service
}

func printAndExit(err error)  {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// 注册进程退出时执行的内容
func OnSlaveProcessExit(ip string, port int) {
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			sig := <-interrupt

			switch sig {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				beforeExit()
				// 退出slave
				os.Exit(0)
			}
		}
	}()
}

// 进程退出时执行的内容
func beforeExit() {
	// consul注销本slave
	port, err := strconv.Atoi(localPort)
	if err != nil {
		fmt.Println("node child exit get port err"+err.Error())
	} else {
		exitJob(localIp, port)
	}
	fmt.Println(fmt.Sprintf("node child exit pid:%d", os.Getpid()))
	// 关闭端口监听
	rpcServer.Stop()
	// 释放consul的session
	app.ReleaseConsul()
	// 等待任务执行完成
	logger.Error("node on exiting wait running------------------------")
	graceful.WaitRunningJob()
	logger.Error("node on exiting has finished------------------------")
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
func exitJob(ip string, port int) bool {
	return register.NodeDelReg(ip, port)
}

// 主进程杀掉子进程
func killSlave(slaves []*exec.Cmd) {
	for _, cmd :=range slaves {
		logger.Error("master for kill child:", cmd.Process.Pid)
		cmd.Process.Signal(os.Interrupt)
	}
}

func exceptionHandle() {
	if r := recover(); r != nil {
		logger.Error(fmt.Sprintf("ays node crashed exit pid:%d", os.Getpid()), r)
		beforeExit()
	}
}