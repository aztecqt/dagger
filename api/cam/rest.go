/*
- @Author: aztec
- @Date: 2023-11-22 15:58:34
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cam

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/network"
)

const logPrefix = "camApi"
const rootUrl = "https://cam.antalpha.com/api/v1"

type Api struct {
	lang           string
	department     string
	departmentList string
	timezone       string
}

func NewApi() *Api {
	a := &Api{
		lang:           "zh",             // hard code
		department:     "armada/default", // hard code
		departmentList: "armada/default", // hard code
		timezone:       "Asia/Shanghai",  // hard code
	}
	return a
}

func (a *Api) defaultParam() url.Values {
	params := url.Values{}
	params.Set("lang", a.lang)
	params.Set("department", a.department)
	params.Set("timezone", a.timezone)
	return params
}

func (a *Api) defaultParamWithDepartmentList() url.Values {
	params := url.Values{}
	params.Set("lang", a.lang)
	params.Set("department", a.department)
	params.Set("timezone", a.timezone)
	params.Set("department_list", a.departmentList)
	return params
}

func commonHeader() map[string]string {
	headers := map[string]string{}
	headers["Content-Type"] = "application/json"
	return headers
}

func (a *Api) Ping() bool {
	action := "/httpmisc/ping"
	method := "GET"
	params := a.defaultParam()
	url := rootUrl + action + "?" + params.Encode()
	if resp, err := network.ParseHttpResult[RespPong](logPrefix, "Ping", url, method, "", nil, nil, nil); err == nil {
		logger.LogInfo(logPrefix, "ping response: %s", resp.Code)
		return resp.Code == "pong"
	} else {
		logger.LogImportant(logPrefix, err.Error())
		return false
	}
}

// 账号信息。主要用来从AccountAlias查询AccountName
func (a *Api) GetAccountInfo() (*RespAccountInfo, error) {
	action := "/tradeacc/list-simple-accounts-by-department"
	method := "GET"
	params := a.defaultParamWithDepartmentList()
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespAccountInfo](logPrefix, "GetAccountInfo", url, method, "", nil, nil, nil)
}

func (a *Api) GetFundList() (*RespFundList, error) {
	action := "/fund/overview/multi-portfolio/get-config"
	method := "GET"
	params := a.defaultParam()
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespFundList](logPrefix, "GetFundList", url, method, "", nil, nil, nil)
}

func (a *Api) getFundDetailListTaskid() (*RespTaskid, error) {
	action := "/fund/overview/multi-portfolio/get-details"
	method := "GET"
	params := a.defaultParam()
	params.Set("now", strconv.FormatInt(time.Now().UnixNano(), 10))
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespTaskid](logPrefix, "getFundDetailListTaskid", url, method, "", nil, nil, nil)
}

func (a *Api) GetFundDetailList() (*RespFundDetailListInner, error) {
	action := "/fund/overview/multi-portfolio/get-details/fetch-by-task-id"
	method := "GET"
	params := a.defaultParam()

	if resp, err := a.getFundDetailListTaskid(); err == nil {
		params.Set("task_id", resp.TaskId)
		url := rootUrl + action + "?" + params.Encode()
		for i := 0; i < 20; i++ {
			time.Sleep(time.Second * time.Duration(i))
			if resp, err := network.ParseHttpResult[RespFundDetailList](logPrefix, "GetFundDetailList", url, method, "", nil, nil, nil); err == nil {
				if resp.Status == 1 {
					return &resp.Data, nil
				}
			}
		}
		return nil, errors.New("time out")
	} else {
		return nil, errors.New("get task_id failed")
	}
}

func (a *Api) GetFundBasicInfo(fundName string) (*RespBasicInfo, error) {
	action := "/fund/portfolio/detail/get-basic-info"
	method := "GET"
	params := a.defaultParam()
	params.Set("name", fundName)
	params.Set("end_time", fmt.Sprintf("%d", time.Now().UnixNano()))
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespBasicInfo](logPrefix, "GetFundBasicInfo", url, method, "", nil, nil, nil)
}

func (a *Api) GetFundCeFiAssets(fundName string) (*RespAssets, error) {
	action := "/fund/portfolio/detail/get-cefi-asset"
	method := "GET"
	params := a.defaultParam()
	params.Set("fund_name", fundName)
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespAssets](logPrefix, "GetFundCeFiAssets", url, method, "", nil, nil, nil)
}

func (a *Api) GetFundCeFiPositions(fundName string) (*RespPositions, error) {
	action := "/fund/portfolio/detail/get-cefi-position"
	method := "GET"
	params := a.defaultParam()
	params.Set("fund_name", fundName)
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespPositions](logPrefix, "GetFundCeFiPositions", url, method, "", nil, nil, nil)
}

func (a *Api) GetFundRisk(fundName string) (*RespRisk, error) {
	action := "/fund/fund-details/risk"
	method := "GET"
	params := a.defaultParam()
	params.Set("fund_name", fundName)
	url := rootUrl + action + "?" + params.Encode()
	return network.ParseHttpResult[RespRisk](logPrefix, "GetFundRisk", url, method, "", nil, nil, nil)
}

func (a *Api) getOrderRecordTaskId(accName string, t0, t1 time.Time) (*RespTaskid, error) {
	action := "/fund/trading-analysis/list-order-records"
	method := "POST"
	params := a.defaultParam()
	url := rootUrl + action + "?" + params.Encode()

	/*
		end_time: 1702397104251000000
		fund_names: []
		start_time: 1701792304251000000
		tag_names: []
		tradeacc_names: ["tradeacc/bnprop/ltpnew44virtual-be"]
	*/
	payload := make(map[string]interface{})
	payload["start_time"] = t0.UnixNano()
	payload["end_time"] = t1.UnixNano()
	payload["tradeacc_names"] = []string{accName}
	payload["fund_names"] = []string{}
	payload["tag_names"] = []string{}

	return network.ParseHttpResult[RespTaskid](logPrefix, "getOrderRecordTaskId", url, method, util.Object2StringWithoutIntent(payload), commonHeader(), nil, nil)
}

func (a *Api) GetOrderRecord(accName string, t0, t1 time.Time) (*RespOrderRecordInner, error) {
	action := "/fund/trading-analysis/list-order-records/fetch-by-task-id"
	method := "POST"
	params := a.defaultParam()

	if resp, err := a.getOrderRecordTaskId(accName, t0, t1); err == nil {
		payload := make(map[string]interface{})
		payload["task_id"] = resp.TaskId
		url := rootUrl + action + "?" + params.Encode()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second * time.Duration(i))
			if resp, err := network.ParseHttpResult[RespOrderRecord](logPrefix, "GetOrderRecord", url, method, util.Object2String(payload), commonHeader(), nil, nil); err == nil {
				if resp.Status == 1 {
					resp.Data.parse()
					return &resp.Data, nil
				}
			}
		}
		return nil, errors.New("time out")
	} else {
		return nil, errors.New("get task_id failed")
	}
}

func (a *Api) getDealRecordTaskId(accName string, t0, t1 time.Time) (*RespTaskid, error) {
	action := "/fund/trading-analysis/list-transaction-records"
	method := "POST"
	params := a.defaultParam()
	url := rootUrl + action + "?" + params.Encode()

	/*
		account_names: ["tradeacc/okuniprop/ltpamokx010-35"]
		end_time: 1702223999000000000
		fund_names: []
		start_time: 1702137600000000000
		tag_names: []
	*/
	payload := make(map[string]interface{})
	payload["start_time"] = t0.UnixNano()
	payload["end_time"] = t1.UnixNano()
	payload["account_names"] = []string{accName}
	payload["fund_names"] = []string{}
	payload["tag_names"] = []string{}

	return network.ParseHttpResult[RespTaskid](logPrefix, "getDealRecordTaskId", url, method, util.Object2StringWithoutIntent(payload), commonHeader(), nil, nil)
}

func (a *Api) GetDealRecord(accName string, t0, t1 time.Time) (*RespDealRecordInner, error) {
	action := "/fund/trading-analysis/list-transaction-records/fetch-by-task-id"
	method := "POST"
	params := a.defaultParam()

	if resp, err := a.getDealRecordTaskId(accName, t0, t1); err == nil {
		payload := make(map[string]interface{})
		payload["task_id"] = resp.TaskId
		url := rootUrl + action + "?" + params.Encode()
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second * time.Duration(i))
			if resp, err := network.ParseHttpResult[RespDealRecord](logPrefix, "GetDealRecord", url, method, util.Object2String(payload), commonHeader(), nil, nil); err == nil {
				if resp.Status == 1 {
					resp.Data.parse()
					return &resp.Data, nil
				}
			}
		}
		return nil, errors.New("time out")
	} else {
		return nil, errors.New("get task_id failed")
	}
}
