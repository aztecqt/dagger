/*
 * @Author: aztec
 * @Date: 2022-03-25 22:19:38
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-05-29 10:31:33
 * @FilePath: \dagger\api\okexv5api\response_public.go
 * @Description:okex的api返回数据。不对外公开，仅在包内做临时传递数据用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5api

import (
	"encoding/binary"
	"io"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type CommonWsResp struct {
	Arg struct {
		InstId   string `json:"instId"`
		InstType string `json:"instType"`
	} `json:"arg"`
}

type CommonRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// 服务器时间
type serverTimeRestResp struct {
	CommonRestResp
	Data []struct {
		TS string `json:"ts"`
	} `json:"data"`
}

// 币种信息
type Currency struct {
	Ccy               string `json:"ccy"`
	Chain             string `json:"chain"`
	MinDepositStr     string `json:"minDep"`
	MinWithdrawStr    string `json:"minWd"`
	MaxWithdrawStr    string `json:"maxWd"`
	WithdrawTickSzStr string `json:"wdTickSz"`
	MinFeeStr         string `json:"minFee"`
	MaxFeeStr         string `json:"maxFee"`

	MinDeposit     decimal.Decimal
	MinWithdraw    decimal.Decimal
	MaxWithdraw    decimal.Decimal
	WithdrawTickSz int
	MinFee         decimal.Decimal
	MaxFee         decimal.Decimal
}

func (c *Currency) Parse() {
	c.MinDeposit = util.String2DecimalPanic(c.MinDepositStr)
	c.MinWithdraw = util.String2DecimalPanic(c.MinWithdrawStr)
	c.MaxWithdraw = util.String2DecimalPanic(c.MaxWithdrawStr)
	c.WithdrawTickSz = util.String2IntPanic(c.WithdrawTickSzStr)
	c.MinFee = util.String2DecimalPanic(c.MinFeeStr)
	c.MaxFee = util.String2DecimalPanic(c.MaxFeeStr)
}

type GetCurrencyResp struct {
	CommonRestResp
	Data []Currency `json:"data"`
}

func (r *GetCurrencyResp) FindAllCurrency(ccy string) []Currency {
	ccys := make([]Currency, 0)
	for _, c := range r.Data {
		if strings.ToLower(c.Ccy) == strings.ToLower(ccy) {
			ccys = append(ccys, c)
		}
	}

	return ccys
}

func (r *GetCurrencyResp) FindCurrency(ccy, chain string) (Currency, bool) {
	for _, c := range r.Data {
		if strings.ToLower(c.Ccy) == strings.ToLower(ccy) && strings.ToLower(c.Chain) == strings.ToLower(chain) {
			return c, true
		}
	}

	return Currency{}, false
}

// 币种信息（非正式API)
type Project struct {
	Classification string  `json:"classification"`
	CurrencyId     int     `json:"currencyId"`
	DayHigh        float64 `json:"dayHigh"`
	DayLow         float64 `json:"dayLow"`
	Last           float64 `json:"last"`
	Open           float64 `json:"open"`
	Symbol         string  `json:"symbol"`
	Volume         float64 `json:"volume"`
	MarketCap      float64 `json:"marketCap"`
	Icon           string  `json:"icon"`
	BaseCurrency   string
	QuoteCurrency  string
}

func (p *Project) Parse() {
	ss := strings.Split(p.Symbol, "_")
	if len(ss) == 2 {
		p.BaseCurrency = ss[0]
		p.QuoteCurrency = ss[1]
	}
}

type GetProjectsResp struct {
	Code int `json:"code"`
	Data struct {
		List []Project `json:"list"`
	} `json:"data"`
}

func (p *GetProjectsResp) Parse() {
	for i := range p.Data.List {
		p.Data.List[i].Parse()
	}
}

type Instrument struct {
	InstID    string `json:"instID"`    // 产品类型
	InstType  string `json:"instType"`  // 产品类型 FUTURES/SWAP/SPOT/MARGIN...
	Uly       string `json:"uly"`       // 标的指数
	BaseCcy   string `json:"baseCcy"`   // 币币中的交易货币币种，如BTC-USDT中的BTC
	QuoteCcy  string `json:"quoteCcy"`  // 币币中的计价货币币种，如BTC-USDT中的USDT
	SettleCcy string `json:"settleCcy"` // 盈亏结算和保证金币种
	CtValCcy  string `json:"ctValCcy"`  // 合约面值计价币种
	CtVal     string `json:"ctVal"`     // 合约面值
	ExpTime   string `json:"expTime"`   // 交割日期（交割合约、期权）
	Lever     string `json:"lever"`     // 最大杠杆倍率
	TickSize  string `json:"tickSz"`    // 下单价格精度
	LotSz     string `json:"lotSz"`     // 下单数量精度
	MinSz     string `json:"minSz"`     // 最小下单数量
	Alias     string `json:"alias"`     // 别名(this_week/next_week/quarter/next_quarter)
	State     string `json:"state"`     // 状态：live：交易中	suspend：暂停中	expired：已过期	preopen：预上线	settlement：资金费结算
}

// 交易对信息
type InstrumentRestResp struct {
	CommonRestResp
	Data []Instrument `json:"data"`
}

// 行情
type TickerResp struct {
	InstId       string          `json:"instId"`
	InstType     string          `json:"instType"`
	Last         decimal.Decimal `json:"last"`
	Sell1        decimal.Decimal `json:"askPx"`
	Buy1         decimal.Decimal `json:"bidPx"`
	VolCcy24h    decimal.Decimal `json:"volCcy24h"`
	TimeStampStr string          `json:"ts"`
	Time         time.Time
	VolUsd24h    decimal.Decimal
}

func (t *TickerResp) parse() {
	t.Time = time.UnixMilli(util.String2Int64Panic(t.TimeStampStr))
	if t.InstType != "SPOT" {
		t.VolUsd24h = t.VolCcy24h.Mul(t.Last)
	} else {
		t.VolUsd24h = t.VolCcy24h
	}
}

type TickerWsResp struct {
	CommonWsResp
	Data []TickerResp `json:"data"`
}

func (t *TickerWsResp) parse() {
	for i := range t.Data {
		t.Data[i].parse()
	}
}

type TickerRestResp struct {
	CommonRestResp
	Data []TickerResp `json:"data"`
}

func (t *TickerRestResp) parse() {
	for i := range t.Data {
		t.Data[i].parse()
	}
}

// 指数行情
type IndexTicker struct {
	InstId     string          `json:"instId"`
	IndexPrice decimal.Decimal `json:"idxPx"`
}

type IndexTickerRestResp struct {
	CommonRestResp
	Data []IndexTicker `json:"data"`
}

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

type KLineRestResp struct {
	CommonRestResp
	DataRaw [][]string `json:"data"`
	Data    []KLineUnit
}

func (kl *KLineRestResp) Build() {
	kl.Data = make([]KLineUnit, 0, len(kl.DataRaw))
	for i := 0; i < len(kl.DataRaw); i++ {
		v := kl.DataRaw[i]
		ku := KLineUnit{
			Time:  time.UnixMilli(util.String2Int64Panic(v[0])),
			Open:  util.String2DecimalPanic(v[1]),
			High:  util.String2DecimalPanic(v[2]),
			Low:   util.String2DecimalPanic(v[3]),
			Close: util.String2DecimalPanic(v[4]),
		}

		if len(v) > 7 {
			ku.VolumeUSD = util.String2DecimalPanic(v[7])
		}

		kl.Data = append(kl.Data, ku)
	}
}

// 标记价格
type MarkPriceResp struct {
	MarkPrice string `json:"markPx"`
	TS        string `json:"ts"`
}

type MarkPriceRestResp struct {
	CommonRestResp
	Data []MarkPriceResp `json:"data"`
}

type MarkPriceWsResp struct {
	CommonWsResp
	Data []MarkPriceResp `json:"data"`
}

// 限价
type PriceLimitResp struct {
	BuyLimit  string `json:"buyLmt"`
	SellLimit string `json:"sellLmt"`
	TS        string `json:"ts"`
}

type PriceLimitRestResp struct {
	CommonRestResp
	Data []PriceLimitResp `json:"data"`
}

type PriceLimitWsResp struct {
	CommonWsResp
	Data []PriceLimitResp `json:"data"`
}

// 市场成交
type TradesWsResp struct {
	CommonWsResp
	Data []struct {
		TradeID   string          `json:"tradeId"`
		Price     decimal.Decimal `json:"px"`
		Size      decimal.Decimal `json:"sz"`
		Side      string          `json:"side"`
		TimeStamp string          `json:"ts"`
	} `json:"data"`
}

// 深度
type DepthResp struct {
	Data []struct {
		Asks      [][4]string `json:"asks"`
		Bids      [][4]string `json:"bids"`
		Checksum  int32       `json:"checksum"`
		TimeStamp string      `json:"ts"`
	} `json:"data"`
}

type DepthWsResp struct {
	CommonWsResp
	DepthResp
	Action string `json:"action"` // snapshot/update
}

type DepthRestResp struct {
	CommonRestResp
	DepthResp
}

// 当前资金费率
type FundingRate struct {
	FundingRate        decimal.Decimal `json:"fundingRate"`
	NextFundingRate    decimal.Decimal `json:"nextFundingRate"`
	FundingTimeStr     string          `json:"fundingTime"`
	NextFundingTimeStr string          `json:"nextFundingTime"`

	FundingTime     time.Time
	NextFundingTime time.Time
}

func (f *FundingRate) parse() {
	f.FundingTime = time.UnixMilli(util.String2Int64Panic(f.FundingTimeStr))
	f.NextFundingTime = time.UnixMilli(util.String2Int64Panic(f.NextFundingTimeStr))
}

type FundingRateRestResp struct {
	CommonRestResp
	Data []FundingRate `json:"data"`
}

func (f *FundingRateRestResp) parse() {
	for i := range f.Data {
		f.Data[i].parse()
	}
}

type FundingRateWsResp struct {
	CommonWsResp
	Data []FundingRate `json:"data"`
}

func (f *FundingRateWsResp) parse() {
	for i := range f.Data {
		f.Data[i].parse()
	}
}

// 历史资金费率
type FundingRateHistory struct {
	FundingRate         decimal.Decimal `json:"fundingRate"`
	RealizedRate        decimal.Decimal `json:"realizedRate"`
	FundingTimeStampStr string          `json:"fundingTime"`
	Method              string          `json:"method"` // current_period/next_period
	FundingTimeStamp    int64
	FundingTime         time.Time
}

func (f *FundingRateHistory) parse() {
	f.FundingTimeStamp = util.String2Int64Panic(f.FundingTimeStampStr)
	f.FundingTime = time.UnixMilli(f.FundingTimeStamp)
}

type FundingRateHistoryRestResp struct {
	CommonRestResp
	Data []FundingRateHistory `json:"data"`
}

func (f *FundingRateHistoryRestResp) parse() {
	for i := range f.Data {
		f.Data[i].parse()
	}
}

// 市场成交
type MarketTrads struct {
	InstId       string          `json:"instId"`
	TradeIdStr   string          `json:"tradeId"`
	Price        decimal.Decimal `json:"px"`
	Size         decimal.Decimal `json:"sz"`
	Side         string          `json:"side"`
	TimeStampStr string          `json:"ts"`
	TradeId      int64
	TimeStamp    int64
}

func (m *MarketTrads) Parse() {
	m.TradeId = util.String2Int64Panic(m.TradeIdStr)
	m.TimeStamp = util.String2Int64Panic(m.TimeStampStr)
}

type GetMarketTradesResp struct {
	CommonRestResp
	Data []MarketTrads `json:"data"`
}

func (r *GetMarketTradesResp) Parse() {
	for i := range r.Data {
		r.Data[i].Parse()
	}
}

// 爆仓信息（外部API）
type LiquidationOrderExt struct {
	BrokenLost  decimal.Decimal `json:"bkLoss"` //穿仓损失
	BrokenPrice decimal.Decimal `json:"bkPx"`   // 破产价格
	Side        string          `json:"side"`   // buy/sell
	Price       decimal.Decimal `json:"price"`  // 成交价格？
	Size        decimal.Decimal `json:"sz"`     // 数量
	TimeStamp   int64           `json:"time"`   // 时间
	Time        time.Time
}

func (l *LiquidationOrderExt) parse() {
	l.Time = time.UnixMilli(l.TimeStamp)
}

type GetLiquidationOrdersExtRest struct {
	CommonRestResp
	Data []struct {
		InstrumentId  string                `json:"instId"`
		ContractValue decimal.Decimal       `json:"ctVal"`
		Details       []LiquidationOrderExt `json:"details"`
	} `json:"data"`
}

func (g *GetLiquidationOrdersExtRest) parse() {
	for i := range g.Data {
		for i2 := range g.Data[i].Details {
			g.Data[i].Details[i2].parse()
		}
	}
}

// 爆仓信息（内部API）
type LiquidationOrderDetial struct {
	BrokenLost   decimal.Decimal `json:"bkLoss"` // 穿仓损失
	BrokenPrice  decimal.Decimal `json:"bkPx"`   // 破产价格
	Side         string          `json:"side"`   // buy/sell
	Size         decimal.Decimal `json:"sz"`     // 数量
	TimeStampStr string          `json:"ts"`     // 强平发生时间
	Time         time.Time
}

func (l *LiquidationOrderDetial) parse() {
	l.Time = time.UnixMilli(util.String2Int64Panic(l.TimeStampStr))
}

type LiquidationOrderWsResp struct {
	CommonWsResp
	Data []struct {
		InstId  string                   `json:"instId"`
		Details []LiquidationOrderDetial `json:"details"`
	} `json:"data"`
}

func (l *LiquidationOrderWsResp) parse() {
	for _, data := range l.Data {
		for i := range data.Details {
			data.Details[i].parse()
		}
	}
}

// defi质押项目
type FinanceDefiStakingOffer struct {
	Ccy          string          `json:"ccy"`          // 币种
	ProductId    string          `json:"productId"`    // 项目Id
	Protocol     string          `json:"protocol"`     // 项目名称
	ProtocolType string          `json:"protocolType"` // staking：简单赚币定期/defi：链上赚币
	Term         string          `json:"term"`         // 项目期限,活期为0，其他则显示定期天数
	Apy          decimal.Decimal `json:"apy"`          // 年化
	EarlyRedeem  bool            `json:"earlyRedeem"`  // 是否支持提前赎回
	InvestData   []struct {
		Ccy       string          `json:"ccy"`
		Balance   decimal.Decimal `json:"bal"`
		MinAmount decimal.Decimal `json:"minAmt"` // 最小申购量
		MaxAmount decimal.Decimal `json:"maxAmt"` // 最大申购量
	} `json:"investData"`
	EarningData []struct {
		Ccy         string `json:"ccy"`
		EarningType string `json:"earningType"` // 收益类型 0：预估收益 1：累计发放收益
	} `json:"earningData"`
	State string `json:"state"` // 项目状态	purchasable：可申购 sold_out：售罄 stop：暂停申购
}

type FinanceDefiStakingOffersResp struct {
	CommonRestResp
	Data []FinanceDefiStakingOffer `json:"data"`
}

// 币种折算率等级
type DiscountInfoUnit struct {
	DiscountRate  decimal.Decimal `json:"discountRate"`
	MaxAmount     decimal.Decimal `json:"maxAmt"`
	MinAmount     decimal.Decimal `json:"minAmt"`
	DiscountRateF float64
	MaxAmountF    float64
	MinAmountF    float64
}

func (u *DiscountInfoUnit) parse() {
	u.DiscountRateF = u.DiscountRate.InexactFloat64()
	u.MaxAmountF = u.MaxAmount.InexactFloat64()
	u.MinAmountF = u.MinAmount.InexactFloat64()
}

type DiscountInfo struct {
	Ccy           string             `json:"ccy"`
	DiscountLvStr string             `json:"discountLv"`
	DiscountInfo  []DiscountInfoUnit `json:"discountInfo"`
	DiscountLv    int
}

func (d *DiscountInfo) parse() {
	d.DiscountLv, _ = util.String2Int(d.DiscountLvStr)
	for i := range d.DiscountInfo {
		d.DiscountInfo[i].parse()
	}
}

func (d *DiscountInfo) CalEq(valueUsd float64) float64 {
	// 无数据表示折扣率为0，持有多少都相当于0
	if d == nil {
		return 0
	}

	// 负数资产不计算折算率
	if valueUsd < 0 {
		return valueUsd
	}

	eq := 0.0
	for _, di := range d.DiscountInfo {
		if di.MaxAmountF > 0 && di.MaxAmountF <= valueUsd {
			eq += (di.MaxAmountF - di.MinAmountF) * di.DiscountRateF
		} else {
			eq += (valueUsd - di.MinAmountF) * di.DiscountRateF
			break
		}
	}
	return eq
}

type DiscountInfoResp struct {
	CommonRestResp
	Data []DiscountInfo `json:"data"`
}

func (r *DiscountInfoResp) parse() {
	for i := range r.Data {
		r.Data[i].parse()
	}
}
