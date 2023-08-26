/*
 * @Author: aztec
 * @Date: 2022-03-25 22:19:38
 * @LastEditors: aztec
 * @LastEditTime: 2023-05-19 08:50:22
 * @FilePath: \dagger\api\okexv5api\response_public.go
 * @Description:okex的api返回数据。不对外公开，仅在包内做临时传递数据用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5api

import (
	"strings"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type CommonWsResp struct {
	Arg struct {
		InstId string `json:"instId"`
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

// 交易对信息
type InstrumentRestResp struct {
	CommonRestResp
	Data []struct {
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
	} `json:"data"`
}

// 行情
type TickerResp struct {
	InstId    string `json:"instId"`
	Last      string `json:"last"`
	Sell1     string `json:"askPx"`
	Buy1      string `json:"bidPx"`
	VolCcy24h string `json:"volCcy24h"`
	TimeStamp string `json:"ts"`
}

type TickerWsResp struct {
	CommonWsResp
	Data []TickerResp `json:"data"`
}

type TickerRestResp struct {
	CommonRestResp
	Data []TickerResp `json:"data"`
}

// k线
type KLineUnit struct {
	TS    int64
	Open  decimal.Decimal
	High  decimal.Decimal
	Low   decimal.Decimal
	Close decimal.Decimal
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
			TS:    util.String2Int64Panic(v[0]),
			Open:  util.String2DecimalPanic(v[1]),
			High:  util.String2DecimalPanic(v[2]),
			Low:   util.String2DecimalPanic(v[3]),
			Close: util.String2DecimalPanic(v[4]),
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
		TradeID string `json:"tradeId"`
		Price   string `json:"px"`
		Size    string `json:"sz"`
		Side    string `json:"side"`
	} `json:"data"`
}

// 深度
type DepthResp struct {
	Data []struct {
		Asks     [][4]string `json:"asks"`
		Bids     [][4]string `json:"bids"`
		Checksum int         `json:"checksum"`
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
type FundingRateResp struct {
	FundingRate     string `json:"fundingRate"`
	NextFundingRate string `json:"nextFundingRate"`
	FundingTime     string `json:"fundingTime"`
	NextFundingTime string `json:"nextFundingTime"`
}

type FundingRateRestResp struct {
	CommonRestResp
	Data []FundingRateResp `json:"data"`
}

type FundingRateWsResp struct {
	CommonWsResp
	Data []FundingRateResp `json:"data"`
}

// 历史资金费率
type FundingRateHistoryResp struct {
	FundingRate string `json:"fundingRate"`
	FundingTime string `json:"fundingTime"`
}

type FundingRateHistoryRestResp struct {
	CommonRestResp
	Data []FundingRateHistoryResp `json:"data"`
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
