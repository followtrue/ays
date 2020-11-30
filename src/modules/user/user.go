package user

import (
	"encoding/json"
	"fmt"
	"github.com/mikemintang/go-curl"
	"ays/src/modules/app"
	"ays/src/modules/logger"
	"strconv"
)

var (
	UserId int
)

type JsonResult struct {
	Code 	int 		`json:"code"`
	Message string 		`json:"message"`
	Data 	DataStruct 	`json:"data"`
}
type DataStruct struct {
	UserId string `json:"userId"`
}

type UserInfoResult struct {
	Code 	int 		`json:"code"`
	Message string 		`json:"message"`
	Data 	InfoStruct 	`json:"data"`
}

type InfoStruct struct {
	UserId 	string `json:"userId"`
	Name 	string `json:"name"`
	Phone 	string `json:"phone"`
	Email 	string `json:"emial"`
}


func GetUserId(token string) (int) {
	//如果有body--需要send body
	req := curl.NewRequest()

	//Http公共参数设置
	req.SetHeaders(map[string]string{
		"Authorization":token,
	})

	url := app.Config.OAUTH.Host
	url += "/v1/user/userid"

	resp, err := req.Send(url, "GET")
	if err != nil {
		logger.Error("auth request failed")
		logger.IfError(err)
		UserId = 0
		return UserId
	}
	if !resp.IsOk() {
		logger.Error("auth request not ok")
		UserId = 0
		return UserId
	}
	var body JsonResult
	err = json.Unmarshal([]byte(resp.Body), &body)
	code := body.Code
	data := body.Data
	UserId, _ = strconv.Atoi(data.UserId)
	if code != 0 || UserId == 0  {
		logger.Error(fmt.Sprintf("auth request code:%v userId:%v", code, UserId))
		return UserId
	}
	return UserId
}

func GetUserInfo(userId int) (InfoStruct) {
	data := InfoStruct{}
	//如果有body--需要send body
	req := curl.NewRequest()

	url := app.Config.OAUTH.Host
	url += "/v1/user/info?id=" + strconv.Itoa(userId)

	resp, err := req.Send(url, "GET")
	if err != nil {
		return data
	}
	if !resp.IsOk() {
		return data
	}
	var body UserInfoResult
	err = json.Unmarshal([]byte(resp.Body), &body)
	data = body.Data
	return data
}