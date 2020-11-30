package ays_run

import (
	"errors"
	"fmt"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/rpc/client"
	jobPb "gitlab.keda-digital.com/kedadigital/ays/src/modules/rpc/proto"
)

func CommandJob(jobDetail models.Job, ip string, port int) (result string, err error) {
	//job验证
	if checkCommandJob(jobDetail) != true {
		logger.Error("job is invalid", jobDetail)
		return "", errors.New("JOB 不存在")
	}
	//执行脚本
	jobpb := &jobPb.JobRequest{
		Command: jobDetail.Command,
		Type:1,	//rpc调用默认为1
		Timeout: jobDetail.TimeOut,
		Id: int64(jobDetail.Id)}
	str, err := client.Exec(ip , port, jobpb)
	//fmt.Println("---------command-job-output-----------")
	//fmt.Println(str)
	//fmt.Println("--------------------")
	if err != nil {
		fmt.Println("command-job---error!!!!!!!!!!!!!!!!!!")
		fmt.Println("---------command-job-err-----------")
		fmt.Println(err.Error())
		fmt.Println("--------------------")
		return str, err
	}
	return str, nil
}

func checkCommandJob(jobDetail models.Job) bool{
	return true
}