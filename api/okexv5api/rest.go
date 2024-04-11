/*
 * @Author: aztec
 * @Date: 2022-03-25 17:20:22
 * @Description: 对okexv5所有rest调用的封装
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package okexv5api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/network"
	"github.com/shopspring/decimal"
)

const rootUrl = "https://www.okx.com"
const restLogPrefix = "okexv5_rest"

// 外部通过设置这个回调来处理关键错误
var ErrorCallback func(e error)

// 通用错误处理
func CheckRestResp(resp CommonRestResp, err error, op, logPrefix string) bool {
	if err != nil {
		logger.LogImportant(logPrefix, "%s failed: %s", op, err.Error())
		return false
	} else if resp.Code != "0" {
		logger.LogImportant(logPrefix, "%s failed, code=%s, msg=%s", op, resp.Code, resp.Msg)
		return false
	} else {
		return true
	}
}

// 获取服务器时间(毫秒数)
func GetServerTS() int64 {
	action := "/api/v5/public/time"
	method := "GET"
	url := rootUrl + action
	resp, err := network.ParseHttpResult[serverTimeRestResp](restLogPrefix, "GetInstruments", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		ts, _ := strconv.ParseInt(resp.Data[0].TS, 10, 64)
		return ts
	} else {
		return 0
	}
}

// 获取币种列表
func GetCurrencies() (*GetCurrencyResp, error) {
	action := "/api/v5/asset/currencies"
	method := "GET"
	url := rootUrl + action
	resp, err := network.ParseHttpResult[GetCurrencyResp](restLogPrefix, "GetCurrencies", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 获取币种列表（外部）
func GetProjects() (*GetProjectsResp, error) {
	action := "/v2/support/info/announce/listProject"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[GetProjectsResp](restLogPrefix, "GetProjects", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		resp.Parse()
	}
	return resp, err
}

// 获取所有可交易产品的信息列表
// instType:SPOT/MARGIN/SWAP/FUTURES/OPTION
func GetInstruments(instType string) (*InstrumentRestResp, error) {
	action := "/api/v5/public/instruments"
	method := "GET"
	params := url.Values{}
	params.Set("instType", instType)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[InstrumentRestResp](restLogPrefix, "GetInstruments", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 获取单个产品信息
func GetInstrument(instType, instId string) (*InstrumentRestResp, error) {
	action := "/api/v5/public/instruments"
	method := "GET"
	params := url.Values{}
	params.Set("instType", instType)
	params.Set("instId", instId)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[InstrumentRestResp](restLogPrefix, "GetInstrument", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 查行情
func GetTicker(instId string) (*TickerRestResp, error) {
	action := "/api/v5/market/ticker"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[TickerRestResp](restLogPrefix, "GetTicker", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 批量查行情(instType:SPOT/SWAP/FUTURES/OPTION)
func GetTickers(instType string) (*TickerRestResp, error) {
	action := "/api/v5/market/tickers"
	method := "GET"
	params := url.Values{}
	params.Set("instType", instType)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[TickerRestResp](restLogPrefix, "GetTicker", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查深度
func GetDepth(instId string, sz int) (*DepthRestResp, error) {
	action := "/api/v5/market/books"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("sz", fmt.Sprintf("%d", sz))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[DepthRestResp](restLogPrefix, "GetDepth", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 查k线
// bar:1m/3m/5m/15m/30m/1H/2H/4H
func GetKlineBefore(instId string, t time.Time, bar string, limit int) (*KLineRestResp, error) {
	return GetKline(instId, time.Time{}, t, bar, limit)
}

func GetKline(instId string, t0, t1 time.Time, bar string, limit int) (*KLineRestResp, error) {
	action := "/api/v5/market/history-candles"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	if !t0.IsZero() {
		params.Set("before", fmt.Sprintf("%d", t0.UnixMilli()))
	}
	if !t1.IsZero() {
		params.Set("after", fmt.Sprintf("%d", t1.UnixMilli()))
	}

	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	params.Set("bar", bar)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[KLineRestResp](restLogPrefix, "GetKline", url, method, "", nil, nil, ErrorCallback)
	resp.Build()
	return resp, err
}

// 查标记价格
func GetMarkPrice(instId string) (*MarkPriceRestResp, error) {
	action := "/api/v5/public/mark-price"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[MarkPriceRestResp](restLogPrefix, "GetMarkPrice", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 查限价
func GetPriceLimit(instId string) (*PriceLimitRestResp, error) {
	action := "/api/v5/public/price-limit"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PriceLimitRestResp](restLogPrefix, "GetPriceLimit", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 查当前费率
func GetFundingRate(instId string) (*FundingRateRestResp, error) {
	action := "/api/v5/public/funding-rate"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[FundingRateRestResp](restLogPrefix, "GetFundingRate", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查历史费率
func GetFundingRateHistory(instId string, limit int, before, after time.Time) (*FundingRateHistoryRestResp, error) {
	action := "/api/v5/public/funding-rate-history"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("limit", fmt.Sprintf("%d", limit))
	if !before.IsZero() {
		params.Set("before", fmt.Sprintf("%d", before.UnixMilli()))
	}
	if !after.IsZero() {
		params.Set("after", fmt.Sprintf("%d", after.UnixMilli()))
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[FundingRateHistoryRestResp](restLogPrefix, "GetFundingRateHistory", url, method, "", nil, nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查询账户配置
func GetAccountConfig() (*AccountConfigRestResp, error) {
	action := "/api/v5/account/config"
	method := "GET"

	url := rootUrl + action
	resp, err := network.ParseHttpResult[AccountConfigRestResp](restLogPrefix, "GetAccountConfig", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 设置杠杆倍率（目前只能按照instId设置，且只能是"cross"模式）
func SetLeverage(instId string, lever int) (*GetSetLeverageRestResp, error) {
	action := "/api/v5/account/set-leverage"
	method := "POST"
	url := rootUrl + action

	req := make(map[string]string)
	req["instId"] = instId
	req["mgnMode"] = "cross"
	req["lever"] = strconv.Itoa(lever)

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[GetSetLeverageRestResp](restLogPrefix, "SetLeverRate", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 获取杠杆倍率
func GetLeverage(instId string) (*GetSetLeverageRestResp, error) {
	action := "/api/v5/account/leverage-info"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("mgnMode", "cross")
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[GetSetLeverageRestResp](restLogPrefix, "GetLeverage", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查询交易账户余额
func GetAccountBalance(currency []string) (*AccountBalanceRestResp, error) {
	action := "/api/v5/account/balance"
	method := "GET"
	if len(currency) > 0 {
		params := url.Values{}
		params.Set("ccy", strings.Join(currency, ","))
		action = action + "?" + params.Encode()
	}
	url := rootUrl + action
	resp, err := network.ParseHttpResult[AccountBalanceRestResp](restLogPrefix, "GetAccountBalance", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 查询资金账户余额
func GetAssetBalance(currency []string) (*AssetBalanceRestResp, error) {
	action := "/api/v5/asset/balances"
	method := "GET"
	if len(currency) > 0 {
		params := url.Values{}
		params.Set("ccy", strings.Join(currency, ","))
		action = action + "?" + params.Encode()
	}
	url := rootUrl + action
	resp, err := network.ParseHttpResult[AssetBalanceRestResp](restLogPrefix, "GetAssetBalance", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 获取最大可买卖/开仓数量
func GetMaxTradeOrOpenSize(instId, tdMode string) (*MaxSizeRestResp, error) {
	action := "/api/v5/account/max-size"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("tdMode", tdMode)
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[MaxSizeRestResp](restLogPrefix, "GetMaxTradeOrOpenSize", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 获取最大可用数量
// 现货杠杆返回的是借币可用来买币的U，和可用来卖的币的数量
// 合约返回的是
func GetMaxAvailableSize(instId, tdMode string, reduceOnly bool) (*MaxAvailableSizeRestResp, error) {
	action := "/api/v5/account/max-avail-size"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("tdMode", tdMode)
	params.Set("reduceOnly", fmt.Sprintf("%v", reduceOnly))
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[MaxAvailableSizeRestResp](restLogPrefix, "GetMaxAvailableSize", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 查询仓位
// instType：MARGIN/SWAP/FUTURES/OPTION，可以不传
func GetPositions(instType, instId string) (*PositionRestResp, error) {
	action := "/api/v5/account/positions"
	method := "GET"
	params := url.Values{}
	if len(instType) > 0 {
		params.Set("instType", instType)
	}
	if len(instId) > 0 {
		params.Set("instId", instId)

	}
	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionRestResp](restLogPrefix, "GetPositions", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 资金划转
func Transfer(ccy string, amount decimal.Decimal, toAsset bool) (*TransferRestResp, error) {
	action := "/api/v5/asset/transfer"
	method := "POST"
	url := rootUrl + action

	from := "6"
	to := "18"
	if toAsset {
		from = "18"
		to = "6"
	}

	req := TransferReq{
		Ccy:      ccy,
		Amount:   amount.String(),
		From:     from,
		To:       to,
		Type:     "0", // 目前仅支持账户内划转
		ClientId: "",
	}

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[TransferRestResp](restLogPrefix, "Transfer", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 提币
// 提币之前，需要先把目标地址加入白名单且免验证才可以
func Withdraw(
	ccy string,
	amount decimal.Decimal,
	isInnerWithdraw bool,
	toAddr, areaCode string,
	fee decimal.Decimal,
	chain string,
	clientId string) (*WithdrawResp, error) {
	action := "/api/v5/asset/withdrawal"
	method := "POST"
	url := rootUrl + action

	req := WithdrawReq{
		Ccy:      ccy,
		Amount:   amount.String(),
		ToAddr:   toAddr,
		Fee:      fee.String(),
		Chain:    chain,
		AreaCode: areaCode,
		ClientId: clientId,
	}

	if isInnerWithdraw {
		req.Dest = "3"
	} else {
		req.Dest = "4"
	}

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[WithdrawResp](restLogPrefix, "Withdraw", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 查询提币结果
func GetWithdrawHistory(clientId string) (*WithdrawHistoryResp, error) {
	action := "/api/v5/asset/withdrawal-history"
	method := "GET"

	if len(clientId) > 0 {
		params := url.Values{}
		params.Set("clientId", clientId)
		action = action + "?" + params.Encode()
	}

	url := rootUrl + action

	resp, err := network.ParseHttpResult[WithdrawHistoryResp](restLogPrefix, "GetWithdrawHistory", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 下单
func MakeOrder(instID, clientOrderId, tag, side, posSide, orderType, tradeMode string, reduceOnly bool, price, size decimal.Decimal) (*MakeorderRestResp, error) {
	action := "/api/v5/trade/order"
	method := "POST"
	url := rootUrl + action

	req := MakeorderRestReq{
		InstId:        instID,
		TradeMode:     tradeMode,
		ClientOrderId: clientOrderId,
		Tag:           tag,
		Side:          side,
		PosSide:       posSide,
		OrderType:     orderType,
		ReduceOnly:    reduceOnly,
		Price:         price.String(),
		Size:          size.String(),
	}

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[MakeorderRestResp](restLogPrefix, "MakeOrder", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 撤单
func CancelOrder(instID, clientOrderId string, orderId int64) (*CancelOrderRestResp, error) {
	action := "/api/v5/trade/cancel-order"
	method := "POST"
	url := rootUrl + action

	req := make(map[string]string)
	req["instId"] = instID
	if orderId > 0 {
		req["ordId"] = strconv.FormatInt(orderId, 10)
	}
	if len(clientOrderId) > 0 {
		req["clOrdId"] = clientOrderId
	}

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[CancelOrderRestResp](restLogPrefix, "CancelOrder", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 批量撤销订单
func CancelOrderBatch(orders []CancelBatchOrderRestReq) (*CancelOrderRestResp, error) {
	if len(orders) > 20 {
		orders = orders[:20]
	}

	action := "/api/v5/trade/cancel-batch-orders"
	method := "POST"
	url := rootUrl + action
	b, _ := json.Marshal(orders)
	postStr := string(b)
	resp, err := network.ParseHttpResult[CancelOrderRestResp](restLogPrefix, "CancelOrderBatch", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 修改订单
func AmendOrder(instID, clientOrderId, reqId string, orderId int64, newPrice, newSize decimal.Decimal) (*AmendOrderRestResp, error) {
	action := "/api/v5/trade/amend-order"
	method := "POST"
	url := rootUrl + action

	req := make(map[string]interface{})
	req["instId"] = instID
	req["cxlOnFail"] = true

	if newPrice.IsPositive() {
		req["newPx"] = newPrice.String()
	}

	if newSize.IsPositive() {
		req["newSz"] = newSize.String()
	}

	if orderId > 0 {
		req["ordId"] = strconv.FormatInt(orderId, 10)
	}
	if len(clientOrderId) > 0 {
		req["clOrdId"] = clientOrderId
	}
	if len(reqId) > 0 {
		req["reqId"] = reqId
	}

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[AmendOrderRestResp](restLogPrefix, "AmendOrder", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 查询订单
func GetOrderInfo(instId string, orderId int64, clientOrderId string) (*OrderRestResp, error) {
	action := "/api/v5/trade/order"
	method := "GET"

	params := url.Values{}
	params.Set("instId", instId)
	if orderId > 0 {
		params.Add("ordId", strconv.FormatInt(orderId, 10))
	}
	if len(clientOrderId) > 0 {
		params.Add("clOrdId", clientOrderId)
	}
	action = action + "?" + params.Encode()
	url := rootUrl + action

	resp, err := network.ParseHttpResult[OrderRestResp](restLogPrefix, "GetOrderInfo", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	resp.LocalTime = time.Now()
	return resp, err
}

// 获取未成交的订单
func GetPendingOrders(instId string) (*OrderRestResp, error) {
	action := "/api/v5/trade/orders-pending"
	method := "GET"

	if len(instId) > 0 {
		params := url.Values{}
		params.Set("instId", instId)
		action = action + "?" + params.Encode()
	}

	url := rootUrl + action
	resp, err := network.ParseHttpResult[OrderRestResp](restLogPrefix, "GetPendingOrders", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 查询成交明细（近3日，2秒60次）
func GetFills(instId string, t0, t1 time.Time) (*FillsResp, error) {
	action := "/api/v5/trade/fills"
	method := "GET"

	params := url.Values{}
	params.Set("instId", instId)
	if !t0.IsZero() {
		params.Set("begin", fmt.Sprintf("%d", t0.UnixMilli()))
	}
	if !t1.IsZero() {
		params.Set("end", fmt.Sprintf("%d", t1.UnixMilli()))
	}

	action = action + "?" + params.Encode()

	url := rootUrl + action
	resp, err := network.ParseHttpResult[FillsResp](restLogPrefix, "GetFills", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查询成交明细（近3月，2秒10次）
func GetFillsHistory(instId string, t0, t1 time.Time) (*FillsResp, error) {
	action := "/api/v5/trade/fills-history"
	method := "GET"

	params := url.Values{}
	params.Set("instId", instId)
	if strings.Contains(instId, "SWAP") {
		params.Set("instType", "SWAP")
	} else if strings.Count(instId, "-") == 1 {
		params.Set("instType", "SPOT")
	} else {
		logger.LogPanic(restLogPrefix, "GetFillsHistory:unknown instType")
	}

	if !t0.IsZero() {
		params.Set("begin", fmt.Sprintf("%d", t0.UnixMilli()))
	}
	if !t1.IsZero() {
		params.Set("end", fmt.Sprintf("%d", t1.UnixMilli()))
	}

	action = action + "?" + params.Encode()

	url := rootUrl + action
	resp, err := network.ParseHttpResult[FillsResp](restLogPrefix, "GetFills", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}

	return resp, err
}

// 查询成交明细（智能选择）
func GetFills_Auto(instId string, t0, t1 time.Time) (*FillsResp, error, int64) {
	limit := int64(86400 * 2)
	if time.Now().Unix()-t1.Unix() < limit {
		resp, err := GetFills(instId, t0, t1)
		return resp, err, 40
	} else {
		resp, err := GetFillsHistory(instId, t0, t1)
		return resp, err, 200
	}
}

// 查询仓位历史
// instType: MARGIN/SWAP/FUTURE/OPTION
// instId: 跟instType二选一
func GetPositionHistory(instType string, instId string, closeType PositionCloseType, after time.Time) (*PositionHistoryResp, error) {
	action := "/api/v5/account/positions-history"
	method := "GET"

	params := url.Values{}

	if len(instId) > 0 {
		params.Set("instId", instId)
	}

	if len(instType) > 0 {
		params.Set("instType", instType)
	}

	if !after.IsZero() {
		params.Set("before", strconv.FormatInt(after.UnixMilli(), 10))
	}

	if closeType != PositionCloseType_None {
		params.Set("type", strconv.FormatInt(int64(closeType), 10))
	}

	action = action + "?" + params.Encode()

	url := rootUrl + action
	resp, err := network.ParseHttpResult[PositionHistoryResp](restLogPrefix, "GetPositionHistory", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查询历史账单
func GetBills(fromBillId string, fromTime time.Time, limit int) (*BillRestResp, error) {
	action := "/api/v5/account/bills-archive"
	method := "GET"

	params := url.Values{}

	if len(fromBillId) > 0 {
		params.Set("before", fromBillId)
	}

	if !fromTime.IsZero() {
		params.Set("begin", strconv.FormatInt(fromTime.UnixMilli(), 10))
	}

	if limit > 0 {
		params.Set("limit", strconv.FormatInt(int64(limit), 10))
	}

	if len(params) > 0 {
		action = action + "?" + params.Encode()
	}

	url := rootUrl + action
	resp, err := network.ParseHttpResult[BillRestResp](restLogPrefix, "GetBillsHistory", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	if err == nil {
		resp.parse()
	}
	return resp, err
}

// 查询市场公共成交数据
// typ: 1: by tradeId 2:by ts
func GetMarketHistoryTrades(instId string, typ int, after, before int64) (*GetMarketTradesResp, error) {
	action := "/api/v5/market/history-trades"
	method := "GET"
	params := url.Values{}
	params.Set("instId", instId)
	params.Set("type", fmt.Sprintf("%d", typ))

	if after > 0 {
		params.Set("after", fmt.Sprintf("%d", after))
	}

	if before > 0 {
		params.Set("before", fmt.Sprintf("%d", before))
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[GetMarketTradesResp](restLogPrefix, "GetMarketHistoryTrades", url, method, "", nil, nil, ErrorCallback)
	resp.Parse()
	return resp, err
}

// 查询某品种的爆仓订单
func GetLiquidationOrders(instId string, filled bool, limit int, page int) (*GetLiquidationOrdersExtRest, error) {
	action := "/priapi/v5/public/liquidation-orders"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))

	ss := strings.Split(instId, "-")
	if len(ss) == 3 {
		params.Set("instFamily", fmt.Sprintf("%s-%s", ss[0], ss[1]))
		params.Set("instType", ss[2])
	} else {
		return nil, errors.New("invalid instId")
	}

	if filled {
		params.Set("state", "filled")
	} else {
		params.Set("state", "unfilled")
	}

	if page > 1 {
		params.Set("page", strconv.FormatInt(int64(page), 10))
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[GetLiquidationOrdersExtRest](restLogPrefix, "GetLiquidationOrders", url, method, "", nil, nil, ErrorCallback)
	resp.parse()
	return resp, err
}

// 查看defi质押项目
func GetFinanceDefiStakingOffers(ccy string) (*FinanceDefiStakingOffersResp, error) {
	action := "/api/v5/finance/staking-defi/offers"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if len(ccy) > 0 {
		params.Set("ccy", ccy)
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[FinanceDefiStakingOffersResp](restLogPrefix, "GetFinanceStakingOffers", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 查询余币宝余额
func GetFinanceSavingBalance(ccy string) (*FinanceSavingBalanceResp, error) {
	action := "/api/v5/finance/savings/balance"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if len(ccy) > 0 {
		params.Set("ccy", ccy)
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[FinanceSavingBalanceResp](restLogPrefix, "GetFinanceSavingBalance", url, method, "", signerIns.getHttpHeaderWithSign(method, action, ""), nil, ErrorCallback)
	return resp, err
}

// 余币宝申购/赎回
func FinanceSavingPurchaseRedempt(ccy string, amt decimal.Decimal, isPurchase bool) (*FinanceSavingPurchageRedemptResultResp, error) {
	action := "/api/v5/finance/savings/purchase-redempt"
	method := "POST"
	url := rootUrl + action

	req := make(map[string]string)
	req["ccy"] = ccy
	req["amt"] = amt.String()
	req["side"] = util.ValueIf(isPurchase, "purchase", "redempt")
	req["rate"] = "0.01" // hard code，固定填最小值，以保证借出

	b, _ := json.Marshal(req)
	postStr := string(b)
	resp, err := network.ParseHttpResult[FinanceSavingPurchageRedemptResultResp](restLogPrefix, "SetLeverRate", url, method, postStr, signerIns.getHttpHeaderWithSign(method, action, postStr), nil, ErrorCallback)
	return resp, err
}

// 查询市场借贷利率
func GetMarketLendingRateSummary(ccy string) (*MarketLendingRateSummaryResp, error) {
	action := "/api/v5/finance/savings/lending-rate-summary"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))
	if len(ccy) > 0 {
		params.Set("ccy", ccy)
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[MarketLendingRateSummaryResp](restLogPrefix, "GetMarketLendingRateSummary", url, method, "", nil, nil, ErrorCallback)
	return resp, err
}

// 查询市场借贷利率历史
func GetMarketLendingRateHistory(ccy string, after, before time.Time, limit int) (*MarketLendingRateHistoryResp, error) {
	action := "/api/v5/finance/savings/lending-rate-history"
	method := "GET"
	params := url.Values{}
	params.Set("t", strconv.FormatInt(time.Now().UnixMilli(), 10))

	if len(ccy) > 0 {
		params.Set("ccy", ccy)
	}

	if !after.IsZero() {
		params.Set("after", strconv.FormatInt(after.UnixMilli(), 10))
	}

	if !before.IsZero() {
		params.Set("before", strconv.FormatInt(before.UnixMilli(), 10))
	}

	if limit > 0 {
		params.Set("limit", strconv.FormatInt(int64(limit), 10))
	}

	action = action + "?" + params.Encode()
	url := rootUrl + action
	resp, err := network.ParseHttpResult[MarketLendingRateHistoryResp](restLogPrefix, "GetMarketLendingRateHistory", url, method, "", nil, nil, ErrorCallback)
	if resp != nil {
		resp.parse()
	}
	return resp, err
}
