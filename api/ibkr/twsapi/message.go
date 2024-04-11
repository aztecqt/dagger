/*
- @Author: aztec
- @Date: 2024-02-27 18:35:31
- @Description: 本模块跟tws服务器沟通时的协议，由tws定义
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// 20240317 21:00:00 US/Eastern
// 或
// 20240317
// 这样格式的
// 只认识US/Eastern这一个时区
func parseDateTimeFormatA(str string) time.Time {
	ss := strings.Split(str, " ")
	if len(ss) == 3 {
		if ss[2] == "US/Eastern" {
			timeStr := strings.Join(ss[:2], " ")
			if t, err := time.ParseInLocation("20060102 15:04:05", timeStr, util.UsEastern); err == nil {
				return t.In(util.East8)
			} else {
				panic("parse time failed")
			}
		} else {
			panic("unknown timezone " + ss[2])
		}
	} else if len(ss) == 1 {
		if t, err := time.ParseInLocation("20060102", ss[0], util.UsEastern); err == nil {
			return t.In(util.East8)
		} else {
			panic("parse time failed")
		}
	} else {
		panic("wrong time format")
	}
}

type deserializable interface {
	deserialize(buf *bytes.Buffer)
}

type MsgHead struct {
	Version   int
	RequestId int
}

func (m *MsgHead) deserialize(buf *bytes.Buffer) {
	m.Version = readInt(buf)
	m.RequestId = readInt(buf)
}

type MsgHeadWithoutRequestId struct {
	Version int
}

func (m *MsgHeadWithoutRequestId) deserialize(buf *bytes.Buffer) {
	m.Version = readInt(buf)
}

type MsgHeadWithoutVersion struct {
	RequestId int
}

func (m *MsgHeadWithoutVersion) deserialize(buf *bytes.Buffer) {
	m.RequestId = readInt(buf)
}

type ManagedAccountsMsg struct {
	MsgHeadWithoutRequestId
	ManagedAccounts []string
}

func (m *ManagedAccountsMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	str := readString(buf)
	m.ManagedAccounts = strings.Split(str, ",")
}

type NextOrderIdMsg struct {
	MsgHeadWithoutRequestId
	NextOrderId int
}

func (m *NextOrderIdMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	m.NextOrderId = readInt(buf)
}

type ErrorMsg struct {
	MsgHeadWithoutRequestId
	RequestId               int
	ErrorCode               int
	ErrorMessage            string
	AdvancedOrderRejectJson string
}

func (m *ErrorMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	if m.Version < 2 {
		m.ErrorMessage = readString(buf)
	} else {
		m.RequestId = readInt(buf)
		m.ErrorCode = readInt(buf)
		m.ErrorMessage = readStringUnquote(buf)
		m.AdvancedOrderRejectJson = readStringUnquote(buf)
	}
}

type AccountSummaryMsg struct {
	MsgHead
	Account  string
	Tag      string
	Value    string
	Currency string
}

func (m *AccountSummaryMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.Account = readString(buf)
	m.Tag = readString(buf)
	m.Value = readString(buf)
	m.Currency = readString(buf)
}

type AccountSummaryEndMsg struct {
	MsgHead
}

type AccountValueMsg struct {
	MsgHeadWithoutRequestId
	Key         string
	Value       string
	Currency    string
	AccountName string
}

func (m *AccountValueMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	m.Key = readString(buf)
	m.Value = readString(buf)
	m.Currency = readString(buf)
	m.AccountName = readString(buf)
}

type AccountDownloadEndMsg struct {
	MsgHeadWithoutRequestId
	AccountName string
}

func (m *AccountDownloadEndMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	m.AccountName = readString(buf)
}

type PortfolioValueMsg struct {
	MsgHeadWithoutRequestId
	Contract       twsmodel.Contract
	Position       decimal.Decimal
	MarketPrice    float64
	MarketValue    float64
	AvgPrice       float64
	UnreallizedPnl float64
	RealizedPnl    float64
	AccountName    string
}

func (m *PortfolioValueMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	m.deserializeContract(buf)
	m.Position = readDecimal(buf)
	m.MarketPrice = readFloat64(buf)
	m.MarketValue = readFloat64(buf)
	m.AvgPrice = readFloat64(buf)
	m.UnreallizedPnl = readFloat64(buf)
	m.RealizedPnl = readFloat64(buf)
	m.AccountName = readString(buf)
}

func (m *PortfolioValueMsg) deserializeContract(buf *bytes.Buffer) {
	m.Contract.ConId = readInt(buf)
	m.Contract.Symbol = readString(buf)
	m.Contract.SecType = readString(buf)
	m.Contract.LastTradeDateOrContractMonth = readString(buf)
	m.Contract.Strike = readFloat64(buf)
	m.Contract.Right = readString(buf)
	m.Contract.Multiplier = readString(buf)
	m.Contract.PrimaryExch = readString(buf)
	m.Contract.Currency = readString(buf)
	m.Contract.LocalSymbol = readString(buf)
	m.Contract.TradingClass = readString(buf)
}

type AccountUpdateTimeMsg struct {
	MsgHeadWithoutRequestId
	TimeStampStr string
}

func (m *AccountUpdateTimeMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
	m.TimeStampStr = readString(buf)
}

type ContractDetailMsg struct {
	MsgHeadWithoutVersion
	Detail twsmodel.ContractDetail
}

func (m *ContractDetailMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutVersion.deserialize(buf)
	m.Detail.Contract.Symbol = readString(buf)
	m.Detail.Contract.SecType = readString(buf)
	m.Detail.Contract.LastTradeDateOrContractMonth = readString(buf)
	m.Detail.Contract.Strike = readFloat64(buf)
	m.Detail.Contract.Right = readString(buf)
	m.Detail.Contract.Exchange = readString(buf)
	m.Detail.Contract.Currency = readString(buf)
	m.Detail.Contract.LocalSymbol = readString(buf)
	m.Detail.MarketName = readString(buf)
	m.Detail.Contract.TradingClass = readString(buf)
	m.Detail.Contract.ConId = readInt(buf)
	m.Detail.MinTick = readDecimal(buf)
	m.Detail.Contract.Multiplier = readString(buf)
	m.Detail.OrderTypes = readString(buf)
	m.Detail.ValidExchanges = readString(buf)
	m.Detail.PriceMagnifier = readInt(buf)
	m.Detail.UnderConId = readInt(buf)
	m.Detail.LongName = readStringUnquote(buf)
	m.Detail.Contract.PrimaryExch = readString(buf)
	m.Detail.ContractMonth = readString(buf)
	m.Detail.Industry = readString(buf)
	m.Detail.Category = readString(buf)
	m.Detail.Subcategory = readString(buf)
	m.Detail.TimeZoneId = readString(buf)
	m.Detail.TradingHours = readString(buf)
	m.Detail.LiquidHours = readString(buf)
	m.Detail.EvRule = readString(buf)
	m.Detail.EvMultiplier = readFloat64(buf)
	c := readInt(buf)
	if c > 0 {
		for i := 0; i < c; i++ {
			tgv := twsmodel.TagValue{}
			tgv.Tag = readString(buf)
			tgv.Value = readString(buf)
			m.Detail.SecIdList = append(m.Detail.SecIdList, tgv)
		}
	}
	m.Detail.AggGroup = readInt(buf)
	m.Detail.UnderSymbol = readString(buf)
	m.Detail.UnderSecType = readString(buf)
	m.Detail.MarketRuleIds = readString(buf)
	m.Detail.RealExpirationDate = readString(buf)
	m.Detail.StockType = readString(buf)
	m.Detail.MinSize = readDecimal(buf)
	m.Detail.SizeIncrement = readDecimal(buf)
	m.Detail.SuggestedSizeIncrement = readDecimal(buf)
}

type ContractDetailEndMsg struct {
	MsgHead
}

type TickPriceMsg struct {
	MsgHead
	TickType    twsmodel.TickType
	TickTypeStr string
	Price       decimal.Decimal
	Size        decimal.Decimal
	Attr        twsmodel.TickAttribute
}

func (m *TickPriceMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.TickType = twsmodel.TickType(readInt(buf))
	m.Price = readDecimal(buf)
	m.Size = readDecimal(buf)
	bitmask := twsmodel.NewBitMask(readInt(buf))
	m.Attr.CanAutoExecute = bitmask.Get(0)
	m.Attr.PastLimit = bitmask.Get(1)
	m.Attr.PreOpen = bitmask.Get(2)

	m.TickTypeStr = twsmodel.TickType2Str(m.TickType)
}

type TickSizeMsg struct {
	MsgHead
	TickType    twsmodel.TickType
	TickTypeStr string
	Size        decimal.Decimal
}

func (m *TickSizeMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.TickType = twsmodel.TickType(readInt(buf))
	m.Size = readDecimal(buf)

	m.TickTypeStr = twsmodel.TickType2Str(m.TickType)
}

type TickStringMsg struct {
	MsgHead
	TickType    twsmodel.TickType
	TickTypeStr string
	Value       string
}

func (m *TickStringMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.TickType = twsmodel.TickType(readInt(buf))
	m.Value = readString(buf)

	m.TickTypeStr = twsmodel.TickType2Str(m.TickType)
}

type TickGenericMsg struct {
	MsgHead
	TickType    twsmodel.TickType
	TickTypeStr string
	Value       float64
}

func (m *TickGenericMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.TickType = twsmodel.TickType(readInt(buf))
	m.Value = readFloat64(buf)

	m.TickTypeStr = twsmodel.TickType2Str(m.TickType)
}

type MarketDataTypeMsg struct {
	MsgHead
	MarketDataType twsmodel.MarketDataType
}

func (m *MarketDataTypeMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHead.deserialize(buf)
	m.MarketDataType = twsmodel.MarketDataType(readInt(buf))
}

type TickReqParamsMsg struct {
	MsgHeadWithoutVersion
	MinTick             decimal.Decimal
	BboExchange         string
	SnapshotPermissions int
}

func (m *TickReqParamsMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutVersion.deserialize(buf)
	m.MinTick = readDecimal(buf)
	m.BboExchange = readString(buf)
	m.SnapshotPermissions = readInt(buf)
}

type TickSnapshotEndMsg struct {
	MsgHead
}

type TickByTickMsg struct {
	MsgHeadWithoutVersion
	Type      twsmodel.TickByTickType
	TypeStr   string
	TimeStamp int
	Time      time.Time

	Last struct {
		Price      decimal.Decimal
		Size       decimal.Decimal
		PassLimit  bool
		Unreported bool
	}

	BidAsk struct {
		BidPrice    decimal.Decimal
		AskPrice    decimal.Decimal
		BidSize     decimal.Decimal
		AskSize     decimal.Decimal
		BidPassLow  bool
		AskPassHigh bool
	}

	MidPoint struct {
		MidPoint decimal.Decimal
	}
}

func (t *TickByTickMsg) deserialize(buf *bytes.Buffer) {
	t.MsgHeadWithoutVersion.deserialize(buf)
	t.Type = twsmodel.TickByTickType(readInt(buf))
	t.TimeStamp = readInt(buf)

	switch t.Type {
	case twsmodel.TickByTickType_Last:
		fallthrough
	case twsmodel.TickByTickType_AllLast:
		t.Last.Price = readDecimal(buf)
		t.Last.Size = readDecimal(buf)
		bm := readBitMask(buf)
		t.Last.PassLimit = bm.Get(0)
		t.Last.Unreported = bm.Get(1)
	case twsmodel.TickByTickType_BidAsk:
		t.BidAsk.BidPrice = readDecimal(buf)
		t.BidAsk.AskPrice = readDecimal(buf)
		t.BidAsk.BidSize = readDecimal(buf)
		t.BidAsk.AskSize = readDecimal(buf)
		bm := readBitMask(buf)
		t.BidAsk.BidPassLow = bm.Get(0)
		t.BidAsk.AskPassHigh = bm.Get(1)
	case twsmodel.TickByTickType_MidPoint:
		t.MidPoint.MidPoint = readDecimal(buf)
	}

	t.TypeStr = twsmodel.TickByTickType2String(t.Type)
	t.Time = time.Unix(int64(t.TimeStamp), 0)
}

type MarketRuleMsg struct {
	Id              int
	PriceIncrements []twsmodel.PriceIncrement
}

func (m *MarketRuleMsg) deserialize(buf *bytes.Buffer) {
	m.Id = readInt(buf)
	count := readInt(buf)
	for i := 0; i < count; i++ {
		pi := twsmodel.PriceIncrement{LowEdge: readDecimal(buf), Increment: readDecimal(buf)}
		m.PriceIncrements = append(m.PriceIncrements, pi)
	}
}

type HistoricalDataBar struct {
	TimeStr  string
	Time     time.Time
	Open     decimal.Decimal
	High     decimal.Decimal
	Low      decimal.Decimal
	Close    decimal.Decimal
	Volume   decimal.Decimal
	WAP      decimal.Decimal
	BarCount int
}

func (m *HistoricalDataBar) deserialize(buf *bytes.Buffer) {
	m.TimeStr = readString(buf)
	m.Open = readDecimal(buf)
	m.High = readDecimal(buf)
	m.Low = readDecimal(buf)
	m.Close = readDecimal(buf)
	m.Volume = readDecimal(buf)
	m.WAP = readDecimal(buf)
	m.BarCount = readInt(buf)

	m.Time = parseDateTimeFormatA(m.TimeStr)
}

type HistoricalDataMsg struct {
	MsgHeadWithoutVersion
	StartTimeStr string
	EndTimeStr   string
	StartTime    time.Time
	EndTime      time.Time
	Bars         []HistoricalDataBar
}

func (m *HistoricalDataMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutVersion.deserialize(buf)
	m.StartTimeStr = readString(buf)
	m.EndTimeStr = readString(buf)
	count := readInt(buf)
	for i := 0; i < count; i++ {
		bar := HistoricalDataBar{}
		bar.deserialize(buf)
		m.Bars = append(m.Bars, bar)
	}

	m.StartTime = parseDateTimeFormatA(m.StartTimeStr)
	m.EndTime = parseDateTimeFormatA(m.EndTimeStr)
}

type OrderStatusMsg struct {
	OrderId       int             // 用户自定义的订单ID
	Status        string          // PendingSubmit/PendingCancel/PreSubmitted/Submitted/ApiCancelled/Cancelled/Filled/Inactive
	Filled        decimal.Decimal // 已成交
	Remaining     decimal.Decimal // 未成交
	AvgFillPrice  decimal.Decimal // 成交均价
	PermId        int             // 由TWS生成的永久订单ID，用于追踪订单。暂时用不上
	ParentId      int             // 用不上
	LastFillPrice decimal.Decimal // 最近一次成交价格
	ClientId      int             // API客户端的连接ID
	WhyHeld       string          // 用不上
	MktCapPrice   decimal.Decimal // 用不上
}

func (o *OrderStatusMsg) String() string {
	return fmt.Sprintf("[permId:%d orderId:%d, filled:%v, remaining:%v, avgPrice:%v, status:%s]",
		o.PermId,
		o.OrderId,
		o.Filled,
		o.Remaining,
		o.AvgFillPrice,
		o.Status)
}

func (m *OrderStatusMsg) deserialize(buf *bytes.Buffer) {
	m.OrderId = readInt(buf)
	m.Status = readString(buf)
	m.Filled = readDecimal(buf)
	m.Remaining = readDecimal(buf)
	m.AvgFillPrice = readDecimal(buf)
	m.PermId = readInt(buf)
	m.ParentId = readInt(buf)
	m.LastFillPrice = readDecimal(buf)
	m.ClientId = readInt(buf)
	m.LastFillPrice = readDecimal(buf)
	m.ClientId = readInt(buf)
	m.WhyHeld = readString(buf)
	m.MktCapPrice = readDecimal(buf)
}

type OpenOrdersMsg struct {
	Contract   twsmodel.Contract
	Order      twsmodel.Order
	OrderState twsmodel.OrderState
}

func (m *OpenOrdersMsg) deserialize(buf *bytes.Buffer) {
	m.Order.OrderId = readInt(buf)

	// contract
	m.Contract.ConId = readInt(buf)
	m.Contract.Symbol = readString(buf)
	m.Contract.SecType = readString(buf)
	m.Contract.LastTradeDateOrContractMonth = readString(buf)
	m.Contract.Strike = readFloat64(buf)
	m.Contract.Right = readString(buf)
	m.Contract.Multiplier = readString(buf)
	m.Contract.Exchange = readString(buf)
	m.Contract.Currency = readString(buf)
	m.Contract.LocalSymbol = readString(buf)
	m.Contract.TradingClass = readString(buf)

	// order
	m.Order.Action = readString(buf)
	m.Order.TotalQuantity = readDecimal(buf)
	m.Order.OrderType = readString(buf)
	m.Order.LmtPrice = readDecimalMax(buf)
	m.Order.AuxPrice = readFloat64Max(buf)
	m.Order.Tif = readString(buf)
	m.Order.OcaGroup = readString(buf)
	m.Order.Account = readString(buf)
	m.Order.OpenClose = readString(buf)
	m.Order.Origin = readInt(buf)
	m.Order.OrderRef = readString(buf)
	m.Order.ClientId = readInt(buf)
	m.Order.PermId = readInt(buf)
	m.Order.OutsideRth = readBool(buf)
	m.Order.Hidden = readBool(buf)
	m.Order.DiscretionaryAmt = readFloat64(buf)
	m.Order.GoodAfterTime = readString(buf)
	readString(buf) // skip deprecated sharesAllocation field
	m.Order.FaGroup = readString(buf)
	m.Order.FaMethod = readString(buf)
	m.Order.FaPercentage = readString(buf)
	m.Order.FaProfile = readString(buf)
	m.Order.ModelCode = readString(buf)
	m.Order.GoodTillDate = readString(buf)
	m.Order.Rule80A = readString(buf)
	m.Order.PercentOffset = readFloat64Max(buf)
	m.Order.SettlingFirm = readString(buf)
	m.Order.ShortSaleSlot = readInt(buf)
	m.Order.DesignatedLocation = readString(buf)
	m.Order.ExemptCode = readInt(buf)
	m.Order.AuctionStrategy = readInt(buf)
	m.Order.StartingPrice = readFloat64Max(buf)
	m.Order.StockRefPrice = readFloat64Max(buf)
	m.Order.Delta = readFloat64Max(buf)
	m.Order.StockRangeLower = readFloat64Max(buf)
	m.Order.StockRangeUpper = readFloat64Max(buf)
	m.Order.DisplaySize = readIntMax(buf)
	m.Order.BlockOrder = readBool(buf)
	m.Order.SweepToFill = readBool(buf)
	m.Order.AllOrNone = readBool(buf)
	m.Order.MinQty = readIntMax(buf)
	m.Order.OcaType = readInt(buf)
	readBool(buf)
	readBool(buf)
	readFloat64Max(buf)
	m.Order.ParentId = readInt(buf)
	m.Order.TriggerMethod = readInt(buf)
	m.Order.Volatility = readFloat64Max(buf)
	m.Order.VolatilityType = readInt(buf)
	m.Order.DeltaNeutralOrderType = readString(buf)
	m.Order.DeltaNeutralAuxPrice = readFloat64Max(buf)
	if m.Order.DeltaNeutralOrderType != "" {
		m.Order.DeltaNeutralConId = readInt(buf)
		m.Order.DeltaNeutralSettlingFirm = readString(buf)
		m.Order.DeltaNeutralClearingAccount = readString(buf)
		m.Order.DeltaNeutralClearingIntent = readString(buf)
		m.Order.DeltaNeutralOpenClose = readString(buf)
		m.Order.DeltaNeutralShortSale = readBool(buf)
		m.Order.DeltaNeutralShortSaleSlot = readInt(buf)
		m.Order.DeltaNeutralDesignatedLocation = readString(buf)
		m.Order.ContinuousUpdate = readInt(buf)
		m.Order.ReferencePriceType = readInt(buf)
		m.Order.TrailStopPrice = readFloat64Max(buf)
		m.Order.TrailingPercent = readFloat64Max(buf)
		m.Order.BasisPoints = readFloat64Max(buf)
		m.Order.BasisPointsType = readIntMax(buf)
	}
	m.Contract.ComboLegsDescription = readString(buf)
	readInt(buf) // 跳过comboLegsCount
	readInt(buf) // 跳过orderComboLegsCount
	smartComboRoutingParamsCount := readInt(buf)
	for i := 0; i < smartComboRoutingParamsCount; i++ {
		tgv := twsmodel.TagValue{}
		tgv.Tag = readString(buf)
		tgv.Value = readString(buf)
		m.Order.SmartComboRoutingParams = append(m.Order.SmartComboRoutingParams, tgv)
	}
	m.Order.ScaleInitLevelSize = readIntMax(buf)
	m.Order.ScaleSubsLevelSize = readIntMax(buf)
	m.Order.ScalePriceIncrement = readFloat64Max(buf)
	if m.Order.ScalePriceIncrement > 0 && m.Order.ScalePriceIncrement != math.MaxFloat64 {
		m.Order.ScalePriceAdjustValue = readFloat64Max(buf)
		m.Order.ScalePriceAdjustInterval = readIntMax(buf)
		m.Order.ScaleProfitOffset = readFloat64Max(buf)
		m.Order.ScaleAutoReset = readBool(buf)
		m.Order.ScaleInitPosition = readIntMax(buf)
		m.Order.ScaleInitFillQty = readIntMax(buf)
		m.Order.ScaleRandomPercent = readBool(buf)
	}
	m.Order.HedgeType = readString(buf)
	if m.Order.HedgeType != "" {
		m.Order.HedgeParam = readString(buf)
	}
	m.Order.OptOutSmartRouting = readBool(buf)
	m.Order.ClearingAccount = readString(buf)
	m.Order.ClearingIntent = readString(buf)
	m.Order.NotHeld = readBool(buf)
	readBool(buf) // 跳过contract.DeltaNeutralContract
	m.Order.AlgoStrategy = readString(buf)
	if m.Order.AlgoStrategy != "" {
		algoParamsCount := readInt(buf)
		for i := 0; i < algoParamsCount; i++ {
			tgv := twsmodel.TagValue{}
			tgv.Tag = readString(buf)
			tgv.Value = readString(buf)
			m.Order.AlgoParams = append(m.Order.AlgoParams, tgv)
		}
	}
	m.Order.Solicited = readBool(buf)
	m.Order.WhatIf = readBool(buf)

	// order state
	m.OrderState.Status = readString(buf)
	m.OrderState.InitMarginBefore = readString(buf)
	m.OrderState.MaintMarginBefore = readString(buf)
	m.OrderState.EquityWithLoanBefore = readString(buf)
	m.OrderState.InitMarginChange = readString(buf)
	m.OrderState.MaintMarginChange = readString(buf)
	m.OrderState.EquityWithLoanChange = readString(buf)
	m.OrderState.InitMarginAfter = readString(buf)
	m.OrderState.MaintMarginAfter = readString(buf)
	m.OrderState.EquityWithLoanAfter = readString(buf)
	m.OrderState.Commission = readFloat64Max(buf)
	m.OrderState.MinCommission = readFloat64Max(buf)
	m.OrderState.MaxCommission = readFloat64Max(buf)
	m.OrderState.CommissionCurrency = readString(buf)
	m.OrderState.WarningText = readString(buf)

	// order again
	m.Order.RandomizeSize = readBool(buf)
	m.Order.RandomizePrice = readBool(buf)
	if m.Order.OrderType == "PEG BENCH" {
		m.Order.ReferenceContractId = readInt(buf)
		m.Order.IsPeggedChangeAmountDecrease = readBool(buf)
		m.Order.PeggedChangeAmount = readFloat64Max(buf)
		m.Order.ReferenceChangeAmount = readFloat64Max(buf)
		m.Order.ReferenceExchange = readString(buf)
	}

	readInt(buf) // 跳过order.Conditions
	m.Order.AdjustedOrderType = readString(buf)
	m.Order.TriggerPrice = readFloat64Max(buf)
	m.Order.TrailStopPrice = readFloat64Max(buf)
	m.Order.LmtPriceOffset = readFloat64Max(buf)
	m.Order.AdjustedStopPrice = readFloat64Max(buf)
	m.Order.AdjustedStopLimitPrice = readFloat64Max(buf)
	m.Order.AdjustedTrailingAmount = readFloat64Max(buf)
	m.Order.AdjustableTrailingUnit = readInt(buf)
	readString(buf) // 跳过SoftDollarTier
	readString(buf)
	readString(buf)
	m.Order.CashQty = readFloat64Max(buf)
	m.Order.DontUseAutoPriceForHedge = readBool(buf)
	m.Order.IsOmsContainer = readBool(buf)
	m.Order.DiscretionaryUpToLimitPrice = readBool(buf)
	readBool(buf) // 跳过UsePriceMgmtAlgo
	m.Order.Duration = readIntMax(buf)
	m.Order.PostToAts = readIntMax(buf)
	m.Order.AutoCancelParent = readBool(buf)
	m.Order.MinTradeQty = readIntMax(buf)
	m.Order.MinCompeteSize = readIntMax(buf)
	m.Order.CompeteAgainstBestOffset = readFloat64Max(buf)
	m.Order.MidOffsetAtWhole = readFloat64Max(buf)
	m.Order.MidOffsetAtHalf = readFloat64Max(buf)
}

type OpenOrderEndMsg struct {
	MsgHeadWithoutRequestId
}

func (m *OpenOrderEndMsg) deserialize(buf *bytes.Buffer) {
	m.MsgHeadWithoutRequestId.deserialize(buf)
}
