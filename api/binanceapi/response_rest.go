/*
 * @Author: aztec
 * @Date: 2022-10-20
 * @Description: api响应数据结构
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package binanceapi

import (
	"time"

	"github.com/shopspring/decimal"
)

// rest错误码和错误消息
// {"code":-2014,"msg":"API-key format invalid."}
type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// 服务器时间
type ServerTime struct {
	ErrorMessage
	ServerTime int64 `json:"serverTime"`
}

// 频率限制
type ExchangeInfo_RateLimit struct {
	ErrorMessage
	TimeZone   string `json:"timezone"`
	ServerTime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType  string `json:"rateLimitType"`
		Interval       string `json:"interval"`
		IntervalNumber int    `json:"intervalNum"`
		Limit          int    `json:"limit"`
	} `json:"rateLimits"`
}

// symbol集合
type Symbol struct {
	Symbol   string                   `json:"symbol"`
	Status   string                   `json:"status"`
	BaseCcy  string                   `json:"baseAsset"`
	QuoteCcy string                   `json:"quoteAsset"`
	Filters  []map[string]interface{} `json:"filters"`
}

func (s *Symbol) FindFilterByType(ftype string) map[string]interface{} {
	for _, filter := range s.Filters {
		if t, ok := filter["filterType"]; ok {
			if t == ftype {
				return filter
			}
		}
	}

	return nil
}

type ExchangeInfo_Symbols struct {
	ErrorMessage
	TimeZone   string   `json:"timezone"`
	ServerTime int64    `json:"serverTime"`
	Symbols    []Symbol `json:"symbols"`
}

// 最新价格
type LatestPrice struct {
	ErrorMessage
	Symbol string          `json:"symbol"`
	Price  decimal.Decimal `json:"price"`
	Ts     int64           `json:"time"`
}

// 买一卖一
type BookTicker struct {
	ErrorMessage
	Symbol      string          `json:"symbol"`
	BidPrice    decimal.Decimal `json:"bidPrice"`
	BidQuantity decimal.Decimal `json:"bidQty"`
	AskPrice    decimal.Decimal `json:"askPrice"`
	AskQuantity decimal.Decimal `json:"askQty"`
	Ts          int64           `json:"time"`
}

// K线
type KLine [][]interface{}

// 账户信息
type AccountInfo struct {
	FeeRates struct {
		Maker decimal.Decimal `json:"maker"`
		Taker decimal.Decimal `json:"taker"`
	} `json:"commissionRates"`

	Timestamp int64 `json:"updateTime"`
	Balances  []struct {
		Asset  string          `json:"asset"`
		Free   decimal.Decimal `json:"free"`
		Frozen decimal.Decimal `json:"locked"`
	} `json:"balances"`
}

// 市场交易数据
type MarketTrade struct {
	Id        int64           `json:"a"`
	Price     decimal.Decimal `json:"p"`
	Quantity  decimal.Decimal `json:"q"`
	Timestamp int64           `json:"T"`
	IsSell    bool            `json:"m"`
	Foo       bool            `json:"M"`
}

// 下单返回（Ack）
type MakeOrderResponse_Ack struct {
	ErrorMessage
	Symbol          string `json:"symbol"`
	OrderID         int64  `json:"orderId"`
	ClientOrderID   string `json:"clientOrderId"`
	TransactionTime int64  `json:"transactTime"`
}

// 下单返回（Result）
type MakeOrderResponse_Result struct {
	ErrorMessage
	Symbol          string          `json:"symbol"`
	OrderID         int64           `json:"orderId"`
	ClientOrderID   string          `json:"clientOrderId"`
	TransactionTime int64           `json:"transactTime"`
	Price           decimal.Decimal `json:"price"`
	Size            decimal.Decimal `json:"origQty"`
	FilledSize      decimal.Decimal `json:"executedQty"`
	Status          string          `json:"status"`
}

// 撤单返回
type CancelOrderResponse struct {
	ErrorMessage
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
}

// 撤销交易对订单
type CancelOpenOrdersResponse []OrderStatus

// 订单状态
type OrderStatus struct {
	Symbol           string          `json:"symbol"`
	OrderId          int64           `json:"orderId"`
	ClientOrderID    string          `json:"clientOrderId"`
	Side             string          `json:"side"`
	Status           string          `json:"status"`
	RefreshTimestamp int64           `json:"updateTime"`
	Price            decimal.Decimal `json:"price"`
	Size             decimal.Decimal `json:"origQty"`
	FilledSize       decimal.Decimal `json:"executedQty"`
}

// 查询订单结果
type GetOrderResponse struct {
	ErrorMessage
	OrderStatus
	LocalTime time.Time
}

// 查询当前挂单的结果
type GetOpenOrdersResponse []OrderStatus

// ListenKey
type ListenKeyResponse struct {
	ErrorMessage
	ListenKey string `json:"listenKey"`
}
