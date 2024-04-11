/*
 * @Author: aztec
 * @Date: 2023-02-12 19:47:31
 * @Description: 币安websocket返回的各种对象
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binanceapi

import (
	"time"

	"github.com/shopspring/decimal"
)

// 公共数据
type WSPayload_Common struct {
	EventType string `json:"e"`
	TimeStamp int64  `json:"E"`
}

// 精简ticker
type WSPayload_MiniTicker struct {
	WSPayload_Common
	Pair        string          `json:"s"`
	LatestPrice decimal.Decimal `json:"c"`
	Volume      decimal.Decimal `json:"v"`
	VolumeUsd   decimal.Decimal `json:"q"`
}

// 完整ticker
type WSPayload_Ticker struct {
	WSPayload_Common
	Pair        string          `json:"s"`
	LatestPrice decimal.Decimal `json:"c"`
	Volume      decimal.Decimal `json:"v"`
	VolumeUsd   decimal.Decimal `json:"q"`
	Buy1        decimal.Decimal `json:"b"`
	Buy1Size    decimal.Decimal `json:"B"`
	Sell1       decimal.Decimal `json:"a"`
	Sell1Size   decimal.Decimal `json:"A"`
}

// 有限档深度信息
type WSPayload_Depth struct {
	Bids [][]decimal.Decimal `json:"bids"`
	Asks [][]decimal.Decimal `json:"asks"`
}

// 账户信息推送有三种Payload，分别为：
const WSPayloadEventType_AccountUpdate = "outboundAccountPosition"        // 账户更新
const WSAccountPayloadEventType_BalanceUpdate = "outboundAccountPosition" // 余额更新(暂未使用)
const WSPayloadEventType_OrderUpdate = "executionReport"                  // 订单更新

// 账户更新
type WSPayload_AccountUpdate struct {
	WSPayload_Common
	AccountUpdateTimeStamp int64 `json:"E"`
	Detail                 []struct {
		AssetName string          `json:"a"`
		Free      decimal.Decimal `json:"f"`
		Frozen    decimal.Decimal `json:"l"`
	} `json:"B"`
}

// 订单更新(跟Rest中的OrderStatus顺序、字段保持一致)
type WSPayload_OrderUpdate struct {
	WSPayload_Common
	Symblo             string          `json:"s"`
	OrderID            int64           `json:"i"`
	ClientOrderID      string          `json:"c"` // 注意，因为撤单会被BN作为一个新的订单，C表示原始id，c表示当前id
	OrignClientOrderId string          `json:"C"`
	Side               string          `json:"S"`
	Status             string          `json:"X"` // NEW/CANCELED/PARTIALLY_FILLED/FILLED/其他情况都视为订单结束
	RefreshTimeStamp   int64           `json:"T"`
	Price              decimal.Decimal `json:"p"`
	Size               decimal.Decimal `json:"q"`
	FilledSize         decimal.Decimal `json:"z"`
	FillingSize        decimal.Decimal `json:"l"`
	FillingPrice       decimal.Decimal `json:"L"`
	Fee                decimal.Decimal `json:"n"`
	FeeAsset           string          `json:"N"`
	StratergyId        int             `json:"j"`
	P                  string          `json:"P"`
	Q                  string          `json:"Q"`
	I                  int64           `json:"I"`
	LocalTime          time.Time
}

// 合约交易对状态信息流
type WsPayload_ContractInfo struct {
	WSPayload_Common
	Symbol          string `json:"s"`
	ContractType    string `json:"ct"`
	DelistTimeStamp int64  `json:"dt"`
	OpenTime        int64  `json:"ot"`
	ContractStatus  string `json:"cs"`
	Brackets        []struct {
		Bs  int             `json:"bs"`  // 层级
		Bnf decimal.Decimal `json:"bnf"` // 该层对应的名义价值下限
		Bnc decimal.Decimal `json:"bnc"` // 该层对应的名义价值上限
		Mmr decimal.Decimal `json:"mmr"` // 该层对应的维持保证金率
		Cf  decimal.Decimal `json:"cf"`  // 速算数
		Mi  decimal.Decimal `json:"mi"`  // 该层杠杆下界
		Ma  decimal.Decimal `json:"ma"`  // 该层杠杆上界
	}
}
