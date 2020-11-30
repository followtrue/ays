package ays_run

import (
	"errors"
	"github.com/mikemintang/go-curl"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/graceful"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
)


const (
	HttpRequestTypeGet 		int = 1		// Get
	HttpRequestTypePost 	int = 2		// Post
	HttpRequestTypePut 		int = 3		// Put
	HttpRequestTypeDelete 	int = 4		// Delete
	HttpRequestTypePatch 	int = 5		// Patch
)

//HTTP请求类型
var HttpRequestType map[int]string = map[int]string{
	HttpRequestTypeGet 		: "GET"		,
	HttpRequestTypePost 	: "POST"	,
	HttpRequestTypePut 		: "PUT"		,
	HttpRequestTypeDelete 	: "DELETE"	,
	HttpRequestTypePatch 	: "PATCH"	,
}

func HttpJob(jobDetail models.Job) (result bool, err error) {
	graceful.AddJob() // 记录执行中的任务，进程退出前等待任务执行
	defer graceful.FinishJob() // 标记任务执行完成

	//job验证
	if checkHttpJob(jobDetail) != true {
		logger.Info("job is invalid", jobDetail)
		return false, errors.New("JOB 不存在")
	}

	//如果有body--需要send body
	req := curl.NewRequest()

	//Http公共参数设置
	if jobDetail.HttpRequestType == HttpRequestTypePost {
		jobDetail.HttpRequestHeader["Content-Type"] = "application/json"
	}

	req.SetHeaders(jobDetail.HttpRequestHeader)

	//POST-Body设置
	if jobDetail.HttpRequestType == HttpRequestTypePost {
		req.SetPostData(jobDetail.HttpRequestBody)
	}

	resp, err := req.Send(jobDetail.HttpRequestUrl, HttpRequestType[jobDetail.HttpRequestType])

	if err != nil {
		logger.Error("Http Job Failed", jobDetail.HttpRequestUrl, err.Error(),resp)
		return false, err
	}

	if !resp.IsOk() {
		//失败重试
		logger.Error("Http Job Send Failed", jobDetail.HttpRequestUrl, resp.Raw, resp.Body)
		return false, errors.New(resp.Body)
	}
	//执行成功
	return true, nil
}

func checkHttpJob(jobDetail models.Job) bool{
	//执行方式是否HTTP
	if jobDetail.RunType != 1 {
		return false
	}
	//HTTP请求方式是否支持

	//HTTP请求地址是否正
	return true
}