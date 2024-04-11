/*
 * @Author: aztec
 * @Date: 2022-03-26 21:29:35
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2024-02-13 11:47:02
 * @FilePath: \market_collectorc:\work\svn\quant\go\src\dagger\api\okexv5api\websocket.go
 * @Description: okexv5的ws
 * 关于消息处理。okexv5的行情数据，大都在头部的arg节点下包含instID，这种消息可以按照instID进行分发
 * 因此外部可以通过instID进行订阅
 * 而另一些ws数据没有这个字段（比如订单、仓位、权益等私有数据），因此处理方法有所不同
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

const publicURL = "wss://ws.okx.com:8443/ws/v5/public"
const privateURL = "wss://ws.okx.com:8443/ws/v5/private"
const wsLogPrefix = "okexv5_ws"
const wsLogPrefixPublic = "okexv5_public_ws"
const wsLogPrefixPrivate = "okexv5_private_ws"

type WsClient struct {
	publicWsConn  api.WsConnection
	privateWsConn api.WsConnection

	// 内部数据解析（unmarshal）
	rawRespFns map[string]api.OnRecvWSRawMsg

	// 外部回调(instID-fn)
	tickerRespFns            map[string]api.OnRecvWSMsg
	markPriceRespFns         map[string]api.OnRecvWSMsg
	priceLimitRespFns        map[string]api.OnRecvWSMsg
	tradesRespFns            map[string]api.OnRecvWSMsg
	depthRespFns             map[string]api.OnRecvWSMsg
	fundingRateRespFns       map[string][]api.OnRecvWSMsg
	liquidationOrdersRespFns map[string]api.OnRecvWSMsg
	muFns                    sync.Mutex

	// 外部回调
	accountBalanceRespFn api.OnRecvWSMsg
	positionRespFn       api.OnRecvWSMsg
	ordersRespFn         api.OnRecvWSMsg
}

func (ws *WsClient) Start() {
	logger.LogImportant(wsLogPrefix, "starting...")
	ws.publicWsConn.Start(publicURL, wsLogPrefixPublic, ws.onRecvMsg)
	p1 := api.Pinger{}
	p1.Start(&ws.publicWsConn, wsLogPrefix, "ping", 25, 50)

	ws.privateWsConn.Start(privateURL, wsLogPrefixPrivate, ws.onRecvMsg)
	p2 := api.Pinger{}
	p2.Start(&ws.privateWsConn, wsLogPrefix, "ping", 25, 50)

	// 内部消息处理（Unmarshal)
	ws.rawRespFns = make(map[string]api.OnRecvWSRawMsg)
	ws.rawRespFns["tickers"] = ws.rawRespTicker
	ws.rawRespFns["mark-price"] = ws.rawRespMarkPrice
	ws.rawRespFns["price-limit"] = ws.rawRespPriceLimit
	ws.rawRespFns["trades"] = ws.rawRespTrades
	ws.rawRespFns["books"] = ws.rawRespDepth
	ws.rawRespFns["books5"] = ws.rawRespDepth
	ws.rawRespFns["books50-l2-tbt"] = ws.rawRespDepth
	ws.rawRespFns["funding-rate"] = ws.rawRespFundingRate
	ws.rawRespFns["liquidation-orders"] = ws.rawRespLiquidationOrders
	ws.rawRespFns["account"] = ws.rawRespAccountBalance
	ws.rawRespFns["positions"] = ws.rawRespPosition
	ws.rawRespFns["orders"] = ws.rawRespOrders

	// 外部消息处理(instID-callback)
	ws.tickerRespFns = make(map[string]api.OnRecvWSMsg)
	ws.markPriceRespFns = make(map[string]api.OnRecvWSMsg)
	ws.priceLimitRespFns = make(map[string]api.OnRecvWSMsg)
	ws.tradesRespFns = make(map[string]api.OnRecvWSMsg)
	ws.depthRespFns = make(map[string]api.OnRecvWSMsg)
	ws.fundingRateRespFns = make(map[string][]api.OnRecvWSMsg)
	ws.liquidationOrdersRespFns = make(map[string]api.OnRecvWSMsg)
}

// #region public channels
func (ws *WsClient) subscribePublicChannelWithInstID(channel string, instID string, fn api.OnRecvWSMsg, fnMap *map[string]api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		fmt.Sprintf("%s(%s)", channel, instID),
		fmt.Sprintf(`{"op":"subscribe","args":[{"channel":"%s","instId":"%s"}]}`, channel, instID),
		true,
		nil,
		[]string{"subscribe", channel, instID})
	ws.publicWsConn.Subscribe(&s)

	ws.muFns.Lock()
	(*fnMap)[instID] = fn
	ws.muFns.Unlock()

	return &s
}

func (ws *WsClient) unsubscribePublicChannelWithInstID(channel string, instID string) {
	s := api.WsSubscriber{}
	s.Init(
		fmt.Sprintf("%s(%s)", channel, instID),
		fmt.Sprintf(`{"op":"unsubscribe","args":[{"channel":"%s","instId":"%s"}]}`, channel, instID),
		false,
		nil,
		[]string{"unsubscribe", channel, instID})
	ws.publicWsConn.Subscribe(&s)
}

func (ws *WsClient) subscribePublicChannelWithInstType(channel string, instType string, fn api.OnRecvWSMsg, fnMap *map[string]api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		fmt.Sprintf("%s(%s)", channel, instType),
		fmt.Sprintf(`{"op":"subscribe","args":[{"channel":"%s","instType":"%s"}]}`, channel, instType),
		true,
		nil,
		[]string{"subscribe", channel, instType})
	ws.publicWsConn.Subscribe(&s)

	ws.muFns.Lock()
	(*fnMap)[instType] = fn
	ws.muFns.Unlock()

	return &s
}

func (ws *WsClient) unsubscribePublicChannelWithInstType(channel string, instType string) {
	s := api.WsSubscriber{}
	s.Init(
		fmt.Sprintf("%s(%s)", channel, instType),
		fmt.Sprintf(`{"op":"unsubscribe","args":[{"channel":"%s","instType":"%s"}]}`, channel, instType),
		false,
		nil,
		[]string{"unsubscribe", channel, instType})
	ws.publicWsConn.Subscribe(&s)
}

func (ws *WsClient) findFromFnMap(fnMap map[string]api.OnRecvWSMsg, instId string) api.OnRecvWSMsg {
	var fn api.OnRecvWSMsg
	var ok bool

	ws.muFns.Lock()
	fn, ok = fnMap[instId]
	ws.muFns.Unlock()

	if ok {
		return fn
	} else {
		return nil
	}
}

func (ws *WsClient) subscribePublicChannelWithInstIDMulti(channel string, instID string, fn api.OnRecvWSMsg, fnMap *map[string][]api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		fmt.Sprintf("%s(%s)", channel, instID),
		fmt.Sprintf(`{"op":"subscribe","args":[{"channel":"%s","instId":"%s"}]}`, channel, instID),
		true,
		nil,
		[]string{"subscribe", channel, instID})
	ws.publicWsConn.Subscribe(&s)

	(*fnMap)[instID] = append((*fnMap)[instID], fn)
	return &s
}

// 行情数据
func (ws *WsClient) SubscribeTicker(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("tickers", instID, fn, &ws.tickerRespFns)
	return s
}

func (ws *WsClient) UnsubscribeTicker(instID string) {
	ws.unsubscribePublicChannelWithInstID("tickers", instID)
}

// 标记价格
func (ws *WsClient) SubscribeMarkPrice(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("mark-price", instID, fn, &ws.markPriceRespFns)
	return s
}

func (ws *WsClient) UnubscribeMarkPrice(instID string) {
	ws.unsubscribePublicChannelWithInstID("mark-price", instID)
}

// 限价
func (ws *WsClient) SubscribePriceLimit(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("price-limit", instID, fn, &ws.priceLimitRespFns)
	return s
}

func (ws *WsClient) UnsubscribePriceLimit(instID string) {
	ws.unsubscribePublicChannelWithInstID("price-limit", instID)
}

// 成交数据
func (ws *WsClient) SubscribeTrades(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("trades", instID, fn, &ws.tradesRespFns)
	return s
}

func (ws *WsClient) UnsubscribeTrades(instID string) {
	ws.unsubscribePublicChannelWithInstID("trades", instID)
}

// 深度数据(400档增量)
func (ws *WsClient) SubscribeDepth(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("books", instID, fn, &ws.depthRespFns)
	return s
}

func (ws *WsClient) UnsubscribeDepth(instID string) {
	ws.unsubscribePublicChannelWithInstID("books", instID)
}

// 深度数据(5档)
func (ws *WsClient) SubscribeDepth5(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("books5", instID, fn, &ws.depthRespFns)
	return s
}

func (ws *WsClient) UnsubscribeDepth5(instID string) {
	ws.unsubscribePublicChannelWithInstID("books5", instID)
}

// 深度数据(50档，需要vip4)
func (ws *WsClient) SubscribeDepth50tbt(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstID("books50-l2-tbt", instID, fn, &ws.depthRespFns)
	return s
}

func (ws *WsClient) UnsubscribeDepth50tbt(instID string) {
	ws.unsubscribePublicChannelWithInstID("books50-l2-tbt", instID)
}

// 资金费率
func (ws *WsClient) SubscribeFundingrate(instID string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstIDMulti("funding-rate", instID, fn, &ws.fundingRateRespFns)
	return s
}

func (ws *WsClient) UnsubscribeFundingrate(instID string) {
	ws.unsubscribePublicChannelWithInstID("funding-rate", instID)
}

// 市场爆仓(这个频道根据instType订阅，而不是instId。这里用instType代替instId)
func (ws *WsClient) SubscribeLiquidationOrders(instType string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := ws.subscribePublicChannelWithInstType("liquidation-orders", instType, fn, &ws.liquidationOrdersRespFns)
	return s
}

func (ws *WsClient) UnsubscribeLiquidationOrders(instType string) {
	ws.unsubscribePublicChannelWithInstType("liquidation-orders", instType)
}

// #endregion

// #region private channels
func (ws *WsClient) loginStrGen() string {
	sign, timeStamp := signerIns.signWithUnix11Ts("GET", "/users/self/verify", "")
	return fmt.Sprintf(`{"op": "login","args":[{"apiKey":"%s","passphrase":"%s","timestamp" :"%s","sign":"%s"}]}`, signerIns.key, signerIns.pass, timeStamp, sign)
}

func (ws *WsClient) Login() {
	s1 := api.WsSubscriber{}
	s1.Init(
		"login",
		"",
		true,
		ws.loginStrGen,
		[]string{`"login"`}) //{"event":"login", "msg" : "", "code": "0"}
	ws.privateWsConn.Login(&s1)

	s2 := api.WsSubscriber{}
	s2.Init(
		"login",
		"",
		true,
		ws.loginStrGen,
		[]string{`"login"`}) //{"event":"login", "msg" : "", "code": "0"}
	ws.publicWsConn.Login(&s2)
}

// 账户数据
func (ws *WsClient) SubscribeAccountBalance(fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		"account",
		`{"op": "subscribe","args": [{"channel": "account"}]}`,
		true,
		nil,
		[]string{`"event":"subscribe","arg":{"channel":"account"`})
	ws.privateWsConn.Subscribe(&s)
	ws.accountBalanceRespFn = fn
	return &s
}

func (ws *WsClient) UnsubscribeAccountBalance() {
	s := api.WsSubscriber{}
	s.Init(
		"account",
		`{"op": "unsubscribe","args": [{"channel":"account"}]}`,
		true,
		nil,
		[]string{"unsubscribe", "account"})
	ws.privateWsConn.Subscribe(&s)
}

// 仓位
func (ws *WsClient) SubscribePosition(fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		"positions",
		`{"op": "subscribe","args": [{"channel": "positions","instType": "ANY"}]}`,
		true,
		nil,
		[]string{`"event":"subscribe","arg":{"channel":"positions"`})
	ws.privateWsConn.Subscribe(&s)
	ws.positionRespFn = fn
	return &s
}

func (ws *WsClient) UnsubscribePosition() {
	s := api.WsSubscriber{}
	s.Init(
		"positions",
		`{"op": "unsubscribe","args": [{"channel":"positions","instType":"ANY"}]}`,
		true,
		nil,
		[]string{"unsubscribe", "positions", "ANY"})
	ws.privateWsConn.Subscribe(&s)
}

// 订单
func (ws *WsClient) SubscribeOrders(fn api.OnRecvWSMsg) *api.WsSubscriber {
	s := api.WsSubscriber{}
	s.Init(
		"orders",
		`{"op": "subscribe","args": [{"channel":"orders","instType":"ANY"}]}`,
		true,
		nil,
		[]string{"subscribe", "orders", "ANY"})
	ws.privateWsConn.Subscribe(&s)
	ws.ordersRespFn = fn
	return &s
}

func (ws *WsClient) UnsubscribeOrders() {
	s := api.WsSubscriber{}
	s.Init(
		"orders",
		`{"op": "unsubscribe","args": [{"channel":"orders","instType":"ANY"}]}`,
		true,
		nil,
		[]string{"unsubscribe", "orders", "ANY"})
	ws.privateWsConn.Subscribe(&s)
}

// #endregion

// #region 消息处理
func (ws *WsClient) onRecvMsg(msg api.WSRawMsg) {
	if strings.IndexByte(msg.Str, '{') == 0 && strings.Contains(msg.Str, `"data"`) {
		// 根据channel，把消息发送到各个负责解析原始数据的chan中
		channel := util.FetchMiddleString(&msg.Str, `"arg":{"channel":"`, `"`)
		if fn, ok := ws.rawRespFns[channel]; ok && fn != nil {
			fn(msg)
		}
	}
}

func (ws *WsClient) rawRespTicker(msg api.WSRawMsg) {
	r := TickerWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		r.parse()
		fn := ws.findFromFnMap(ws.tickerRespFns, r.Arg.InstId)
		if fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespPriceLimit(msg api.WSRawMsg) {
	r := PriceLimitWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if fn := ws.findFromFnMap(ws.priceLimitRespFns, r.Arg.InstId); fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespMarkPrice(msg api.WSRawMsg) {
	r := MarkPriceWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if fn := ws.findFromFnMap(ws.markPriceRespFns, r.Arg.InstId); fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespTrades(msg api.WSRawMsg) {
	r := TradesWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if fn := ws.findFromFnMap(ws.tradesRespFns, r.Arg.InstId); fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespDepth(msg api.WSRawMsg) {
	r := DepthWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if fn := ws.findFromFnMap(ws.depthRespFns, r.Arg.InstId); fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespFundingRate(msg api.WSRawMsg) {
	r := FundingRateWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		r.parse()
		if fns, ok := ws.fundingRateRespFns[r.Arg.InstId]; ok {
			for _, fn := range fns {
				fn(r)
			}
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespLiquidationOrders(msg api.WSRawMsg) {
	r := LiquidationOrderWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		r.parse()
		if fn := ws.findFromFnMap(ws.liquidationOrdersRespFns, r.Arg.InstType); fn != nil {
			fn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespAccountBalance(msg api.WSRawMsg) {
	r := AccountBalanceWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if ws.accountBalanceRespFn != nil {
			ws.accountBalanceRespFn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespPosition(msg api.WSRawMsg) {
	r := PositionWsResp{}
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if ws.positionRespFn != nil {
			ws.positionRespFn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

func (ws *WsClient) rawRespOrders(msg api.WSRawMsg) {
	r := OrderWsResp{}
	r.LocalTime = msg.LocalTime
	err := json.Unmarshal(msg.Data, &r)
	if err == nil {
		if ws.ordersRespFn != nil {
			ws.ordersRespFn(r)
		}
	} else {
		ws.logUnmarshalError(wsLogPrefixPublic, r, err, msg.Str)
	}
}

// unmarshal 错误输出
func (ws *WsClient) logUnmarshalError(prefix string, respStruct interface{}, err error, msgstr string) {
	logger.LogImportant(
		prefix,
		"ws data unmarshal failed, type={%s}, err=%s, msg=%s",
		reflect.TypeOf(respStruct).Name(),
		err.Error(),
		msgstr)
}

// #endregion
