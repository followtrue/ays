package tools

import (
	"fmt"
	"github.com/gin-gonic/gin/json"
	"github.com/mikemintang/go-curl"
	"path/filepath"
	"time"
	"ays/src/modules/config"
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

//获取当前月每日数组
func GetMonthDays(year int, month int) (days []string) {
	//year, month, _ := time.Now().Date()
	t := time.Date(year, time.Month(month + 1), 0, 0, 0, 0, 0, time.UTC)
	for day := 1; day <= t.Day(); day++ {
		date := time.Now().Format("2006-01-") + fmt.Sprintf("%02d", day)
		days = append(days, string(date))
	}
	return days
}

func GetConf()  (*config.Config, error){
	var err error
	ConfDir = "/etc/ays/conf"
	EnvDir = filepath.Join(ConfDir, "env")

	Config, err = config.Read(EnvDir)

	return Config, err
}

//截取字符串 start 起点下标 length 需要截取的长度
func Substr(str string, start int, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

func SendEmail(emailParams map[string]interface{}, host string) {
	req := curl.NewRequest()
	url := host + "/ays/v1/job/dispatch"

	req.SetHeaders(map[string]string{
		"Content-Type":"application/json",
	})
	params ,_ := json.Marshal(emailParams)
	strParams  := string(params)
	req.SetPostData(map[string]interface{}{
		"job_alias" : "ays_email_send",
		"params" : strParams,
	})
	req.Send(url, "POST")
}

func SendSms(smsParams map[string]interface{}, host string) {
	req := curl.NewRequest()
	url := host + "/ays/v1/job/dispatch"

	req.SetHeaders(map[string]string{
		"Content-Type":"application/json",
	})
	params ,_ := json.Marshal(smsParams)
	strParams  := string(params)
	req.SetPostData(map[string]interface{}{
		"job_alias" : "ays_sms_send",
		"params" : strParams,
	})
	req.Send(url, "POST")
}