package app

import (
	"fmt"
	"ays/src/models"
	"ays/src/modules/config"
	"ays/src/modules/consul"
	"ays/src/modules/tools"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	// AppDir 应用根目录
	AppDir string // 应用根目录
	// ConfDir 配置文件目录
	ConfDir string // 配置目录
	// 环境配置目录
	EnvDir string
	// LogDir 日志目录
	//LogDir string // 日志目录
	// Setting 应用配置
	Config *config.Config // 应用配置
)

// 环境变量 初始化
func InitEnv() {
	var err error
	AppDir, err = tools.WorkDir()
	if err != nil {
		fmt.Println(err.Error())
	}
	ConfDir = "/etc/ays/conf"
	EnvDir = filepath.Join(ConfDir, "env")

	Config, err = config.Read(EnvDir)
	if err != nil {
		fmt.Println("读取应用配置失败", err.Error())
	}
}

// 初始化consul连接
func InitConsul()  {
	consul.ConsulClient = consul.InitClient(Config)
	consul.ConsulSession = consul.CreateSession()
}

// 程序退出时调用
func ReleaseConsul()  {
	consul.ReleaseClient()
}

// 数据库 初始化
func InitDb()  {
	models.DB = models.CreateDb(Config)
}

// 日志文件路径不存在时创建
func InitLog() {
	// 手动记录
	//logPath := Config.AYS_LOG
	//if !tools.FileExist(logPath) {
	//	writeFile([]byte("init ays log"), logPath)
	//}
	//logger.InitLogger()

	// 重定向标准输出、标准错误
	accPath := Config.ACCESS_LOG
	if !tools.FileExist(accPath) {
		writeFile([]byte("init access log"), accPath)
	}
	accFile, err := os.OpenFile(accPath, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	os.Stdout = accFile
	os.Stderr = accFile
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