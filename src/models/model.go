package models

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/config"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"strings"
	"time"
)

var TablePrefix = ""
var DB *xorm.Engine
type CommonMap map[string]interface{}

// 创建数据库连接（长连接）
func CreateDb(config *config.Config) *xorm.Engine {
	dsn := getDbEngineDSN(config)
	engine, err := xorm.NewEngine(config.DB.Engine, dsn)
	if err != nil {
		logger.Fatal("创建xorm引擎失败", err)
	}

	engine.SetMaxIdleConns(config.DB.MaxIdleConns)
	engine.SetMaxOpenConns(config.DB.MaxOpenConns)

	if config.DB.Prefix != "" {
		// 设置表前缀
		TablePrefix = config.DB.Prefix
		mapper := core.NewPrefixMapper(core.SnakeMapper{}, config.DB.Prefix)
		engine.SetTableMapper(mapper)
	}

	// 本地环境开启日志
	if config.ENV_OPT.Local == config.ENV {
		engine.ShowSQL(true)
		engine.Logger().SetLevel(core.LOG_DEBUG)
	}

	go keepDbAlived(engine)

	return engine
}

// 拼接数据库连接
func getDbEngineDSN(config *config.Config) string {
	engine := strings.ToLower(config.DB.Engine)
	dsn := ""
	switch engine {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s",
			config.DB.User,
			config.DB.Password,
			config.DB.Host,
			config.DB.Port,
			config.DB.Database,
			config.DB.Charset)
	case "postgres":
		dsn = fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable",
			config.DB.User,
			config.DB.Password,
			config.DB.Host,
			config.DB.Port,
			config.DB.Database)
	}

	return dsn
}

func keepDbAlived(engine *xorm.Engine) {
	t := time.Tick(180 * time.Second)
	for {
		<-t
		engine.Ping()
	}
}