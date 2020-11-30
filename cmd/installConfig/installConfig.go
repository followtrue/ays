package main

import (
	"fmt"
	"ays/src/libs/env"
	"ays/src/modules/tools"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func main()  {
	apd           := kingpin.New("ays_install_config", "ays install config")
	manager       := apd.Command("manager", "install manager config")
	managerENV    := manager.Flag("env", "manager env: local | uat | gray | pro | pronew | uatnew").Short('e').Default("uat").String()
	//managerIp     := manager.Flag("ip", "manager listen ip").Short('p').Default("127.0.0.1").String()
	//managerPort   := manager.Flag("port", "manager listen port").Short('P').Default("18080").String()
	node          := apd.Command("node", "install node config")
	nodeENV       := node.Flag("env", "node env: local | uat | gray | pro").Short('e').Default("uat").String()

	switch kingpin.MustParse(apd.Parse(os.Args[1:])) {
	case manager.FullCommand():
		checkENV(*managerENV)
		installConfigPub(*managerENV)
		//InstallConfigConsul(*managerIp, *managerPort) // 改用定时扫描，不适用consul监听通知
	case node.FullCommand():
		checkENV(*nodeENV)
		installConfigPub(*nodeENV)
	}
}

// 检查env选项是否正确
func checkENV(env string)  {
	envMap := map[string]int{
		"local": 1,
		"uat": 1,
		"gray": 1,
		"pro": 1,
		"pronew": 1,
		"uatnew": 1,
	}

	if _,ok := envMap[env]; !ok {
		fmt.Println("env not in: local | uat | gray | pro | pronew | uatnew")
		os.Exit(1)
	}
}

// manager和node配置共同部分
func installConfigPub(env string)  {
	InstallLog()
	InstallConfigEnv(env)
	InstallConfigLogrotate()
	installRocketmqCpp()
}

// 安装rocketmq-client-cpp
func installRocketmqCpp() {
	var fromPath, toPath string
	// .a
	installFile("config/rocketmq_cpp/librocketmq.a","/usr/local/lib64/rocketmq/librocketmq.a")
	// .so
	installFile("config/rocketmq_cpp/librocketmq.so","/usr/local/lib64/rocketmq/librocketmq.so")
	// .h
	hDir := "config/rocketmq_cpp/rocketmq"
	hToDir := "/usr/local/include/rocketmq"
	fileList, err := env.AssetDir(hDir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, file := range fileList {
		fromPath = filepath.Join(hDir, file)
		toPath = filepath.Join(hToDir, file)
		installFile(fromPath , toPath)
	}

	err = writeFile([]byte("/usr/local/lib64/rocketmq"), "/etc/ld.so.conf.d/rocketmq.conf")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	lna := exec.Command("ln", "-s", "/usr/local/lib64/rocketmq/librocketmq.a", "/lib64/librocketmq.a")
	if _, err := lna.Output(); err != nil {
		fmt.Println(err.Error())
	}

	lnso := exec.Command("ln", "-s", "/usr/local/lib64/rocketmq/librocketmq.so", "/lib64/librocketmq.so")
	if _, err := lnso.Output(); err != nil {
		fmt.Println(err.Error())
	}

	reloadConfig := exec.Command("ldconfig")
	if _, err := reloadConfig.Output(); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("rocketmq配置 安装完成")
}

func InstallLog()  {
	path := "/tmp/log/ays"
	if !tools.FileExist(path) {
		createDirIfNotExists(path)
	}
	cmd := exec.Command("chown", "-R", "www:www", path)
	if _, err := cmd.Output(); err != nil {
		fmt.Println(err.Error())
	}
}

// 安装env 配置文件
// env [local | uat | gray | pro]
func InstallConfigEnv(env string) {
	// 安装env
	envPath := "/etc/ays/conf/env/env.json"
	fromPath := "config/env/"+env+".json"

	installFile(fromPath, envPath)

	if tools.FileExist(envPath) {
		fmt.Println("env 安装成功")
	} else {
		fmt.Println("env 安装失败")
	}
}

// 安装consul配置
func InstallConfigConsul(ip string, port string)  {
	consulPath := "/opt/consul/conf/watch_node_list.json"
	consulData, err := env.Asset("config/consul_watch/watch_node_list.json")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	consulStr := string(consulData)
	consulStr = strings.Replace(consulStr, "ays_ip", ip, 1)
	consulStr = strings.Replace(consulStr, "ays_port", port, 1)

	writeFile([]byte(consulStr), consulPath)

	if tools.FileExist(consulPath) {
		fmt.Println("consul配置 安装成功")
	} else {
		fmt.Println("consul配置 安装失败")
	}
}

// 安装日志切割logrotate配置
func InstallConfigLogrotate() {
	logrotatePath := "/etc/logrotate.d/ays"
	fromPath := "config/logrotate.d/ays"

	installFile(fromPath, logrotatePath)

	if tools.FileExist(logrotatePath) {
		lnso := exec.Command("chmod", "0644", logrotatePath)
		if _, err := lnso.Output(); err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("logrotate配置 安装成功")
	} else {
		fmt.Println("logrotate配置 安装失败")
	}
}

// 检测目录是否存在
func createDirIfNotExists(path string) {
	tmpPath := string(os.PathSeparator)
	pathArr := strings.Split(path, string(os.PathSeparator))

	for _, value := range pathArr {
		tmpPath = filepath.Join(tmpPath, value)
		if tools.FileExist(tmpPath) {
			continue
		}
		err := os.Mkdir(tmpPath, 0755)
		if err != nil {
			fmt.Println(fmt.Sprintf("创建目录失败:%s", err.Error()))
		}
	}
}

// 安装文件
func installFile (fromPath string, toPath string) error {
	fromData, err := env.Asset(fromPath)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	err = writeFile(fromData, toPath)

	return err
}

// 写文件
func writeFile (data []byte, toPath string) error {
	dir := path.Dir(toPath)
	createDirIfNotExists(dir)

	err := ioutil.WriteFile(toPath, data, 0755)
	if err != nil {
		fmt.Println(err.Error())
	}

	if tools.FileExist(toPath) {
		fmt.Println("已安装文件："+toPath)
	}

	return err
}