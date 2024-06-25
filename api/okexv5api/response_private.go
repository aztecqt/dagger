/*
 * @Author: aztec
 * @Date: 2022-04-06 13:19:27
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2024-06-19 13:06:55
 * @FilePath: \stratergyc:\work\svn\quant\go\src\dagger\api\okexv5api\response_private.go
 * @Description:okex的api返回数据。不对外公开，仅在包内做临时传递数据用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5api

import (
	"strconv"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// 账号等级
type AccLevel string

const (
	AccLevel_NoMargin  AccLevel = "1" // 非保证金模式
	AccLevel_SingleCcy          = "2" // 单币种保证金模式
	AccLevel_MultiCcy           = "3" // 跨币种保证金模式
	AccLevel_Portfolio          = "4" // 组合保证金模式
)

// 现货对冲模式
type SpotOffsetType string

const (
	SpotOffsetType_Usdt SpotOffsetType = "1" // 现货对冲模式：USDT
	SpotOffsetType_Coin                = "2" // 现货对冲模式：币
	SpotOffsetType_None                = "3" // 非现货对冲模式
)

// 交易模式
type TradeMode string

const (
	TradeMode_Cross    TradeMode = "cross"    // 全仓
	TradeMode_Cash               = "cash"     // 非保证金模式
	TradeMode_Isolated           = "isolated" // 逐仓
)

// 仓位模式
type PositionMode string

const (
	PositonMode_LS   PositionMode = "long_short_mode" // 开平仓模式
	PositionMode_Net              = "net_mode"
)

// 查询账户配置
type AccountConfig struct {
	UID            string `json:"uid"`
	AccLevel       string `json:"acctLv"`         // 见AccLevel
	PosMode        string `json:"posMode"`        // 仓位模式，long_short_mode/net_mode
	Level          string `json:"level"`          // 真实等级
	LevelTmp       string `json:"levelTmp"`       // 体验等级
	AutoLoan       bool   `json:"autoLoan"`       // 自动借币
	SpotOffsetType string `json:"spotOffsetType"` // 现货对冲模式：1=U模式，2=币模式，3=非现货对冲
}

type AccountConfigRestResp struct {
	Code string          `json:"code"`
	Data []AccountConfig `json:"data"`
}

// 设置/获取杠杆倍率返回
type GetSetLeverageRestResp struct {
	CommonRestResp
	Data []struct {
		LeverStr string `json:"lever"`
		MgnMode  string `json:"mgnMode"`
		InstId   string `json:"instId"`
		PosSide  string `json:"posSide"`
		Lever    int
	} `json:"data"`
}

func (r *GetSetLeverageRestResp) parse() {
	for i, _ := range r.Data {
		r.Data[i].Lever = util.String2IntPanic(r.Data[i].LeverStr)
	}
}

// 账户资产信息（交易账户）
type AccountBalanceResp struct {
	AdjEq          decimal.Decimal `json:"adjEq"`       // 有效保证金
	MaintainMargin decimal.Decimal `json:"mmr"`         // 维持保证金
	MarginRatio    decimal.Decimal `json:"mgnRatio"`    // 维持保证金率
	PositionValue  decimal.Decimal `json:"notionalUsd"` // 仓位总价值（除以有效保证金=杠杆率）
	TotalEq        decimal.Decimal `json:"totalEq"`     // 总权益

	Details []struct {
		Currency string `json:"ccy"`
		UTime    string `json:"uTime"`
		Eq       string `json:"eq"`
		Frozen   string `json:"frozenBal"`
		CashBal  string `json:"cashBal"`
	} `json:"details"`
}

type AccountBalanceRestResp struct {
	CommonRestResp
	Data []AccountBalanceResp `json:"data"`
}

type AccountBalanceWsResp struct {
	Data []AccountBalanceResp `json:"data"`
}

// 账户资产信息（资金账户）
type AssetBalanceResp struct {
	Currency  string `json:"ccy"`
	Balance   string `json:"bal"`
	Frozen    string `json:"frozenBal"`
	Available string `json:"availBal"`
}

type AssetBalanceRestResp struct {
	CommonRestResp
	Data []AssetBalanceResp `json:"data"`
}

// 最大可买卖/开仓（组合保证金模式下，衍生品全仓模式不支持）
type MaxSizeResp struct {
	Ccy           string          `json:"ccy"`
	InstId        string          `json:"instId"`
	AvailableBuy  decimal.Decimal `json:"maxBuy"`
	AvailableSell decimal.Decimal `json:"maxSell"`
}

type MaxSizeRestResp struct {
	CommonRestResp
	Data []MaxSizeResp `json:"data"`
}

// 最大可用
type MaxAvailableSizeResp struct {
	InstId        string          `json:"instId"`
	AvailableBuy  decimal.Decimal `json:"availBuy"`
	AvailableSell decimal.Decimal `json:"availSell"`
}

type MaxAvailableSizeRestResp struct {
	CommonRestResp
	Data []MaxAvailableSizeResp `json:"data"`
}

// 划转请求
type TransferReq struct {
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"`
	ClientId string `json:"clientId"`
}

// 划转结果
type TransferResp struct {
	TransId  string `json:"transId"`
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	From     string `json:"from"`
	To       string `json:"to"`
	ClientId string `json:"clientId"`
}

type TransferRestResp struct {
	CommonRestResp
	Data []AssetBalanceResp `json:"data"`
}

// 提币请求
type WithdrawReq struct {
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	Dest     string `json:"dest"` // 3=内部转账，4=链上提币
	ToAddr   string `json:"toAddr"`
	Fee      string `json:"fee"`
	Chain    string `json:"chain"`
	AreaCode string `json:"areaCode"` // 内部转账填写
	ClientId string `json:"clientId"`
}

// 提币结果返回
type WithdrawResp struct {
	CommonRestResp
}

// 查询提币返回
/*-3：撤销中
-2：已撤销
-1：失败
0：等待提币
1：提币中
2：提币成功
7: 审核通过
10: 等待划转
4, 5, 6, 8, 9, 12: 等待客服审核*/
type WithdrawStatus struct {
	ClientId string `json:"clientId"`
	State    string `json:"state"`
}
type WithdrawHistoryResp struct {
	CommonRestResp
	Data []WithdrawStatus `json:"data"`
}

// 仓位
type PositionUnit struct {
	InstType string `json:"instType"`
	MgnMode  string `json:"mgnMode"`
	PosSide  string `json:"posSide"`
	InstId   string `json:"instId"`
	TradeId  string `json:"tradeId"`
	UTime    string `json:"uTime"`
	Pos      string `json:"pos"`
	AvailPos string `json:"availPos"`
	AvgPx    string `json:"avgPx"`
	LiqPx    string `json:"liqPx"`
	MarkPx   string `json:"markPx"`
}

type PositionWsResp struct {
	Data []PositionUnit `json:"data"`
}

type PositionRestResp struct {
	CommonRestResp
	Data []PositionUnit `json:"data"`
}

// 账单类型
const BillTypeRawString = `1：划转
2：交易
3：交割
4：自动换币
5：强平
6：保证金划转
7：扣息
8：资金费
9：自动减仓
10：穿仓补偿
11：系统换币
12：策略划拨
13：对冲减仓
14：大宗交易
15：一键借币
22：一键还债
24：价差交易
250：跟单人分润支出
251：跟单人分润退还`

const BillSubTypeRawString = `1：买入
2：卖出
3：开多
4：开空
5：平多
6：平空
9：市场借币扣息
11：转入
12：转出
14：尊享借币扣息
160：手动追加保证金
161：手动减少保证金
162：自动追加保证金
114：自动换币买入
115：自动换币卖出
118：系统换币转入
119：系统换币转出
100：强减平多
101：强减平空
102：强减买入
103：强减卖出
104：强平平多
105：强平平空
106：强平买入
107：强平卖出
108：穿仓补偿
110：强平换币转入
111：强平换币转出
125：自动减仓平多
126：自动减仓平空
127：自动减仓买入
128：自动减仓卖出
131：对冲买入
132：对冲卖出
170：到期行权（实值期权买方）
171：到期被行权（实值期权卖方）
172：到期作废（非实值期权的买方和卖方）
112：交割平多
113：交割平空
117：交割/行权穿仓补偿
173：资金费支出
174：资金费收入
200：系统转入
201：手动转入
202：系统转出
203：手动转出
204：大宗交易买
205：大宗交易卖
206：大宗交易开多
207：大宗交易开空
208：大宗交易平多
209：大宗交易平空
210：一键借币的手动借币
211：一键借币的手动还币
212：一键借币的自动借币
213：一键借币的自动还币
16：强制还币
17：强制借币还息
224：还债转入
225：还债转出
236：兑换主流币用户账户转入
237：兑换主流币用户账户转出
250：永续分润支出
251：永续分润退还
280：现货分润支出
281：现货分润退还
270：价差交易买
271：价差交易卖
272：价差交易开多
273：价差交易开空
274：价差交易平多
275：价差交易平空
290：系统转出小额资产
`

var BillTypes map[string]string
var BillSubTypes map[string]string

// 账单
type Bill struct {
	BillId           string          `json:"billId"`   // 账单Id
	InstId           string          `json:"instId"`   // instId
	Type             string          `json:"type"`     // 账单类型
	SubType          string          `json:"subType"`  // 账单子类型
	TimeStampStr     string          `json:"ts"`       // 时间戳
	Ccy              string          `json:"ccy"`      // 账单币种
	Size             decimal.Decimal `json:"sz"`       // 数量
	Balance          decimal.Decimal `json:"bal"`      // 账户层面余额数量
	BalanceChange    decimal.Decimal `json:"balChg"`   // 账户层面余额变动
	Pnl              decimal.Decimal `json:"pnl"`      // 收益
	Fee              decimal.Decimal `json:"fee"`      // 手续费
	Price            decimal.Decimal `json:"px"`       // 价格（见文档）
	ExecType         string          `json:"execType"` //T=taker，M=maker
	Interest         decimal.Decimal `json:"interest"` // 利息
	OrderIdStr       string          `json:"ordId"`    // 订单ID
	From             string          `json:"from"`     // 资金划转来源
	To               string          `json:"to"`       // 资金划转去向
	FillTimeStampStr string          `json:"fillTime"` // 成交时间
	TradeIdStr       string          `json:"tradeId"`  // 成交Id
	ClOrdId          string          `json:"clOrdId"`  // 自定义订单Id

	Time        time.Time
	FillTime    time.Time
	OrderId     int64
	TradeId     int64
	TypeText    string
	SubTypeText string
	BaseCcy     string
	QuoteCcy    string
}

func (b *Bill) parse() {
	b.Ccy = strings.ToLower(b.Ccy)

	ss := strings.Split(b.InstId, "-")
	if len(ss) >= 2 {
		b.BaseCcy = strings.ToLower(ss[0])
		b.QuoteCcy = strings.ToLower(ss[1])
	}

	b.Time = time.UnixMilli(util.String2Int64Panic(b.TimeStampStr))
	b.FillTime = time.UnixMilli(util.String2Int64Panic(b.FillTimeStampStr))
	b.OrderId = util.String2Int64Panic(b.OrderIdStr)
	b.TradeId = util.String2Int64Panic(b.TradeIdStr)

	if BillTypes == nil {
		BillTypes = make(map[string]string)
		ss0 := strings.Split(BillTypeRawString, "\n")
		for _, v := range ss0 {
			if ss1 := strings.Split(v, "："); len(ss1) == 2 {
				BillTypes[ss1[0]] = ss1[1]
			}
		}
	}

	if BillSubTypes == nil {
		BillSubTypes = make(map[string]string)
		ss0 := strings.Split(BillSubTypeRawString, "\n")
		for _, v := range ss0 {
			if ss1 := strings.Split(v, "："); len(ss1) == 2 {
				BillSubTypes[ss1[0]] = ss1[1]
			}
		}
	}

	if v, ok := BillTypes[b.Type]; ok {
		b.TypeText = v
	} else {
		b.TypeText = b.Type
	}

	if v, ok := BillSubTypes[b.SubType]; ok {
		b.SubTypeText = v
	} else {
		b.SubTypeText = b.SubType
	}
}

type BillRestResp struct {
	CommonRestResp
	Data []Bill `json:"data"`
}

func (b *BillRestResp) parse() {
	for i := range b.Data {
		b.Data[i].parse()
	}
}

// #region 订单相关
const (
	OrderStatus_Born            = "born"
	OrderStatus_Alive           = "alive"
	OrderStatus_Canceled        = "canceled"
	OrderStatus_PartiallyFilled = "partially_filled"
	OrderStatus_Filled          = "filled"
)

// 下单请求
type MakeorderRestReq struct {
	InstId        string `json:"instId"`
	TradeMode     string `json:"tdMode"`  // isolated：逐仓 cross：全仓 cash：非保证金
	ClientOrderId string `json:"clOrdId"` //
	Tag           string `json:"tag"`
	Side          string `json:"side"`    // buy sell
	PosSide       string `json:"posSide"` // long short
	OrderType     string `json:"ordType"` // limit post_only
	ReduceOnly    bool   `json:"reduceOnly"`
	Price         string `json:"px"`
	Size          string `json:"sz"`
}

// 下单返回
type MakeorderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 撤单返回
type CancelOrderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 批量撤单请求单元
type CancelBatchOrderRestReq struct {
	InstId  string `json:"instId"`
	OrderId string `json:"ordId"`
}

// 修改订单返回
type AmendOrderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 查询订单
type OrderResp struct {
	InstId        string `json:"instId"`
	OrderId       string `json:"ordId"`
	ClientOrderId string `json:"clOrdId"`
	Tag           string `json:"tag"`
	Price         string `json:"px"`
	Size          string `json:"sz"`
	AccFillSize   string `json:"accFillSz"`
	AvgPrice      string `json:"avgPx"`
	Status        string `json:"state"` // alive/canceled/partially_filled/filled
	UTime         string `json:"uTime"`
}

type OrderRestResp struct {
	CommonRestResp
	Data      []OrderResp `json:"data"`
	LocalTime time.Time
}

type OrderWsResp struct {
	LocalTime time.Time
	Data      []OrderResp `json:"data"`
}

// 查询成交
type Fills struct {
	InstType    string          `json:"instType"`
	InstId      string          `json:"instId"`
	Price       decimal.Decimal `json:"fillPx"`
	Size        decimal.Decimal `json:"fillSz"`
	Side        string          `json:"side"`
	FillTimeStr string          `json:"fillTime"`
	FillTime    time.Time
}

func (f *Fills) Parse() {
	f.FillTime = time.UnixMilli(util.String2Int64Panic(f.FillTimeStr))
}

type FillsResp struct {
	CommonRestResp
	Data []Fills `json:"data"`
}

func (f *FillsResp) parse() {
	for i := range f.Data {
		f.Data[i].Parse()
	}
}

type PositionCloseType int

const (
	PositionCloseType_None         PositionCloseType = 0
	PositionCloseType_PartlyClosed PositionCloseType = 1
	PositionCloseType_AllClosed    PositionCloseType = 2
	PositionCloseType_ForceClosed  PositionCloseType = 3
	PositionCloseType_ForceReduce  PositionCloseType = 4
	PositionCloseType_ADL          PositionCloseType = 5
)

type PositionHistory struct {
	InstType        string          `json:"instType"`
	InstId          string          `json:"instId"`
	TypeStr         string          `json:"type"`
	CreateTimeStamp string          `json:"ctime"`
	UpdateTimeStamp string          `json:"utime"`
	OpenAvgPrice    decimal.Decimal `json:"openAvgPx"`
	CloseAvgPrice   decimal.Decimal `json:"closeAvgPx"`
	OpenMaxPos      decimal.Decimal `json:"openMaxPos"`
	CloseTotalPos   decimal.Decimal `json:"closeTotalPos"`
	Fee             decimal.Decimal `json:"fee"`
	FundingFee      decimal.Decimal `json:"fundingFee"`
	RealizedPnl     decimal.Decimal `json:"realizedPnl"` // 总收益=平仓收益+资金费+手续费
	Pnl             decimal.Decimal `json:"pnl"`         // 平仓收益
	PnlRatio        decimal.Decimal `json:"pnlRatio"`    // ???计算方法未知
	Lever           decimal.Decimal `json:"lever"`
	Direction       string          `json:"direction"`
	DepositCurrency string          `json:"ccy"`

	Type       PositionCloseType
	CreateTime time.Time
	UpdateTime time.Time
}

func (p *PositionHistory) parse() {
	p.Type = PositionCloseType(util.String2IntPanic(p.TypeStr))
	p.CreateTime = time.UnixMilli(util.String2Int64Panic(p.CreateTimeStamp))
	p.UpdateTime = time.UnixMilli(util.String2Int64Panic(p.UpdateTimeStamp))
}

type PositionHistoryResp struct {
	CommonRestResp
	Data []PositionHistory `json:"data"`
}

func (p *PositionHistoryResp) parse() {
	for i := range p.Data {
		p.Data[i].parse()
	}
}

// #endregion 订单相关

// 余币宝余额
type FinanceSavingBalance struct {
	Ccy        string          `json:"ccy"`        // 币种
	Amount     decimal.Decimal `json:"amt"`        // 数量
	Earnings   decimal.Decimal `json:"earnings"`   // 持仓收益
	Rate       decimal.Decimal `json:"rate"`       // 最新出借利率
	LoanAmt    decimal.Decimal `json:"loanAmt"`    // 已出借数量
	PendingAmt decimal.Decimal `json:"pendingAmt"` // 未出借数量
}

type FinanceSavingBalanceResp struct {
	CommonRestResp
	Data []FinanceSavingBalance `json:"data"`
}

// 申购/赎回结果
type FinanceSavingPurchaseRedemptResult struct {
	Ccy    string          `json:"ccy"`  // 币种
	Amount decimal.Decimal `json:"amt"`  // 数量
	Side   string          `json:"side"` // 操作类型purchase：申购 redempt：赎回
	Rate   decimal.Decimal `json:"rate"` // 申购年利率
}

type FinanceSavingPurchageRedemptResultResp struct {
	CommonRestResp
	Date []FinanceSavingPurchaseRedemptResult `json:"data"`
}

// 市场借贷信息
type MarketLendingRateSummary struct {
	Ccy       string          `json:"ccy"`       // 币种，如 BTC
	AvgAmt    decimal.Decimal `json:"avgAmt"`    // 过去24小时平均借贷量
	AvgAmtUsd decimal.Decimal `json:"avgAmtUsd"` // 过去24小时平均借贷美元价值
	AvgRate   decimal.Decimal `json:"avgRate"`   // 过去24小时平均借出利率
	PreRate   decimal.Decimal `json:"preRate"`   // 上一次借贷年利率
	EstRate   decimal.Decimal `json:"estRate"`   // 下一次预估借贷年利率
}

type MarketLendingRateSummaryResp struct {
	CommonRestResp
	Data []MarketLendingRateSummary `json:"data"`
}

// 市场借贷利率历史
type MarketLendingRateHistory struct {
	Ccy    string          `json:"ccy"`  // 币种
	Amount decimal.Decimal `json:"amt"`  // 市场总出借数量
	Rate   decimal.Decimal `json:"rate"` // 出借年利率
	TsStr  string          `json:"ts"`   // 操作类型purchase：申购 redempt：赎回
	Time   time.Time
}

func (m *MarketLendingRateHistory) parse() {
	ts := util.String2Int64Panic(m.TsStr)
	m.Time = time.UnixMilli(ts)
}

type MarketLendingRateHistoryResp struct {
	CommonRestResp
	Date []MarketLendingRateHistory `json:"data"`
}

func (m *MarketLendingRateHistoryResp) parse() {
	for i := range m.Date {
		m.Date[i].parse()
	}
}

// 杠杆借币信息（基础)
type MarketLornInfoBasic struct {
	Ccy   string          `json:"ccy"`
	Quota decimal.Decimal `json:"quota"` // 基础借币限额
	Rate  decimal.Decimal `json:"rate"`
}

// 杠杆借币信息（用户）
type MarketLornInfoUserInfo struct {
	LoanQuotaCoef decimal.Decimal `json:"loanQuotaCoef"` // 借币限额系数
	Level         string          `json:"level"`
}

// 查询借币信息的返回结果
type MarketLoanInfoResp struct {
	CommonRestResp
	Data []struct {
		Basic   []MarketLornInfoBasic    `json:"basic"`
		Vip     []MarketLornInfoUserInfo `json:"vip"`
		Regular []MarketLornInfoUserInfo `json:"regular"`
	}
}

// 虚拟仓位计算
type PositionBuilderSimPos struct {
	InstId string          `json:"instId"`
	Pos    decimal.Decimal `json:"pos"`
}

type PositionBuilderSimAsset struct {
	Ccy    string          `json:"ccy"`
	Amount decimal.Decimal `json:"amt"`
}

// 虚拟仓位计算请求
type PositionBuilderReq struct {
	// 是否代入已有仓位和资产，默认true
	InclRealPosAndEq bool `json:"inclRealPosAndEq"`

	// 现货对冲模式
	// 1：现货对冲模式U模式
	// 2：现货对冲模式币模式
	// 3：衍生品模式
	// 默认是3
	SpotOffsetType string `json:"spotOffsetType"`

	// 模拟仓位列表
	SimPos []PositionBuilderSimPos `json:"simPos"`

	// 模拟资产
	SimAsset []PositionBuilderSimAsset `json:"simAsset"`
}

func (p *PositionBuilderReq) GetSpotAmount(ccy string) decimal.Decimal {
	ccy = strings.ToUpper(ccy)
	for _, ass := range p.SimAsset {
		if ass.Ccy == ccy {
			return ass.Amount
		}
	}

	return decimal.Zero
}

func NewPositionBuilderReq(useReal bool, spotOffsetType int) PositionBuilderReq {
	return PositionBuilderReq{
		InclRealPosAndEq: useReal,
		SpotOffsetType:   strconv.FormatInt(int64(spotOffsetType), 10),
		SimPos:           []PositionBuilderSimPos{},
		SimAsset:         []PositionBuilderSimAsset{},
	}
}

// 虚拟仓位计算结果
type PositionBuilderAsset struct {
	Ccy       string          `json:"ccy"`
	AvailEq   decimal.Decimal `json:"availEq"`
	SpotInUse decimal.Decimal `json:"spotInUse"`
	BorrowMmr decimal.Decimal `json:"borrowMmr"`
	BorrowImr decimal.Decimal `json:"borrowImr"`
}

type PositionBuilderRiskUnit struct {
	RiskUnit   string          `json:"riskUnit"`
	MMR        decimal.Decimal `json:"mmr"`
	IMR        decimal.Decimal `json:"imr"`
	Portfolios []struct {
		InstId string          `json:"instId"`
		Amount decimal.Decimal `json:"amt"`
	} `json:"portfolios"`
}

type PositionBuilderResult struct {
	Equity      decimal.Decimal           `json:"eq"`
	TotalMmr    decimal.Decimal           `json:"totalMmr"`
	TotalImr    decimal.Decimal           `json:"totalImr"`
	BorrowMmr   decimal.Decimal           `json:"borrowMmr"`
	DerivMmr    decimal.Decimal           `json:"derivMmr"`
	MarginRatio decimal.Decimal           `json:"marginRatio"`
	Assets      []PositionBuilderAsset    `json:"assets"`
	RiskUnis    []PositionBuilderRiskUnit `json:"riskUnitData"`
}
type PositionBuilderResp struct {
	CommonRestResp
	Data []PositionBuilderResult `json:"data"`
}

// 手续费率
type TradeFee struct {
	Taker     decimal.Decimal `json:"taker"`
	Maker     decimal.Decimal `json:"maker"`
	TakerUsdt decimal.Decimal `json:"takerU"`
	MakerUsdt decimal.Decimal `json:"makerU"`
	Level     string          `json:"level"`
}

type TradeFeeResp struct {
	CommonRestResp
	Data []TradeFee `json:"data"`
}
