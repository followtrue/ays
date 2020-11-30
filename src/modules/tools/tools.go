package tools

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/libs/constant"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// WorkDir 获取程序运行时根目录
func WorkDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	wd := filepath.Dir(execPath)
	if filepath.Base(wd) == "bin" {
		wd = filepath.Dir(wd)
	}

	return wd, nil
}

// 判断文件是否存在及是否有权限访问
func FileExist(file string) bool {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false
	}
	if os.IsPermission(err) {
		return false
	}

	return true
}

// 文件夹不存在创建文件夹
func CreateDir(path string) error {
	if FileExist(path) {
		return nil
	}
	//pathArr := strings.Split(path, string(os.PathSeparator))
	err := os.MkdirAll(path, 0711)

	return err
}

// 文件不存在创建文件
func CreateFile(filePath string) error {
	if FileExist(filePath) {
		return nil
	}

	err := CreateDir(filepath.Dir(filePath))
	if err != nil {
		return err
	}

	_, err = os.Create(filePath)

	return err
}

// 成功返回
func Success(c *gin.Context, data interface{}) {
	code := constant.SUCCESS_CODE
	c.JSON(http.StatusOK, gin.H{
		"code":code,
		"message":"",
		"data":data,
	})
	return
}

// 错误返回
func Error(c *gin.Context, message string) {
	data := [] string{}
	code := constant.ERROR_CODE
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"message": message,
		"data": data,
	})
	return
}

// 生成addr
func GenerateAddr(ip string, port int) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

// 监听端口返回Listener
func GetListener(addr string) (net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// 从sockFD文件获取listener
func GetFileListener(file *os.File) (net.Listener, error) {
	return net.FileListener(file)
}

// master创建子进程共享的sockFd，用于监听端口的交接
func PubSockFd(ip string, port int) (net.Listener, *os.File, error) {
	addr := GenerateAddr(ip, port)
	tcpListener, err := GetListener(addr)
	if err != nil {
		return nil, nil, err
	}

	tcpListen, ok := tcpListener.(*net.TCPListener)
	if !ok {
		return nil, nil, errors.New("listener is not tcp listener")
	}

	sockFd, err := tcpListen.File()
	if err != nil {
		return nil, nil, err
	}

	return tcpListener, sockFd, nil
}

// 复制文件
func CopyFile(orgFile, targetFile string) (res bool, err error) {
	src, err := os.Open(orgFile)
	if err != nil {
		return false, err
	}
	defer src.Close()

	dst, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return false, err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// 获取目录下所有文件的路径
func GetAllFiles(dir string) ([]string, error) {
	var filepaths, tmpPaths []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			tmpPaths, err = GetAllFiles(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}

			filepaths = append(filepaths, tmpPaths...)
		} else {
			filepaths = append(filepaths, filepath.Join(dir, file.Name()))
		}
	}

	return filepaths, nil
}

// 获取queue的topic
const QUEUE_PRE = "ays_queue"
func GetQueueName(ip string, port int) string {
	ip = strings.Replace(ip, ".", "_", -1)
	return fmt.Sprintf("%s_%s_%d", QUEUE_PRE, ip, port)
}

// 检测alias是否是数字、字母、下划线
func CheckAlias(alias string) bool {
	reg, err := regexp.Compile(`\w+`)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	expString := reg.FindString(alias)
	return expString == alias
}