/*
 * @Author: aztec
 * @Date: 2022-03-30 13:08:33
 * @LastEditors: aztec
 * @Description: 接口定义
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package common

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderDir int

const (
	OrderDir_None OrderDir = iota
	OrderDir_Sell
	OrderDir_Buy
)

func OrderDir2Str(d OrderDir) string {
	switch d {
	case OrderDir_Buy:
		return "buy"
	case OrderDir_Sell:
		return "sell"
	case OrderDir_None:
		return "none"
	default:
		return "unknown"
	}
}

func DirIsOpposite(a, b OrderDir) bool {
	return a == OrderDir_Buy && b == OrderDir_Sell || a == OrderDir_Sell && b == OrderDir_Buy
}

func OppositeDir(dir OrderDir) OrderDir {
	if dir == OrderDir_Buy {
		return OrderDir_Sell
	} else if dir == OrderDir_Sell {
		return OrderDir_Buy
	} else {
		return OrderDir_None
	}
}

// 订单成交（实时）
type Deal struct {
	LocalTime time.Time // 该成交时间到达本地的时间戳，用于计算系统性能
	UTime     time.Time // 该成交的服务器时间戳
	O         Order
	Price     decimal.Decimal
	Amount    decimal.Decimal
}

// 订单成交（历史）
type DealHistory struct {
	Time   time.Time
	Dir    OrderDir
	Price  decimal.Decimal
	Amount decimal.Decimal
}

type DepthObserver interface {
	OnDepthChanged()
}

type OrderObserver interface {
	OnDeal(d Deal)
}

type ChDeal chan Deal  // TODO remove
type OnDeal func(Deal) // TODO remove

// 订单
// 订单需要通过trader创建
// 创建后所有操作（撤销、修改）和查询，均通过该接口完成
// 撤销和修改两个接口，使用者可以无限制调用，直到订单完结
type Order interface {
	GetID() (string, string) // id, clientId
	GetExchangeName() string // 交易所名称
	GetType() string         // 交易对名称
	String() string
	GetStatus() string
	GetExtend() string
	GetDir() OrderDir
	GetPrice() decimal.Decimal
	GetSize() decimal.Decimal
	GetFilled() decimal.Decimal
	GetUnfilled() decimal.Decimal
	GetAvgPrice() decimal.Decimal
	IsSupportModify() bool
	Modify(price, size decimal.Decimal) // 不修改的填0，不能俩都是0
	Cancel()
	GetBornTime() time.Time
	GetUpdateTime() time.Time
	IsAlive() bool
	IsFinished() bool
	HasFatalError() bool // 错误订单一定会Finished，换句话说FatalError是Finished的子集
	AddObserver(obs OrderObserver)
}

// 币种权益
type Balance interface {
	Rights() decimal.Decimal
	Frozen() decimal.Decimal
	Available() decimal.Decimal
}

// 合约仓位
type Position interface {
	Long() decimal.Decimal
	Short() decimal.Decimal
	LongAvgPx() decimal.Decimal
	ShortAvgPx() decimal.Decimal
	Net() decimal.Decimal
}

// 通用行情接口
type CommonMarket interface {
	Type() string
	String() string
	Ready() bool
	ReadyStr() string
	Uninit()
	LatestPrice() decimal.Decimal
	HistoryPrice(t0, t1 time.Time) []PriceRecord
	OrderBook() *Orderbook
	AlignPriceNumber(price decimal.Decimal) decimal.Decimal
	AlignPrice(price decimal.Decimal, dir OrderDir, makeOnly bool) decimal.Decimal
	AlignSize(size decimal.Decimal) decimal.Decimal
	MinSize() decimal.Decimal
	AddDepthObserver(obs DepthObserver)
	RemoveDepthObserver(obs DepthObserver)
}

// 合约行情接口
type FutureMarket interface {
	CommonMarket

	MarkPrice() decimal.Decimal                                            // 标记价格，用于计算仓位浮动盈亏的价格。如果交易所不提供，则使用中间价代替
	ValueAmount() decimal.Decimal                                          // 单位合约面值数量
	ValueCurrency() string                                                 // 面值单位币种，usdt合约为币，usd合约为usdt
	SettlementCurrency() string                                            // 保证金币种
	Symbol() string                                                        // 币种，表示是哪种币的合约。用小写，如：btc
	ContractType() string                                                  // 合约种类，表示是哪种合约。可选：usd_swap/usdt_swap/this_week/next_week/this_quarter/next_quarter
	FundingInfo() (decimal.Decimal, decimal.Decimal, time.Time, time.Time) // 当期费率、下期费率、当期时间
}

// 现货行情接口
type SpotMarket interface {
	CommonMarket

	BaseCurrency() string  // 交易货币币种，如 BTC-USDT 中的 BTC
	QuoteCurrency() string // 计价货币币种，如 BTC-USDT 中的USDT
}

// 通用交易接口
type CommonTrader interface {
	Uninit()
	Market() CommonMarket
	String() string
	Ready() bool
	ReadyStr() string // 方便查看哪里没就绪
	MakeOrder(price, amount decimal.Decimal, dir OrderDir, makeOnly, reduceOnly bool, purpose string, observer OrderObserver) Order
	Orders() []Order
	FeeTaker() decimal.Decimal
	FeeMaker() decimal.Decimal

	// 在某个方向上最多可交易的数量
	// 注意合约有反向仓位时，只能返回可平仓数量，不考虑新开仓数量
	AvilableAmount(dir OrderDir, price decimal.Decimal) decimal.Decimal
}

// 合约交易器接口
type FutureTrader interface {
	CommonTrader

	FutureMarket() FutureMarket
	Lever() int
	Balance() Balance
	AssetId() int // 合约保证金资产Id，不同交易器中的权益，如果是同一个Id，则认为是同一份资产
	Position() Position
}

// 现货交易接口
type SpotTrader interface {
	CommonTrader

	SpotMarket() SpotMarket
	BaseBalance() Balance
	QuoteBalance() Balance
	AssetId() int // 现货资产Id，下同。不同交易器中的权益，如果是同一个资产Id，则认为是同一份资产
}

// 全币种费率信息接口
// 独立于Market对象，单独抽象一个针对全永续合约费率监控的接口
type FundingFeeObserver interface {
	AddType(t string)
	GetFeeInfo(t string) (FundingFeeInfo, bool)
	AllTypes() []string
	AllFeeInfo() []FundingFeeInfo
}

type FundingFeeInfo struct {
	TradeType   string                        // 合约Id/名称
	RefreshTime time.Time                     // 数据刷新时间
	PriceRatio  decimal.Decimal               // 当前价格比例（现货/合约）
	VolUSD24h   decimal.Decimal               // 24小时成交额
	FeeRate     decimal.Decimal               // 当期费率
	FeeTime     time.Time                     // 当期费率时间
	NextFeeRate decimal.Decimal               // 下期费率
	NextFeeTime time.Time                     // 下期费率时间
	FeeHistory  map[time.Time]decimal.Decimal // 历史费率
}

type ContractInfo struct {
	ValueAmount   decimal.Decimal // 单位合约面值数量
	ValueCurrency string          // 面值单位币种，usdt合约为币，usd合约为usdt
	LatestPrice   decimal.Decimal // 最新价格
	Depth         Orderbook       // 深度
}

// 合约行情观察器
// 应实现为低成本/低频率方式刷新所有合约交易对的形式
type ContractObserver interface {
	Currencys() []string                      // 支持的合约币种
	GetContractInfo(ccy string) *ContractInfo // 查询某个合约信息
}

// 中心化交易所
// 一个CEx对应一个中心化交易所的账号
// 总管所有账号数据
// 可以创建各种行情器、交易器对象，以及转账提现等功能
// CEx相当于航母，行情器、交易器等相当于舰载机
type CEx interface {
	Name() string
	Instruments() []*Instruments

	// 创建合约行情器/交易器
	// 这里symbol/contractType采用交易所无关的统一命名
	// symbol: 币种，表示是哪种币的合约。用小写，如：btc
	// contract_type：合约种类，表示是哪种合约。可选：usd_swap/usdt_swap/this_week/next_week/this_quarter/next_quarter
	// 各交易所自行转化成自己的表达方式
	FutureMarkets() []FutureMarket
	FutureTraders() []FutureTrader
	UseFutureMarket(ccy string, contractType string, withDepth bool) FutureMarket
	UseFutureTrader(ccy string, contractType string, lever int) FutureTrader // lever填0表示自动设置最合适的杠杆率

	SpotMarkets() []SpotMarket
	SpotTraders() []SpotTrader
	UseSpotMarket(baseCcy string, quoteCcy string, withDepth bool) SpotMarket
	UseSpotTrader(baseCcy string, quoteCcy string) SpotTrader

	UseFundingFeeInfoObserver(maxLength int) FundingFeeObserver
	FundingFeeInfoObserver() FundingFeeObserver

	UseContractObserver(contractType string) ContractObserver

	// 查询历史成交记录
	GetFutureDealHistory(ccy, contractType string, t0, t1 time.Time) []DealHistory
	GetSpotDealHistory(baseCcy, quoteCcy string, t0, t1 time.Time) []DealHistory

	Exit()
}
