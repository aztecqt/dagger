/*
 * @Author: aztec
 * @Date: 2022-10-20
 * @Description: api响应数据结构
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package binanceapi

import (
	"encoding/binary"
	"io"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
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
	Symbol        string                   `json:"symbol"`
	Status        string                   `json:"status"`
	Status2       string                   `json:"contractStatus"`
	BaseCcy       string                   `json:"baseAsset"`
	QuoteCcy      string                   `json:"quoteAsset"`
	ContractSize  decimal.Decimal          `json:"contractSize"`
	SpotEnabled   bool                     `json:"isSpotTradingAllowed"`
	MarginEnabled bool                     `json:"isMarginTradingAllowed"`
	Filters       []map[string]interface{} `json:"filters"`
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

// 24小时价格变动
type Ticker24hr struct {
	Symbol      string          `json:"symbol"`
	LastPrice   decimal.Decimal `json:"lastPrice"`
	LastQty     decimal.Decimal `json:"lastQty"`
	OpenPrice   decimal.Decimal `json:"openPrice"`
	HighPrice   decimal.Decimal `json:"highPrice"`
	LowPrice    decimal.Decimal `json:"lowPrice"`
	Volume      decimal.Decimal `json:"volume"`
	VolumeQuote decimal.Decimal `json:"quoteVolume"`
	Count       int             `json:"count"`
}

// K线
type KLine [][]interface{}

// k线
type KLineUnit struct {
	Time      time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	VolumeUSD decimal.Decimal
}

func (ku KLineUnit) Serialize(w io.Writer) {
	binary.Write(w, binary.LittleEndian, ku.Time.UnixMilli())
	binary.Write(w, binary.LittleEndian, ku.Open.InexactFloat64())
	binary.Write(w, binary.LittleEndian, ku.High.InexactFloat64())
	binary.Write(w, binary.LittleEndian, ku.Low.InexactFloat64())
	binary.Write(w, binary.LittleEndian, ku.Close.InexactFloat64())
	binary.Write(w, binary.LittleEndian, ku.VolumeUSD.InexactFloat64())
}

func (ku *KLineUnit) Deserialize(r io.Reader) bool {
	ms := int64(0)
	if binary.Read(r, binary.LittleEndian, &ms) != nil {
		return false
	} else {
		ku.Time = time.UnixMilli(ms)
	}

	fvalue := 0.0
	if binary.Read(r, binary.LittleEndian, &fvalue) != nil {
		return false
	} else {
		ku.Open = decimal.NewFromFloat(fvalue)
	}

	fvalue = 0.0
	if binary.Read(r, binary.LittleEndian, &fvalue) != nil {
		return false
	} else {
		ku.High = decimal.NewFromFloat(fvalue)
	}

	fvalue = 0.0
	if binary.Read(r, binary.LittleEndian, &fvalue) != nil {
		return false
	} else {
		ku.Low = decimal.NewFromFloat(fvalue)
	}

	fvalue = 0.0
	if binary.Read(r, binary.LittleEndian, &fvalue) != nil {
		return false
	} else {
		ku.Close = decimal.NewFromFloat(fvalue)
	}

	fvalue = 0.0
	if binary.Read(r, binary.LittleEndian, &fvalue) != nil {
		return false
	} else {
		ku.VolumeUSD = decimal.NewFromFloat(fvalue)
	}

	return true
}

func (k *KLineUnit) FromRaw(raw []interface{}) {
	k.Time = time.UnixMilli(raw[0].(int64))
	k.Open = util.String2DecimalPanic(raw[1].(string))
	k.High = util.String2DecimalPanic(raw[4].(string))
	k.Low = util.String2DecimalPanic(raw[2].(string))
	k.Close = util.String2DecimalPanic(raw[3].(string))
	k.VolumeUSD = util.String2DecimalPanic(raw[5].(string))
}

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

// 费率信息
type FundingFee struct {
	Symbol           string          `json:"symbol"`
	FundingTimeStamp int64           `json:"fundingTime"` // 毫秒
	FundingRate      decimal.Decimal `json:"fundingRate"`
}

// 市场持仓量
type MarketHold struct {
	Pair                 string          `json:"pair"`
	ContractType         string          `json:"contractType"`
	SumOpenInterest      decimal.Decimal `json:"sumOpenInterest"`
	SumOpenInterestValue decimal.Decimal `json:"sumOpenInterestValue"`
}

// 合约杠杆分层标准
type LeverageBracket struct {
	Symbol   string `json:"symbol"` // 币本位合约使用（BTCUSD）
	Pair     string `json:"pair"`   // U本位合约使用（ETHUSDT）
	Brackets []struct {
		Bracket         int             `json:"bracket"`          // 等级
		InitialLeverage int             `json:"initialLeverage"`  // 杠杆倍率
		QtyCap          decimal.Decimal `json:"qtyCap"`           // 上限（币本位，单位：币）
		QtylFloor       decimal.Decimal `json:"qtylFloor"`        // 下限（币本位，单位：币）
		NotionalCap     decimal.Decimal `json:"notionalCap"`      // 上限（U本位，单位：U）
		NotionalFloor   decimal.Decimal `json:"notionalFloor"`    // 下限（U本位，单位：U）
		Mmr             decimal.Decimal `json:"maintMarginRatio"` // 维持保证金率
		Cum             decimal.Decimal `json:"cum"`              // 速算数。实际仓位对应档位的mmr，乘以仓位数量，减去速算数，就是维持保证金
	} `json:"brackets"`
}

// 自己成交记录(合约)
type FutureUserTrade struct {
	Id             int64           `json:"id"`
	Maker          bool            `json:"maker"`
	IsBuyer        bool            `json:"buyer"`
	Symbol         string          `json:"symbol"`
	Price          decimal.Decimal `json:"price"`
	Quantity       decimal.Decimal `json:"qty"`
	RealizedProfit decimal.Decimal `json:"realizedPnl"`
	TimeStamp      int64           `json:"time"`
	Fee            decimal.Decimal `json:"commission"`
	FeeCcy         string          `json:"commissionAsset"`
}

// 自己成交记录(现货)
type SpotUserTrade struct {
	Id        int64           `json:"id"`
	IsMaker   bool            `json:"isMaker"`
	IsBuyer   bool            `json:"isBuyer"`
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"qty"`
	TimeStamp int64           `json:"time"`
	Fee       decimal.Decimal `json:"commission"`
	FeeCcy    string          `json:"commissionAsset"`
}

// 资金流水
type AccountIncome struct {
	IncomeType string          `json:"incomeType"`
	Income     decimal.Decimal `json:"income"`
	Asset      string          `json:"asset"`
	TimeStamp  int64           `json:"time"`
	TransId    int64           `json:"tranId"`

	AssetLower string
	Time       time.Time
}

// 当前持仓
type PositionRisk struct {
	Symbol         string          `json:"symbol"`
	PositionSide   string          `json:"positionSide"`
	PositionAmount decimal.Decimal `json:"positionAmt"`
	EntryPrice     decimal.Decimal `json:"entryPrice"`
	MarkPrice      decimal.Decimal `json:"markPrice"`
	LiqPrice       decimal.Decimal `json:"liquidationPrice"`
	UnrealPnl      decimal.Decimal `json:"unRealizedProfit"`
	UpdateTime     int64           `json:"updateTime"`
}

func (a *AccountIncome) Parse() {
	a.AssetLower = strings.ToLower(a.Asset)
	a.Time = time.UnixMilli(a.TimeStamp)
}

// 下架计划
type DelistPlan struct {
	DelistTimeStamp int64    `json:"delistTime"`
	Symbols         []string `json:"symbols"`
	DelistTime      time.Time
}

func (d *DelistPlan) Parse() {
	d.DelistTime = time.UnixMilli(d.DelistTimeStamp)
}

// 最新标记价格和资金费率
type PremiumIndexResp struct {
	Symbol               string          `json:"symbol"`
	MarkPrice            decimal.Decimal `json:"markPrice"`
	IndexPrice           decimal.Decimal `json:"indexPrice"`
	LatestFr             decimal.Decimal `json:"lastFundingRate"`
	NextFundingTimeStamp int64           `json:"nextFundingTime"`
	InterestRate         decimal.Decimal `json:"interestRate"`
	TimeStamp            int64           `json:"time"`
	NextFundingTime      time.Time
	Time                 time.Time
}

func (p *PremiumIndexResp) Parse() {
	p.NextFundingTime = time.UnixMilli(p.NextFundingTimeStamp)
	p.Time = time.UnixMilli(p.TimeStamp)
}

// 质押折扣率
type CollateralRate struct {
	Asset          string          `json:"asset"`
	CollateralRate decimal.Decimal `json:"collateralRate"`
}

// 利息历史
type InterestHistory struct {
	Timestamp int64           `json:"interestAccuredTime"`
	Asset     string          `json:"asset"`
	Interest  decimal.Decimal `json:"interest"`
}

type GetInterestHistoryResp struct {
	ErrorMessage
	Rows []InterestHistory `json:"rows"`
}

// 交易手续费
type SpotTradeFee struct {
	Symbol   string          `json:"symbol"`
	MakerFee decimal.Decimal `json:"makerCommission"`
	TakerFee decimal.Decimal `json:"takerCommission"`
}

// 获取交易手续费
type GetSpotTradeFeeResp []SpotTradeFee
