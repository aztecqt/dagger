/*
 * @Author: aztec
 * @Date: 2022-10-20 16:02:49
 * @Description: 币安Usdt合约api
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binancefutureapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/util/network"
)

func realUrl(url string, isusdt bool) string {
	if isusdt {
		return strings.Replace(url, "dapi", "fapi", -1)
	} else {
		return strings.Replace(url, "fapi", "dapi", -1)
	}
}

func apiType(isusdt bool) string {
	if isusdt {
		return "future_usdt"
	} else {
		return "future"
	}
}

const rootUrl = "https://fapi.binance.com"
const restLogPrefix = "binance_contract_rest"

// 获取服务器时间（毫秒数）
var serverTsDelta int64

func GetServerTs(isusdt bool) int64 {
	action := "/fapi/v1/time"
	method := "GET"
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.ServerTime](restLogPrefix, "GetServerTS", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(isusdt))
	}, binanceapi.ErrorCallback)
	if err == nil {
		return rst.ServerTime
	} else {
		return 0
	}
}

func GetExchangeInfo_RateLimit(isusdt bool) (*binanceapi.ExchangeInfo_RateLimit, error) {
	action := "/fapi/v1/exchangeInfo"
	method := "GET"
	params := url.Values{}
	params.Set("symbol", "BTCUSDT")
	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.ExchangeInfo_RateLimit](restLogPrefix, "GetExchangeInfo_RateLimit", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(isusdt))
	}, binanceapi.ErrorCallback)
	if err == nil && serverTsDelta == 0 {
		serverTsDelta = rst.ServerTime - time.Now().UnixMilli()
	}
	return rst, err
}

// 本地推算服务器时间（毫秒数）
func ServerTs(isusdt bool) int64 {
	if serverTsDelta == 0 {
		sts := GetServerTs(isusdt)
		if sts != 0 {
			serverTsDelta = sts - time.Now().UnixMilli()
		}
	}

	if serverTsDelta == 0 {
		return 0
	} else {
		return time.Now().UnixMilli() + serverTsDelta
	}
}

// 合约最新价格
func GetLatestPrice(symbol string, isusdt bool) (*[]binanceapi.LatestPrice, error) {
	action := "/fapi/v1/ticker/price"
	method := "GET"
	paramStr := ""
	single := false
	if len(symbol) > 0 {
		params := url.Values{}
		params.Set("symbol", symbol)
		single = true
		paramStr = params.Encode()
	}

	if len(paramStr) > 0 {
		action = action + "?" + paramStr
	}

	url := rootUrl + action
	if single {
		rst, err := network.ParseHttpResult[binanceapi.LatestPrice](restLogPrefix, "GetContractLatestPrice", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(isusdt))
		}, binanceapi.ErrorCallback)
		if err == nil {
			respArry := make([]binanceapi.LatestPrice, 0)
			respArry = append(respArry, *rst)
			return &respArry, nil
		} else {
			return nil, err
		}
	} else {
		rst, err := network.ParseHttpResult[[]binanceapi.LatestPrice](restLogPrefix, "GetContractLatestPrice", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(isusdt))
		}, binanceapi.ErrorCallback)
		return rst, err
	}
}

// 合约买一卖一价格
func GetBookTicker(symbol string, isusdt bool) (*[]binanceapi.BookTicker, error) {
	action := "/fapi/v1/ticker/bookTicker"
	method := "GET"
	paramStr := ""
	single := false
	if len(symbol) > 0 {
		params := url.Values{}
		params.Set("symbol", symbol)
		single = true
		paramStr = params.Encode()
	}

	if len(paramStr) > 0 {
		action = action + "?" + paramStr
	}

	url := rootUrl + action
	if single {
		rst, err := network.ParseHttpResult[binanceapi.BookTicker](restLogPrefix, "GetContractBookTicker", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(isusdt))
		}, binanceapi.ErrorCallback)
		if err == nil {
			respArry := make([]binanceapi.BookTicker, 0)
			respArry = append(respArry, *rst)
			return &respArry, nil
		} else {
			return nil, err
		}
	} else {
		rst, err := network.ParseHttpResult[[]binanceapi.BookTicker](restLogPrefix, "GetContractBookTicker", realUrl(url, isusdt), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(isusdt))
		}, binanceapi.ErrorCallback)
		return rst, err
	}
}
