package logger

import (
	"fmt"
	"time"
)

//var logger *zap.Logger

type Level int8

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
	PANIC
)

func InitLogger() {
	//logFile := "/tmp/log/ays/ays.log"
	//tools.CreateFile(logFile)
	//
	//cfg := zap.NewProductionConfig()
	//cfg.OutputPaths = []string{
	//	logFile,
	//}
	//
	//var err error
	//logger, err = cfg.Build()
	//if err != nil {
	//	panic(err)
	//}
}

func Debug(msg string, data ...interface{}) {
	write(DEBUG, msg, data)
}

func Info(msg string, data ...interface{}) {
	write(INFO, msg, data)
}

func Warn(msg string, data ...interface{}) {
	write(WARN, msg, data)
}

func Error(msg string, data ...interface{}) {
	write(ERROR, msg, data)
}

func Fatal(msg string, data ...interface{}) {
	write(FATAL, msg, data)
}

func Panic(msg string, data ...interface{}) {
	write(PANIC, msg, data)
}

func IfError(err error) {
	if err != nil {
		Error(err.Error(), err)
	}
}

func IfFatal(err error) {
	if err != nil {
		Fatal(err.Error(), err)
	}
}

func IfPanic(err error) {
	if err != nil {
		Panic(err.Error(), err)
	}
}

func write(level Level, msg string, data ...interface{}) {
	fmt.Println(fmt.Sprintf("[%s](%d) %s", time.Now().Format("2006-01-02 15:04:05"), level, msg))
	fmt.Print(data)

	//switch level {
	//case DEBUG:
	//	logger.Debug(msg, zap.Any("data", data))
	//case INFO:
	//	logger.Info(msg, zap.Any("data", data))
	//case WARN:
	//	logger.Warn(msg, zap.Any("data", data))
	//case ERROR:
	//	logger.Error(msg, zap.Any("data", data))
	//case FATAL:
	//	logger.Fatal(msg, zap.Any("data", data))
	//case PANIC:
	//	logger.Panic(msg, zap.Any("data", data))
	//}
}