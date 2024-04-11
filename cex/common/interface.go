/*
 * @Author: aztec
 * @Date: 2022-03-30 13:08:33
  - @LastEditors: Please set LastEditors
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
	OrderDir_Buy
	OrderDir_Sell
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

// 深度观察者
type DepthObserver interface {
	OnDepthChanged()
}

// 成交观察者
type OrderObserver interface {
	OnDeal(d Deal)
}

// 市场爆仓观察者
type LiquidationObserver interface {
	OnLiquidation(px, sz decimal.Decimal, dir OrderDir)
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
	Ccy() string
	Rights() decimal.Decimal
	Frozen() decimal.Decimal
	Available() decimal.Decimal
}

// 合约仓位
type Position interface {
	Symbol() string
	ContractType() string
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
	TradingTime() TradingTimes // 获取近期的交易时间配置。返回nil表示24小时交易不停盘
	Ready() bool
	UnreadyReason() string
	Uninit()
	LatestPrice() decimal.Decimal
	OrderBook() *Orderbook
	AlignPriceNumber(price decimal.Decimal) decimal.Decimal
	AlignPrice(price decimal.Decimal, dir OrderDir, makeOnly bool) decimal.Decimal
	AlignSize(size decimal.Decimal) decimal.Decimal
	MinSize() decimal.Decimal
	AddDepthObserver(o DepthObserver)
	RemoveDepthObserver(o DepthObserver)
}

// 合约行情接口
type FutureMarket interface {
	CommonMarket

	Symbol() string                                                        // 币种，表示是哪种币的合约。用小写，如：btc
	ContractType() string                                                  // 合约种类，表示是哪种合约。可选：usd_swap/usdt_swap/this_week/next_week/this_quarter/next_quarter
	IsUsdtContract() bool                                                  // 是否为U本位合约
	MarkPrice() decimal.Decimal                                            // 标记价格，用于计算仓位浮动盈亏的价格。如果交易所不提供，则使用中间价代替
	ValueAmount() decimal.Decimal                                          // 单位合约面值数量
	ValueCurrency() string                                                 // 面值单位币种，usdt合约为币，usd合约为usdt
	SettlementCurrency() string                                            // 保证金币种
	FundingInfo() (decimal.Decimal, decimal.Decimal, time.Time, time.Time) // 当期费率、下期费率、当期时间
	AddLiquidationObserver(o LiquidationObserver)                          // 注册市场爆仓观察器
	RemoveLiquidationObserver(o LiquidationObserver)                       //
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
	UnreadyReason() string // 方便查看哪里没就绪
	BuyPriceRange() (min, max decimal.Decimal)
	SellPriceRange() (min, max decimal.Decimal)
	MakeOrder(price, amount decimal.Decimal, dir OrderDir, makeOnly, reduceOnly bool, purpose string, observer OrderObserver) Order
	Orders() []Order
	FeeTaker() decimal.Decimal
	FeeMaker() decimal.Decimal

	// 在某个方向上最多可交易的数量
	// 注意合约有反向仓位时，只能返回可平仓数量，不考虑新开仓数量
	AvailableAmount(dir OrderDir, price decimal.Decimal) decimal.Decimal
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
	GetFeeInfo(instId string) (FundingFeeInfo, bool)
	AllInstIds() []string
	AllFeeInfo() []FundingFeeInfo
	Ready() (float64, bool)
}

type FundingFeeInfo struct {
	InstId      string                        // 合约Id/名称
	SpotPrice   decimal.Decimal               // 当前价格
	SwapPrice   decimal.Decimal               // 当前价格
	VolUSD24h   decimal.Decimal               // 24小时成交额
	FeeRate     decimal.Decimal               // 当期费率
	FeeTime     time.Time                     // 当期费率时间
	NextFeeRate decimal.Decimal               // 下期费率
	NextFeeTime time.Time                     // 下期费率时间
	FeeHistory  map[time.Time]decimal.Decimal // 历史费率
}

func (f *FundingFeeInfo) FundingFeeOk() bool {
	return !f.FeeTime.IsZero()
}

func (f *FundingFeeInfo) NextFundingFeeOk() bool {
	return !f.NextFeeTime.IsZero()
}

func (f *FundingFeeInfo) HistoryFundingFeeOk() bool {
	return f.FeeHistory != nil && len(f.FeeHistory) > 0
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

// 统一账号整体风险。不同交易所的同一账号风险计算方式可能不同。这里统一抽象为风险等级
// 交易所在实现时，宜用配置文件来指定每个风险等级的具体标准
type UniAccRiskLevel int

const (
	UniAccRiskLevel_Safe UniAccRiskLevel = iota
	UniAccRiskLevel_Warning
	UniAccRiskLevel_Danger
)

func UniAccRiskLevel2String(l UniAccRiskLevel) string {
	switch l {
	case UniAccRiskLevel_Safe:
		return "safe"
	case UniAccRiskLevel_Warning:
		return "warning"
	case UniAccRiskLevel_Danger:
		return "danger"
	default:
		return "invalid"
	}
}

// 统一账户整体风险级别
// 所有支持统一账户的交易所都应该计算这个
type UniAccRisk struct {
	Level          UniAccRiskLevel   // 当前风险等级。所有策略逻辑仅依赖这一个值
	PositionValue  decimal.Decimal   // 仓位价值
	TotalMargin    decimal.Decimal   // 总保证金
	MaintainMargin decimal.Decimal   // 有效保证金
	Details        map[string]string // 详情，仅用于显示，不用于计算
}

// 金融接口
type Finance interface {
	GetSavingApy(ccy string) decimal.Decimal
	GetSavedBalance(ccy string) decimal.Decimal
	Save(ccy string, amount decimal.Decimal) bool
	Draw(ccy string, amount decimal.Decimal) bool
}

// 中心化交易所
// 一个CEx对应一个中心化交易所的账号
// 总管所有账号数据
// 可以创建各种行情器、交易器对象，以及转账提现等功能
// CEx相当于航母，行情器、交易器等相当于舰载机
type CEx interface {
	Name() string
	Instruments() []*Instruments
	GetSpotInstrument(baseCcy, quoteCcy string) *Instruments
	GetFutureInstrument(symbol, contractType string) *Instruments
	GetUniAccRisk() UniAccRisk

	// 创建合约行情器/交易器
	// 这里symbol/contractType采用交易所无关的统一命名
	// symbol: 币种，表示是哪种币的合约。用小写，如：btc
	// contract_type：合约种类，表示是哪种合约。参看枚举：ContractType
	// 各交易所自行转化成自己的表达方式
	FutureMarkets() []FutureMarket
	FutureTraders() []FutureTrader
	UseFutureMarket(symbol, contractType string) FutureMarket
	UseFutureTrader(symbol, contractType string, lever int) FutureTrader // lever填0表示自动设置最合适的杠杆率

	SpotMarkets() []SpotMarket
	SpotTraders() []SpotTrader
	UseSpotMarket(baseCcy, quoteCcy string) SpotMarket
	UseSpotTrader(baseCcy, quoteCcy string) SpotTrader

	// 获取金融接口
	GetFinance() Finance

	GetAllPositions() []Position
	GetAllBalances() []Balance

	UseFundingFeeInfoObserver() FundingFeeObserver
	FundingFeeInfoObserver() FundingFeeObserver

	UseContractObserver(contractType string) ContractObserver

	// 查询k线
	GetSpotKline(baseCcy, quoteCcy string, t0, t1 time.Time, intervalSec int) []KUnit
	GetFutureKline(symbol, contractType string, t0, t1 time.Time, intervalSec int) []KUnit

	// 查询历史成交记录
	GetSpotDealHistory(baseCcy, quoteCcy string, t0, t1 time.Time) []DealHistory
	GetFutureDealHistory(symbol, contractType string, t0, t1 time.Time) []DealHistory

	Exit()
}
