/*
- @Author: aztec
- @Date: 2024-03-11 11:07:49
- @Description: 外部调用twsapi时得到的返回值，由本模块自己定义
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

type RespCode int

const (
	RespCode_Ok              RespCode = iota
	RespCode_ConnectionError RespCode = iota
	RespCode_TimeOut         RespCode = iota
)

// 通用结果
type CommonResponse struct {
	RespCode
}

// 市场规则查询结果
type MarketRuleResponse struct {
	RespCode
	Err        *ErrorMsg
	MarketRule *MarketRuleMsg
}

// 订阅MarketData结果
type MarketDataResponse struct {
	RespCode
	Err *ErrorMsg
}

// 下单结果
type OrderResponse struct {
	RespCode
	Err         *ErrorMsg
	OrderStatus *OrderStatusMsg
}

// 查询contract详细信息
type ContractDetailResponse struct {
	RespCode
	Err            *ErrorMsg
	MatchedDetails []ContractDetailMsg
}

// 查询历史数据
type HisotricalDataResponse struct {
	RespCode
	Err            *ErrorMsg
	HistoricalData *HistoricalDataMsg
}
