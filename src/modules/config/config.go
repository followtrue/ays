package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
)

type Config struct {
	AYS_LOG          string
	ACCESS_LOG       string
	ERROR_LOG        string
	DB struct{
		Engine       string
		Host         string
		Port         int
		User         string
		Password     string
		Database     string
		Prefix       string
		Charset      string
		MaxIdleConns int
		MaxOpenConns int
	}
	ENV_OPT struct{
		Local        string
		Uat          string
		Gray         string
		Pro          string
	}
	CONSUL struct{
		Address      string
		WatchTimeOutTimes int
		WatchRate    int
	}
	ENV              string
	NODE_LIST_NAME   string

	MQ struct{
		Group 		string
		Namesrv 	string 
		LogPath 	string
		LogLevel 	string
		LogFilesize	int
		LogFilenum 	int
		PullMaxNum 	int
	}

	OAUTH struct{
		Host 	string
	}

	AYS struct{
		Host 	string
	}
}

// 读取配置
func Read(EnvDir string) (*Config, error) {
	// 读取配置文件
	err := loadFile(EnvDir)
	if err != nil {
		return nil, err
	}

	// 首次读取config
	var s Config
	reloadConfig(&s)

	// 监听config更改
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		reloadConfig(&s)
		fmt.Println("Config file changed:", e.Name)
		fmt.Println("db config:", s.DB.User)
	})

	return &s, nil
}

// 读取配置文件
func loadFile(EnvDir string) error {
	viper.AutomaticEnv()
	goPath := viper.GetString("GOPATH")
	goPathMul := strings.Split(goPath, ":")
	for _, goPathSingle := range goPathMul {
		viper.AddConfigPath(filepath.Join(goPathSingle, "src/gitlab.keda-digital.com/kedadigital/ays/config/env"))
	}
	viper.AddConfigPath(EnvDir)
	viper.SetConfigName("env")
	err := viper.ReadInConfig() // Find and read the config file
	return err
}

// 重新读取config
func reloadConfig(config *Config) {
	config.ENV = viper.GetString("env")
	config.NODE_LIST_NAME = viper.GetString("node_list_name")

	config.ENV_OPT.Local = "local"
	config.ENV_OPT.Uat = "uat"
	config.ENV_OPT.Gray = "gray"
	config.ENV_OPT.Pro = "pro"

	config.DB.Engine = viper.GetString("db.engine")
	config.DB.Host = viper.GetString("db.host")
	config.DB.Port = viper.GetInt("db.port")
	config.DB.User = viper.GetString("db.user")
	config.DB.Password = viper.GetString("db.password")
	config.DB.Database = viper.GetString("db.database")
	config.DB.Prefix = viper.GetString("db.prefix")
	config.DB.Charset = viper.GetString("db.charset")
	config.DB.MaxIdleConns = viper.GetInt("db.max_idle_conns")
	config.DB.MaxOpenConns = viper.GetInt("db.max_open_conns")

	config.CONSUL.Address = viper.GetString("consul.address")
	config.CONSUL.WatchTimeOutTimes = viper.GetInt("consul.watch_time_out")
	config.CONSUL.WatchRate = viper.GetInt("consul.watch_rate")
	config.MQ.Group = viper.GetString("mq.group")
	config.MQ.Namesrv = viper.GetString("mq.namesrv")
	config.MQ.LogPath = viper.GetString("mq.log_path")
	config.MQ.LogLevel = viper.GetString("mq.log_level")
	config.MQ.LogFilesize = viper.GetInt("mq.log_filesize")
	config.MQ.LogFilenum = viper.GetInt("mq.log_filenum")
	config.MQ.PullMaxNum = viper.GetInt("mq.pull_max_num")

	config.AYS_LOG = "/tmp/log/ays/ays.log"
	config.ACCESS_LOG = "/tmp/log/ays/access.log"
	config.ERROR_LOG = "/tmp/log/ays/error.log"

	config.OAUTH.Host = viper.GetString("oauth.host")
	config.AYS.Host   = viper.GetString("ays.host")
}
