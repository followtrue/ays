package ays_run

import (
	"errors"
	"fmt"
	"ays/src/models"
	"ays/src/modules/logger"
	"ays/src/modules/tools"
	"golang.org/x/net/context"
)

func LocalJob(jobDetail models.Job) (result string, err error) {
	//job验证
	if checkLocalJob(jobDetail) != true {
		logger.Info("job is invalid", jobDetail)
		return "", errors.New("JOB 不存在")
	}
	//执行脚本
	ctx := context.Background()
	res, err := tools.ExecShell(ctx, jobDetail.Command)
	fmt.Println("---------local-job-output-----------")
	fmt.Println(res)
	fmt.Println("--------------------")
	if err != nil {
		fmt.Println("local-job---error!!!!!!!!!!!!!!!!!!")
		fmt.Println("----------local-job-err----------")
		fmt.Println(err.Error())
		fmt.Println("--------------------")
		return "", err
	}
	return res, nil
}

func checkLocalJob(jobDetail models.Job) bool{
	return true
}