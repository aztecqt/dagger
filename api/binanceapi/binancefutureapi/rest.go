/*
 * @Author: aztec
 * @Date: 2022-10-20 16:02:49
 * @Description: 币安Usdt合约api
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binancefutureapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/util/network"
)

type APIClass int

const (
	API_ClassicUsd APIClass = iota
	API_ClassicUsdt
	API_UnifiedUsd
	API_UnifiedUsdt
)

// 默认全部使用经典U本位合约的url格式
// 币本位合约、统一账户的U、币本位合约做修改
func realUrl(url string, ac APIClass) string {
	// https://fapi.binance.com/fapi/v1/userTrades
	// https://dapi.binance.com/dapi/v1/userTrades
	// https://papi.binance.com/papi/v1/um/userTrades
	// https://papi.binance.com/papi/v1/cm/userTrades
	if ac == API_ClassicUsd {
		url = strings.ReplaceAll(url, "fapi", "dapi")
	} else if ac == API_UnifiedUsd {
		url = strings.ReplaceAll(url, "fapi.binance", "papi.binance")
		url = strings.ReplaceAll(url, "fapi/v1", "papi/v1/cm")
	} else if ac == API_UnifiedUsdt {
		url = strings.ReplaceAll(url, "fapi.binance", "papi.binance")
		url = strings.ReplaceAll(url, "fapi/v1", "papi/v1/um")
	}
	return url
}

// 有些接口，统一账户没有，只能使用经典账户的接口
func realUrlMissingInUnified(url string, ac APIClass) string {
	if ac == API_ClassicUsd || ac == API_UnifiedUsd {
		url = strings.ReplaceAll(url, "fapi", "dapi")
	}
	return url
}

func IsUsdtContract(ac APIClass) bool {
	return ac == API_ClassicUsdt || ac == API_UnifiedUsdt
}

func IsClassicContract(ac APIClass) bool {
	return ac == API_ClassicUsd || ac == API_ClassicUsdt
}

func apiType(ac APIClass) string {
	switch ac {
	case API_ClassicUsd:
		return "classic_usd_contract"
	case API_ClassicUsdt:
		return "classic_usdt_contract"
	case API_UnifiedUsd:
		return "unified_usd_contract"
	case API_UnifiedUsdt:
		return "unified_usdt_contract"
	default:
		return "unknown_api_class"
	}
}

const rootUrl = "https://fapi.binance.com"
const restLogPrefix = "binance_contract_rest"

// 获取服务器时间（毫秒数）
var serverTsDelta int64

func ServerTsCm() int64 {
	return ServerTs(API_ClassicUsd)
}

func ServerTsUm() int64 {
	return ServerTs(API_ClassicUsdt)
}

func GetServerTs(ac APIClass) int64 {
	action := "/fapi/v1/time"
	method := "GET"
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.ServerTime](restLogPrefix, "GetServerTS", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	if err == nil {
		return rst.ServerTime
	} else {
		return 0
	}
}

func GetExchangeInfo_RateLimit(ac APIClass) (*binanceapi.ExchangeInfo_RateLimit, error) {
	action := "/fapi/v1/exchangeInfo"
	method := "GET"
	params := url.Values{}
	params.Set("symbol", "BTCUSDT")
	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.ExchangeInfo_RateLimit](restLogPrefix, "GetExchangeInfo_RateLimit", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	if err == nil && serverTsDelta == 0 {
		serverTsDelta = rst.ServerTime - time.Now().UnixMilli()
	}
	return rst, err
}

// 获取交易对信息
func GetExchangeInfo_Symbols(ac APIClass) (*binanceapi.ExchangeInfo_Symbols, error) {
	action := "/fapi/v1/exchangeInfo"
	method := "GET"
	url := rootUrl + action

	rst, err := network.ParseHttpResult[binanceapi.ExchangeInfo_Symbols](restLogPrefix, "GetExchangeInfo_Symbols", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	if err == nil && serverTsDelta == 0 {
		serverTsDelta = rst.ServerTime - time.Now().UnixMilli()
	}
	return rst, err
}

// 本地推算服务器时间（毫秒数）
func ServerTs(ac APIClass) int64 {
	if serverTsDelta == 0 {
		sts := GetServerTs(ac)
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
func GetLatestPrice(symbol string, ac APIClass) (*[]binanceapi.LatestPrice, error) {
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
	if single && IsUsdtContract(ac) {
		rst, err := network.ParseHttpResult[binanceapi.LatestPrice](restLogPrefix, "GetContractLatestPrice", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
		if err == nil {
			respArry := make([]binanceapi.LatestPrice, 0)
			respArry = append(respArry, *rst)
			return &respArry, nil
		} else {
			return nil, err
		}
	} else {
		rst, err := network.ParseHttpResult[[]binanceapi.LatestPrice](restLogPrefix, "GetContractLatestPrice", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
		return rst, err
	}
}

// 合约买一卖一价格
func GetBookTicker(symbol string, ac APIClass) (*[]binanceapi.BookTicker, error) {
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
		rst, err := network.ParseHttpResult[binanceapi.BookTicker](restLogPrefix, "GetContractBookTicker", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
		if err == nil {
			respArry := make([]binanceapi.BookTicker, 0)
			respArry = append(respArry, *rst)
			return &respArry, nil
		} else {
			return nil, err
		}
	} else {
		rst, err := network.ParseHttpResult[[]binanceapi.BookTicker](restLogPrefix, "GetContractBookTicker", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
		return rst, err
	}
}

// 24小时价格变动（好奇怪的名字）
func Get24hrTicker(ac APIClass, symbols ...string) (*[]binanceapi.Ticker24hr, error) {
	action := "/fapi/v1/ticker/24hr"
	method := "GET"
	paramStr := ""
	single := false
	params := url.Values{}
	if len(symbols) > 0 {
		if len(symbols) == 1 {
			params.Set("symbol", symbols[0])
			single = true
		} else {
			d, _ := json.Marshal(symbols)
			symbolsstr := string(d)
			params.Set("symbols", symbolsstr)
		}
	}
	paramStr = params.Encode()
	action = action + "?" + paramStr
	url := rootUrl + action
	if single {
		rst, err := network.ParseHttpResult[binanceapi.Ticker24hr](restLogPrefix, "GetFuture24hrTicker", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, "spot")
		}, binanceapi.ErrorCallback)
		if err == nil {
			respArry := []binanceapi.Ticker24hr{*rst}
			return &respArry, nil
		} else {
			return nil, err
		}

	} else {
		rst, err := network.ParseHttpResult[[]binanceapi.Ticker24hr](restLogPrefix, "GetFuture24hrTicker", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, "spot")
		}, binanceapi.ErrorCallback)
		return rst, err
	}
}

func GetKline_Usdt(symbol, interval string, t0, t1 time.Time, limit int) (*binanceapi.KLine, error) {
	return GetKline(symbol, interval, t0, t1, limit, API_ClassicUsdt)
}

func GetKline_Usd(symbol, interval string, t0, t1 time.Time, limit int) (*binanceapi.KLine, error) {
	return GetKline(symbol, interval, t0, t1, limit, API_ClassicUsd)
}

// 取K线
// 返回：[[开盘时间，开盘价，最高，最低，收盘价，成交额]]
func GetKline(symbol, interval string, t0, t1 time.Time, limit int, ac APIClass) (*binanceapi.KLine, error) {
	return getKlineFromEndpoint("/fapi/v1/klines", symbol, interval, t0, t1, limit, ac)
}

// 取溢价指数K线
func GetPremiumIndexKline(symbol, interval string, t0, t1 time.Time, limit int, ac APIClass) (*binanceapi.KLine, error) {
	return getKlineFromEndpoint("/fapi/v1/premiumIndexKlines", symbol, interval, t0, t1, limit, ac)
}

func getKlineFromEndpoint(action, symbol, interval string, t0, t1 time.Time, limit int, ac APIClass) (*binanceapi.KLine, error) {
	method := "GET"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("limit", fmt.Sprintf("%d", limit))
	if !t0.IsZero() {
		params.Set("startTime", fmt.Sprintf("%d", t0.UnixMilli()))
	}
	if !t1.IsZero() {
		params.Set("endTime", fmt.Sprintf("%d", t1.UnixMilli()))
	}
	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.KLine](restLogPrefix, "GetKline", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, "spot")
	}, binanceapi.ErrorCallback)

	for i := 0; i < len(*rst); i++ {
		(*rst)[i][0] = int64((*rst)[i][0].(float64))
	}

	return rst, err
}

// 取历史费率信息
func GetHistoryFundingRate(symbol string, t0, t1 time.Time, limit int, ac APIClass) (*[]binanceapi.FundingFee, error) {
	action := "/fapi/v1/fundingRate"
	method := "GET"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(int64(limit), 10))
	if !t0.IsZero() {
		params.Set("startTime", strconv.FormatInt(t0.UnixMilli(), 10))
	}
	if !t1.IsZero() {
		params.Set("endTime", strconv.FormatInt(t1.UnixMilli(), 10))
	}
	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[[]binanceapi.FundingFee](restLogPrefix, "GetHistoryFundingRate", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	return rst, err
}

// 获取最新资金费率/指数价格
func GetPremiumIndex(symbol string, ac APIClass) (*binanceapi.PremiumIndexResp, error) {
	action := "/fapi/v1/premiumIndex"
	method := "GET"
	params := url.Values{}
	params.Set("symbol", symbol)
	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[binanceapi.PremiumIndexResp](restLogPrefix, "GetPremiumIndex", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	if err == nil {
		rst.Parse()
	}
	return rst, err
}

// 获取当前的市场合约持仓量
// pair: BTCUSD
// contractType：ALL, CURRENT_QUARTER, NEXT_QUARTER, PERPETUAL
func GetMarketHold(symbolOrPair string, ac APIClass) (*[]binanceapi.MarketHold, error) {
	action := "/futures/data/openInterestHist"
	method := "GET"
	params := url.Values{}
	if IsUsdtContract(ac) {
		params.Set("symbol", symbolOrPair)
	} else {
		params.Set("pair", symbolOrPair)
		params.Set("contractType", "PERPETUAL") // 仅支持永续
	}
	params.Set("period", "1d")
	params.Set("limit", "1")

	paramsStr := params.Encode()
	action = action + "?" + paramsStr
	url := rootUrl + action
	rst, err := network.ParseHttpResult[[]binanceapi.MarketHold](restLogPrefix, "GetMarketHold", realUrlMissingInUnified(url, ac), method, "", nil, func(resp *http.Response, body []byte) {
		binanceapi.ProcessResponse(resp, body, apiType(ac))
	}, binanceapi.ErrorCallback)
	return rst, err
}

// 获取杠杆分层标准
func GetLeverageBracket(symbolOrPair string, ac APIClass) (*[]binanceapi.LeverageBracket, error) {
	action := "/fapi/v1/leverageBracket"
	method := "GET"
	params := url.Values{}
	if IsUsdtContract(ac) {
		params.Set("symbol", symbolOrPair)
	} else {
		params.Set("pair", symbolOrPair)
	}

	header, paramstr, err := binanceapi.SignerIns.Sign(params)
	url := fmt.Sprintf("%s%s?%s", rootUrl, action, paramstr)

	rst, err := network.ParseHttpResult[[]binanceapi.LeverageBracket](
		restLogPrefix,
		"GetLeverageBracket",
		realUrl(url, ac),
		method,
		"",
		header, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
	return rst, err
}

// 获取成交记录
func GetUserTrade(symbol string, t0, t1 time.Time, limit int, fromId int64, ac APIClass) (*[]binanceapi.FutureUserTrade, error) {
	action := "/fapi/v1/userTrades"
	method := "GET"
	params := url.Values{}
	params.Set("symbol", symbol)
	if !t0.IsZero() {
		params.Set("startTime", strconv.FormatInt(t0.UnixMilli(), 10))
	}

	if !t1.IsZero() {
		params.Set("endTime", strconv.FormatInt(t0.UnixMilli(), 10))
	}

	if fromId > 0 {
		params.Set("fromId", strconv.FormatInt(fromId, 10))
	}

	if limit > 0 {
		params.Set("limit", strconv.FormatInt(int64(limit), 10))
	}

	header, paramstr, err := binanceapi.SignerIns.Sign(params)
	url := fmt.Sprintf("%s%s?%s", rootUrl, action, paramstr)

	rst, err := network.ParseHttpResult[[]binanceapi.FutureUserTrade](
		restLogPrefix,
		"GetUserTrade",
		realUrl(url, ac),
		method,
		"",
		header, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)
	return rst, err
}

// 获取资金流水
func GetAccountIncome(symbol string, incomeType string, t0, t1 time.Time, limit int, page int, ac APIClass) (*[]binanceapi.AccountIncome, error) {
	action := "/fapi/v1/income"
	method := "GET"
	params := url.Values{}

	if len(symbol) > 0 {
		params.Set("symbol", symbol)
	}

	if len(incomeType) > 0 {
		params.Set("incomeType", incomeType)
	}

	if !t0.IsZero() {
		params.Set("startTime", strconv.FormatInt(t0.UnixMilli(), 10))
	}

	if !t1.IsZero() {
		params.Set("endTime", strconv.FormatInt(t0.UnixMilli(), 10))
	}

	if limit > 0 {
		params.Set("limit", strconv.FormatInt(int64(limit), 10))
	}

	if page > 0 {
		params.Set("page", strconv.FormatInt(int64(page), 10))
	}

	header, paramstr, err := binanceapi.SignerIns.Sign(params)
	url := fmt.Sprintf("%s%s?%s", rootUrl, action, paramstr)

	rst, err := network.ParseHttpResult[[]binanceapi.AccountIncome](
		restLogPrefix,
		"GetAccountIncome",
		realUrl(url, ac),
		method,
		"",
		header, func(resp *http.Response, body []byte) {
			binanceapi.ProcessResponse(resp, body, apiType(ac))
		}, binanceapi.ErrorCallback)

	for i := range *rst {
		(*rst)[i].Parse()
	}

	return rst, err
}
