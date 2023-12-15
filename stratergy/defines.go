/*
 * @Author: aztec
 * @Date: 2022-04-11 09:24:40

 * @FilePath: \stratergyc:\work\svn\go\src\dagger\stratergy\defines.go
 * @Description: 策略数据层的结构体定义，用于把策略核心数据导出为外部可识别的格式
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package stratergy

import (
	"reflect"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

const (
	Key_Brief    = "brief"
	Key_Observer = "observers"
	Key_Params   = "params"
	Key_Detail   = "detail"
)

type OrderExport struct {
	ID         string `json:"id"`
	CID        string `json:"cid"`
	Dir        int    `json:"dir"`
	Price      string `json:"px"`
	Size       string `json:"sz"`
	Filled     string `json:"filled"`
	AvgPrice   string `json:"avgpx"`
	BornTime   int64  `json:"btime"`
	UpdateTime int64  `json:"utime"`
	Status     string `json:"st"`
	Finished   bool   `json:"finished"`
	Extend     string `json:"ext"`
}

func (e *OrderExport) From(o common.Order) {
	e.ID, e.CID = o.GetID()
	e.Dir = int(o.GetDir())
	e.Price = o.GetPrice().String()
	e.Size = o.GetSize().String()
	e.Filled = o.GetFilled().String()
	e.AvgPrice = o.GetAvgPrice().String()
	e.BornTime = o.GetBornTime().UnixMilli()
	e.UpdateTime = o.GetUpdateTime().UnixMilli()
	e.Status = o.GetStatus()
	e.Finished = o.IsFinished()
	e.Extend = o.GetExtend()
}

type CommonMarketExport struct {
	Type     string `json:"type"`
	Ready    bool   `json:"ready"`
	ReadyStr string `json:"readys"`
	Price    string `json:"px"`
	Buy1     string `json:"b1"`
	Sell1    string `json:"s1"`
}

func (e *CommonMarketExport) From(m common.CommonMarket) {
	e.Type = m.Type()
	e.Ready = m.Ready()
	e.ReadyStr = m.ReadyStr()
	e.Price = m.LatestPrice().String()
	e.Buy1 = m.OrderBook().Buy1().String()
	e.Sell1 = m.OrderBook().Sell1().String()
}

type FutureMarketExport struct {
	CommonMarketExport
	FundingRate     string `json:"fr"`
	NextFundingRate string `json:"nfr"`
	FundingTime     int64  `json:"ft"`
	NextFundingTime int64  `json:"nft"`
}

func (e *FutureMarketExport) From(m common.FutureMarket) {
	e.CommonMarketExport.From(m)
	fr, nfr, ft, nft := m.FundingInfo()
	e.FundingRate = fr.String()
	e.NextFundingRate = nfr.String()
	e.FundingTime = ft.UnixMilli()
	e.NextFundingTime = nft.UnixMilli()
}

type SpotMarketExport struct {
	CommonMarketExport
	BaseCurrency  string `json:"base_ccy"`
	QuoteCurrency string `json:"quote_ccy"`
}

func (e *SpotMarketExport) From(m common.SpotMarket) {
	e.CommonMarketExport.From(m)
	e.BaseCurrency = m.BaseCurrency()
	e.QuoteCurrency = m.QuoteCurrency()
}

type CommonTraderExport struct {
	Ready    bool          `json:"ready"`
	ReadyStr string        `json:"readys"`
	Orders   []OrderExport `json:"orders"`
}

func (e *CommonTraderExport) From(t common.CommonTrader) {
	e.Ready = t.Ready()
	e.ReadyStr = t.ReadyStr()
	orders := t.Orders()
	e.Orders = make([]OrderExport, 0, len(orders))
	for _, o := range orders {
		oe := OrderExport{}
		oe.From(o)
		e.Orders = append(e.Orders, oe)
	}
}

type FutureTraderExport struct {
	CommonTraderExport
	Market   FutureMarketExport `json:"market"`
	Rights   string             `json:"rights"`
	Frozen   string             `json:"frozen"`
	Lever    int                `json:"lever"`
	LongPos  string             `json:"pos_l"`
	ShortPos string             `json:"pos_s"`
}

func (e *FutureTraderExport) From(t common.FutureTrader) {
	e.CommonTraderExport.From(t)

	e.Market = FutureMarketExport{}
	e.Market.From(t.FutureMarket())

	e.Rights = t.Balance().Rights().String()
	e.Frozen = t.Balance().Frozen().String()
	e.Lever = t.Lever()
	e.LongPos = t.Position().Long().String()
	e.ShortPos = t.Position().Short().String()
}

type SpotTraderExport struct {
	CommonTraderExport
	Market       SpotMarketExport `json:"market"`
	BaseBalance  string           `json:"base_bal"`
	QuoteBalance string           `json:"quote_bal"`
}

func (e *SpotTraderExport) From(t common.SpotTrader) {
	e.CommonTraderExport.From(t)

	e.Market = SpotMarketExport{}
	e.Market.From(t.SpotMarket())

	e.BaseBalance = t.BaseBalance().Rights().String()
	e.QuoteBalance = t.QuoteBalance().Rights().String()
}

type FundingFeeInfoExport struct {
	RefreshTime int64                     `json:"rt"`
	PriceRatio  decimal.Decimal           `json:"pr"`
	VolUSD24h   decimal.Decimal           `json:"volusd24h"`
	FeeRate     decimal.Decimal           `json:"fr"`
	FeeTime     int64                     `json:"ft"`
	NextFeeRate decimal.Decimal           `json:"nfr"`
	NextFeeTime int64                     `json:"nft"`
	FeeHistory  map[int64]decimal.Decimal `json:"fhistory"`
}

func (f *FundingFeeInfoExport) from(ffi common.FundingFeeInfo) {
	f.RefreshTime = ffi.RefreshTime.UnixMilli()
	f.PriceRatio = ffi.PriceRatio
	f.VolUSD24h = ffi.VolUSD24h
	f.FeeRate = ffi.FeeRate
	f.FeeTime = ffi.FeeTime.UnixMilli()
	f.NextFeeRate = ffi.NextFeeRate
	f.NextFeeTime = ffi.NextFeeTime.UnixMilli()
	f.FeeHistory = make(map[int64]decimal.Decimal)
	for t, d := range ffi.FeeHistory {
		f.FeeHistory[t.UnixMilli()] = d
	}
}

type FundingFeeObserverExport struct {
	FeeInfos map[string]FundingFeeInfoExport `json:"fee_infos"`
}

func (f *FundingFeeObserverExport) from(ffo common.FundingFeeObserver) {
	fis := ffo.AllFeeInfo()
	f.FeeInfos = make(map[string]FundingFeeInfoExport)
	for _, fi := range fis {
		ffie := FundingFeeInfoExport{}
		ffie.from(fi)
		f.FeeInfos[fi.TradeType] = ffie
	}
}

type CExExport struct {
	Name     string                        `json:"name"`
	FTraders map[string]FutureTraderExport `json:"ftraders"`
	STraders map[string]SpotTraderExport   `json:"straders"`
	FFOE     FundingFeeObserverExport      `json:"funding_fee"`
}

func (e *CExExport) From(ex common.CEx) {
	e.Name = ex.Name()

	fts := ex.FutureTraders()
	e.FTraders = make(map[string]FutureTraderExport)
	for _, ft := range fts {
		fte := FutureTraderExport{}
		fte.From(ft)
		e.FTraders[fte.Market.Type] = fte
	}

	sts := ex.SpotTraders()
	e.STraders = make(map[string]SpotTraderExport)
	for _, st := range sts {
		ste := SpotTraderExport{}
		ste.From(st)
		e.STraders[ste.Market.Type] = ste
	}

	ffo := ex.FundingFeeInfoObserver()
	if ffo != nil && !reflect.ValueOf(ffo).IsNil() {
		e.FFOE = FundingFeeObserverExport{}
		e.FFOE.from(ffo)
	}
}

// DataLineSnapshot
type DLSnapshot struct {
	Value   float64 `json:"val"`
	Version int     `json:"ver"`
}

type Brief struct {
	ClassName string `json:"class"`
	TimeStamp int64  `json:"ts"`
}

type DataLineBrief struct {
	DatalineIntervalMs int                   `json:"dl_interval"`
	DatalineMaxLength  int                   `json:"dl_max_length"`
	DatalineSnapshot   map[string]DLSnapshot `json:"dl_snapshot"`
}

type Detail struct {
	Class        string                 `json:"class"`
	Observed     bool                   `json:"observed"`
	TimeStamp    int64                  `json:"ts"`
	ParamVersion int                    `json:"p_ver"`
	Status       interface{}            `json:"status"`
	Exchanges    map[string]interface{} `json:"exchanges"`
}

type Observer struct {
	AliveTimeMs int64 `json:"ts"`
	ParamVer    int   `json:"p_ver"`
}

type Deal struct {
	DealType     string          `json:"dtype"` // 如费率策略中可以为"spot","ct","hedge"，在influx中作为field名
	ExchangeName string          `json:"ex"`
	InstId       string          `json:"inst_id"`
	TimeStempMs  int64           `json:"ts"`
	Price        decimal.Decimal `json:"px"`
	Amount       decimal.Decimal `json:"amt"`
	Dir          int             `json:"dir"`

	TimeStampMicro int64 // 用于influx避免重复
}

type ChDeal chan Deal

func (d *Deal) From(deal common.Deal) {
	d.ExchangeName = deal.O.GetExchangeName()
	d.InstId = deal.O.GetType()
	d.TimeStampMicro = time.Now().UnixMicro()
	d.TimeStempMs = time.Now().UnixMilli()
	d.Price = deal.Price
	d.Amount = deal.Amount
	d.Dir = int(deal.O.GetDir())
}
