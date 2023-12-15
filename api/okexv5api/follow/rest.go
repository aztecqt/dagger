/*
 * @Author: aztec
 * @Date: 2023-09-29 10:02:45
 * @Description: ok的跟单api，是非公开api
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package follow

import (
	"net/url"
	"strconv"
	"time"

	"github.com/aztecqt/dagger/util/network"
)

const rootUrl = "https://www.okx.com"
const logPrefix = "okexv5_follow"

// 分页获取交易员列表
// pagesize最大20
func GetTraders(pagesize, page int) (*TraderListResp, error) {
	// action := "/priapi/v5/ecotrade/public/popular-star-rate" // 这个返回结果少
	action := "/priapi/v5/ecotrade/public/trade-expert-rate" // 这个返回结果多
	method := "GET"
	params := url.Values{}
	params.Set("size", strconv.FormatInt(int64(pagesize), 10))
	params.Set("num", strconv.FormatInt(int64(page+1), 10))
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[TraderListResp](logPrefix, "GetTrader", url, method, "", nil, nil, nil)
	return resp, err
}

// 获取当前持仓详情
func GetPositionDetail(traderId string) (*PositionDetailResp, error) {
	action := "/priapi/v5/ecotrade/public/position-detail"
	method := "GET"
	params := url.Values{}
	params.Set("uniqueName", traderId)
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionDetailResp](logPrefix, "GetPositionDetail", url, method, "", nil, nil, nil)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 获取持仓历史
// size貌似无上限，保险起见填100
func GetPositionHistory(traderId string, size int64, afterId int64) (*PositionHistoryResp, error) {
	action := "/priapi/v5/ecotrade/public/position-history"
	method := "GET"
	params := url.Values{}
	params.Set("uniqueName", traderId)
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("size", strconv.FormatInt(size, 10))
	if afterId > 0 {
		params.Set("after", strconv.FormatInt(afterId, 10))
	}
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionHistoryResp](logPrefix, "GetPositionHistory", url, method, "", nil, nil, nil)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 获取当前持仓详情V2
func GetPositionDetailV2(traderId string) (*PositionDetailRespV2, error) {
	action := "/priapi/v5/ecotrade/public/positions-v2"
	method := "GET"
	params := url.Values{}
	params.Set("uniqueName", traderId)
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionDetailRespV2](logPrefix, "GetPositionDetail", url, method, "", nil, nil, nil)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 获取持仓历史
func GetPositionHistoryV2(traderId string, limit int64, afterId int64) (*PositionHistoryRespV2, error) {
	action := "/priapi/v5/ecotrade/public/history-positions"
	method := "GET"
	params := url.Values{}
	params.Set("uniqueName", traderId)
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("limit", strconv.FormatInt(limit, 10))
	if afterId > 0 {
		params.Set("after", strconv.FormatInt(afterId, 10))
	}
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionHistoryRespV2](logPrefix, "GetPositionHistory", url, method, "", nil, nil, nil)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 获取操作记录
// before：获取比这个orderId更晚的数据
// after：获取比这个orderId更早的数据
func GetTradeRecord(traderId string, limit int64, afterId, beforeId int64) (*TradeRecordResp, error) {
	//https: //www.okx.com/priapi/v5/ecotrade/public/trade-records?t=1698138196058&limit=20&startModify=1690128000000&endModify=1698163199000&uniqueName=563E3A78CDBAFB4E
	action := "/priapi/v5/ecotrade/public/trade-records"
	method := "GET"
	params := url.Values{}
	params.Set("uniqueName", traderId)
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("limit", strconv.FormatInt(limit, 10))
	if afterId > 0 {
		params.Set("after", strconv.FormatInt(afterId, 10))
	}
	if beforeId > 0 {
		params.Set("before", strconv.FormatInt(beforeId, 10))
	}
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[TradeRecordResp](logPrefix, "GetTradeRecord", url, method, "", nil, nil, nil)
	if err == nil {
		resp.parse()
	}
	return resp, err
}
