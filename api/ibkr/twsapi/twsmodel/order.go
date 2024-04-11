/*
- @Author: aztec
- @Date: 2024-03-05 11:38:46
- @Description: 订单对象
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import (
	"math"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type Order struct {
	OrderId       int             // 订单ID，由用户指定，注意要遵守NextValidId的限制
	Solicited     bool            // 经纪商征求客户意见，用不到
	ClientId      int             // 连接tws的客户端api的id
	PermId        int             // 由TWS分配的永久ID，用于追踪订单
	Action        string          // BUY/SELL
	TotalQuantity decimal.Decimal // 下单数量
	OrderType     string          // 订单类型 MKT/LMT/STP/TRAIL：市价单、限价单、止损单、移动止损单
	LmtPrice      decimal.Decimal // 限价。仅在LMT等订单中使用。其他种类订单应该填0
	AuxPrice      float64         // 通用字段，用来设置止损价格、移动止损数量(?)等。目前用不上。

	// 生效时间Time in force。常用的有：
	// DAY：当日生效（类似的还有Minutes等）
	// GTC：Good until canceled
	// IOC：Immediate or Cancel
	// FOK：Fill-or-Kill
	// 更详细的介绍要看API里的注释和文档
	Tif string

	// 指示TWS是否要把订单传给交易所。似乎总是应该传true
	Transmit bool

	// #region 以下字段不常用，直接从C# API代码中复制过来
	OcaGroup                       string          // One-Cancels-All group identifier.
	OcaType                        int             // Tells how to handle remaining orders in an OCA group when one order or part of an order executes.
	OrderRef                       string          // The order reference. Intended for institutional customers only, although all customers may use it to identify the API client that sent the order when multiple API clients are running.
	ParentId                       int             // The order ID of the parent order, used for bracket and auto trailing stop orders.
	BlockOrder                     bool            // Specifies if the order is an ISE Block order.
	SweepToFill                    bool            // Specifies if the order is a Sweep-to-Fill order.
	DisplaySize                    int             // The publicly disclosed order size, used when placing Iceberg orders.
	TriggerMethod                  int             // Specifies how simulated stop, stop, and stop limit orders are triggered.
	OutsideRth                     bool            // Specifies if the order can be triggered or filled outside of regular trading hours.
	Hidden                         bool            // Specifies if the order is not visible when viewing market depth. This option is only valid for orders routed to the NASDAQ exchange.
	GoodAfterTime                  string          // Specifies the date and time the order becomes active.
	GoodTillDate                   string          // Specifies the date and time the order is valid until.
	OverridePercentageConstraints  bool            // Overrides TWS constraints.
	Rule80A                        string          // The order's Rule 80A.
	AllOrNone                      bool            // Indicates if the order must be filled in its entirety or not at all.
	MinQty                         int             // Identifies a minimum quantity order type.
	PercentOffset                  float64         // The percentage offset amount for relative orders.
	TrailStopPrice                 float64         // The trailing stop price for TRAIL LIMIT orders.
	TrailingPercent                float64         // The trailing amount, as a percentage, for trailing stop orders.
	FaGroup                        string          // The financial advisor group the trade will be allocated to.
	FaProfile                      string          // The financial advisor allocation profile the trade will be allocated to.
	FaMethod                       string          // The financial advisor allocation method the trade will be allocated to.
	FaPercentage                   string          // The percentage of the financial advisor's allocation.
	OpenClose                      string          // Available for institutional clients to determine if this order is to open or close a position
	Origin                         int             // Specifies the order's origin. Same as TWS "Origin" column.
	ShortSaleSlot                  int             // For institutional customers only. Specifies the location of the short sale slot.
	DesignatedLocation             string          // For institutional customers only. Indicates the location where the short sale was initiated.
	ExemptCode                     int             // For IB Execution-Only accounts with applicable securities. Marks the order as exempt from the Short Sale Up-Tick Rule.
	DiscretionaryAmt               float64         // The amount off the limit price for discretionary orders.
	OptOutSmartRouting             bool            // Specifies if SmartRouting should be used.
	AuctionStrategy                int             // For BOX orders only. Specifies the auction strategy.
	StartingPrice                  float64         // The starting price for the auction. For BOX orders only.
	StockRefPrice                  float64         // The stock's reference price.
	Delta                          float64         // The stock's Delta. For BOX orders only.
	StockRangeLower                float64         // The lower value for the acceptable underlying stock price range.
	StockRangeUpper                float64         // The upper value for the acceptable underlying stock price range.
	Volatility                     float64         // The volatility for the option price calculation according to TWS Option Analytics.
	VolatilityType                 int             // The volatility type.
	ContinuousUpdate               int             // Specifies if TWS will automatically update the order's limit price based on the change in the underlying's price.
	ReferencePriceType             int             // Specifies the reference price type to use for options' limit price calculation and stock price range monitoring.
	DeltaNeutralOrderType          string          // Specifies the order type to use for fully or partially executed Delta Neutral orders.
	DeltaNeutralAuxPrice           float64         // Specifies the aux price for the order type.
	DeltaNeutralConId              int             // Specifies the unique contract identifier of the underlying security for Delta Neutral orders.
	DeltaNeutralSettlingFirm       string          // Indicates the firm which will settle the Delta Neutral trade. Institutions only.
	DeltaNeutralClearingAccount    string          // Specifies the beneficiary of the Delta Neutral order.
	DeltaNeutralClearingIntent     string          // Specifies where the clients want their shares to be cleared at. Must be specified by execution-only clients. Valid values are: IB, Away, and PTA (post trade allocation).
	DeltaNeutralOpenClose          string          // Specifies whether the order is an Open or a Close order and is used when the hedge involves a CFD and the order is clearing away.
	DeltaNeutralShortSale          bool            // Used when the hedge involves a stock and indicates whether or not it is sold short.
	DeltaNeutralShortSaleSlot      int             // Indicates a short sale Delta Neutral order. Has a value of 1 (the clearing broker holds shares) or 2 (delivered from a third party). If you use 2, then you must specify a deltaNeutralDesignatedLocation.
	DeltaNeutralDesignatedLocation string          // Identifies third party order origin. Used only when deltaNeutralShortSaleSlot = 2.
	BasisPoints                    float64         // Specifies Basis Points for EFP order. The values increment in 0.01% = 1 basis point. For EFP orders only.
	BasisPointsType                int             // Specifies the increment of the Basis Points. For EFP orders only.
	ScaleInitLevelSize             int             // Defines the size of the first, or initial, order component. For Scale orders only.
	ScaleSubsLevelSize             int             // Defines the order size of the subsequent scale order components. For Scale orders only. Used in conjunction with scaleInitLevelSize().
	ScalePriceIncrement            float64         // Defines the price increment between scale components. For Scale orders only. This value is compulsory.
	ScalePriceAdjustValue          float64         // Modifies the value of the Scale order. For extended Scale orders.
	ScalePriceAdjustInterval       int             // Specifies the interval when the price is adjusted. For extended Scale orders.
	ScaleProfitOffset              float64         // Specifies the offset when to adjust profit. For extended scale orders.
	ScaleAutoReset                 bool            // Restarts the Scale series if the order is cancelled. For extended scale orders.
	ScaleInitPosition              int             // The initial position of the Scale order. For extended scale orders.
	ScaleInitFillQty               int             // Specifies the initial quantity to be filled. For extended scale orders.
	ScaleRandomPercent             bool            // Defines the random percent by which to adjust the position. For extended scale orders.
	HedgeType                      string          // For hedge orders. Possible values include: D - Delta, B - Beta, F - FX, P - Pair
	HedgeParam                     string          // For hedge orders. Beta = x for Beta hedge orders, ratio = y for Pair hedge order
	Account                        string          // The account the trade will be allocated to.
	SettlingFirm                   string          // Indicates the firm which will settle the trade. Institutions only.
	ClearingAccount                string          // Specifies the true beneficiary of the order. For IBExecution customers. This value is required for FUT/FOP orders for reporting to the exchange.
	ClearingIntent                 string          // For execution-only clients to know where do they want their shares to be cleared at. Valid values are: IB, Away, and PTA (post trade allocation).
	AlgoStrategy                   string          // The algorithm strategy. As of API version 9.6, the following algorithms are supported: ArrivalPx, DarkIce, PctVol, Twap, Vwap. For more information about IB's API algorithms, refer to https://www.interactivebrokers.com/en/software/api/apiguide/tables/ibalgo_parameters.htm
	AlgoParams                     []TagValue      // The list of parameters for the IB algorithm. For more information about IB's API algorithms, refer to https://www.interactivebrokers.com/en/software/api/apiguide/tables/ibalgo_parameters.htm
	WhatIf                         bool            // Allows to retrieve the commissions and margin information. When placing an order with this attribute set to true, the order will not be placed as such. Instead it will used to request the commissions and margin information that would result from this order.
	AlgoId                         string          // Identifies orders generated by algorithmic trading.
	NotHeld                        bool            // Orders routed to IBDARK are tagged as “post only” and are held in IB's order book, where incoming SmartRouted orders from other IB customers are eligible to trade against them. For IBDARK orders only.
	SmartComboRoutingParams        []TagValue      // Advanced parameters for Smart combo routing.
	OrderComboLegs                 []OrderComboLeg // List of Per-leg price following the same sequence combo legs are added.
	OrderMiscOptions               []TagValue      // For internal use only. Use the default value XYZ.
	ActiveStartTime                string          // Defines the start time of GTC orders.
	ActiveStopTime                 string          // Defines the stop time of GTC orders.
	ScaleTable                     string          // The list of scale orders. Used for scale orders.
	ModelCode                      string          // Is used to place an order to a model. For example, "Technology" model can be used for tech stocks first created in TWS.
	ExtOperator                    string          // This is a regulartory attribute that applies to all US Commodity (Futures) Exchanges, provided to allow client to comply with CFTC Tag 50 Rules.
	CashQty                        float64         // The native cash quantity.
	Mifid2DecisionMaker            string          // Identifies a person as the responsible party for investment decisions within the firm. Orders covered by MiFID 2 must include either Mifid2DecisionMaker or Mifid2DecisionAlgo field (but not both). Requires TWS 969+.
	Mifid2DecisionAlgo             string          // Identifies the algorithm responsible for investment decisions within the firm. Orders covered under MiFID 2 must include either Mifid2DecisionMaker or Mifid2DecisionAlgo, but cannot have both. Requires TWS 969+.
	Mifid2ExecutionTrader          string          // For MiFID 2 reporting; identifies a person as the responsible party for the execution of a transaction within the firm. Requires TWS 969+.
	Mifid2ExecutionAlgo            string          // For MiFID 2 reporting; identifies the algorithm responsible for the execution of a transaction within the firm. Requires TWS 969+.
	DontUseAutoPriceForHedge       bool            // Don't use auto price for hedge.
	AutoCancelDate                 string          // Specifies the date to auto cancel the order.
	FilledQuantity                 decimal.Decimal // Specifies the initial order quantity to be filled.
	RefFuturesConId                int             // Identifies the reference future conId.
	AutoCancelParent               bool            // Cancels the parent order if child order was cancelled.
	Shareholder                    string          // Identifies the Shareholder.
	ImbalanceOnly                  bool            // Used to specify "imbalance only open orders" or "imbalance only closing orders".
	RouteMarketableToBbo           bool            // Routes market order to Best Bid Offer.
	ParentPermId                   int64           // Parent order Id.
	AdvancedErrorOverride          string          // Accepts a list with parameters obtained from advancedOrderRejectJson.
	ManualOrderTime                string          // Used by brokers and advisors when manually entering, modifying or cancelling orders at the direction of a client. Only used when allocating orders to specific groups or accounts. Excluding "All" group.
	MinTradeQty                    int             // Defines the minimum trade quantity to fill. For IBKRATS orders.
	MinCompeteSize                 int             // Defines the minimum size to compete. For IBKRATS orders.
	CompeteAgainstBestOffset       float64         // Specifies the offset Off The Midpoint that will be applied to the order. For IBKRATS orders.
	MidOffsetAtWhole               float64         // This offset is applied when the spread is an even number of cents wide. This offset must be in whole-penny increments or zero. For IBKRATS orders.
	MidOffsetAtHalf                float64         // This offset is applied when the spread is an odd number of cents wide. This offset must be in half-penny increments. For IBKRATS orders.
	RandomizeSize                  bool            // Randomizes the order's size. Only for Volatility and Pegged to Volatility orders.
	RandomizePrice                 bool            // Randomizes the order's price. Only for Volatility and Pegged to Volatility orders.
	ReferenceContractId            int             // Pegged-to-benchmark orders: this attribute will contain the conId of the contract against which the order will be pegged.
	IsPeggedChangeAmountDecrease   bool            // Pegged-to-benchmark orders: indicates whether the order's pegged price should increase or decrease.
	PeggedChangeAmount             float64         // Pegged-to-benchmark orders: amount by which the order's pegged price should move.
	ReferenceChangeAmount          float64         // Pegged-to-benchmark orders: the amount the reference contract needs to move to adjust the pegged order.
	ReferenceExchange              string          // Pegged-to-benchmark orders: the exchange against which we want to observe the reference contract.
	AdjustedOrderType              string          // Adjusted Stop orders: the parent order will be adjusted to the given type when the adjusted trigger price is penetrated.
	TriggerPrice                   float64         // Adjusted Stop orders: specifies the trigger price to execute.
	LmtPriceOffset                 float64         // Adjusted Stop orders: specifies the price offset for the stop to move in increments.
	AdjustedStopPrice              float64         // Adjusted Stop orders: specifies the stop price of the adjusted (STP) parent.
	AdjustedStopLimitPrice         float64         // Adjusted Stop orders: specifies the stop limit price of the adjusted (STPL LMT) parent.
	AdjustedTrailingAmount         float64         // Adjusted Stop orders: specifies the trailing amount of the adjusted (TRAIL) parent.
	AdjustableTrailingUnit         int             // Adjusted Stop orders: specifies whether the trailing unit is an amount (set to 0) or a percentage (set to 1).
	ConditionsIgnoreRth            bool            // Indicates whether or not conditions will also be valid outside Regular Trading Hours.
	ConditionsCancelOrder          bool            // Conditions can determine if an order should become active or canceled.
	IsOmsContainer                 bool            // Set to true to create tickets from API orders when TWS is used as an OMS.
	DiscretionaryUpToLimitPrice    bool            // Set to true to convert order of type 'Primary Peg' to 'D-Peg'.
	Duration                       int             // Specifies the duration of the order. Format: yyyymmdd hh:mm:ss TZ. For GTD orders.
	PostToAts                      int             // Value must be positive, and it is the number of seconds that SMART order would be parked for at IBKRATS before being routed to exchange.

	// UsePriceMgmtAlgo               *bool           // Specifies whether to use Price Management Algo. CTCI users only.
	// Conditions                     []OrderCondition // Conditions determining when the order will be activated or canceled.
	// Tier                        SoftDollarTier // Define the Soft Dollar Tier used for the order. Only provided for registered professional advisors and hedge and mutual funds.

	// #endregion
}

func NewOrder() Order {
	return Order{
		LmtPrice:                       decimal.NewFromFloat(math.MaxFloat64),
		AuxPrice:                       math.MaxFloat64,
		ActiveStartTime:                "",
		ActiveStopTime:                 "",
		OutsideRth:                     false,
		OpenClose:                      "",
		Origin:                         0,
		Transmit:                       true,
		DesignatedLocation:             "",
		ExemptCode:                     -1,
		MinQty:                         math.MaxInt32,
		PercentOffset:                  math.MaxFloat64,
		OptOutSmartRouting:             false,
		StartingPrice:                  math.MaxFloat64,
		StockRefPrice:                  math.MaxFloat64,
		Delta:                          math.MaxFloat64,
		StockRangeLower:                math.MaxFloat64,
		StockRangeUpper:                math.MaxFloat64,
		Volatility:                     math.MaxFloat64,
		VolatilityType:                 math.MaxInt32,
		DeltaNeutralOrderType:          "",
		DeltaNeutralAuxPrice:           math.MaxFloat64,
		DeltaNeutralConId:              0,
		DeltaNeutralSettlingFirm:       "",
		DeltaNeutralClearingAccount:    "",
		DeltaNeutralClearingIntent:     "",
		DeltaNeutralOpenClose:          "",
		DeltaNeutralShortSale:          false,
		DeltaNeutralShortSaleSlot:      0,
		DeltaNeutralDesignatedLocation: "",
		ReferencePriceType:             math.MaxInt32,
		TrailStopPrice:                 math.MaxFloat64,
		TrailingPercent:                math.MaxFloat64,
		BasisPoints:                    math.MaxFloat64,
		BasisPointsType:                math.MaxInt32,
		ScaleInitLevelSize:             math.MaxInt32,
		ScaleSubsLevelSize:             math.MaxInt32,
		ScalePriceIncrement:            math.MaxFloat64,
		ScalePriceAdjustValue:          math.MaxFloat64,
		ScalePriceAdjustInterval:       math.MaxInt32,
		ScaleProfitOffset:              math.MaxFloat64,
		ScaleAutoReset:                 false,
		ScaleInitPosition:              math.MaxInt32,
		ScaleInitFillQty:               math.MaxInt32,
		ScaleRandomPercent:             false,
		ScaleTable:                     "",
		WhatIf:                         false,
		NotHeld:                        false,
		TriggerPrice:                   math.MaxFloat64,
		LmtPriceOffset:                 math.MaxFloat64,
		AdjustedStopPrice:              math.MaxFloat64,
		AdjustedStopLimitPrice:         math.MaxFloat64,
		AdjustedTrailingAmount:         math.MaxFloat64,
		ExtOperator:                    "",
		CashQty:                        math.MaxFloat64,
		Mifid2DecisionMaker:            "",
		Mifid2DecisionAlgo:             "",
		Mifid2ExecutionTrader:          "",
		Mifid2ExecutionAlgo:            "",
		DontUseAutoPriceForHedge:       false,
		AutoCancelDate:                 "",
		FilledQuantity:                 decimal.Zero,
		RefFuturesConId:                math.MaxInt32,
		AutoCancelParent:               false,
		Shareholder:                    "",
		ImbalanceOnly:                  false,
		RouteMarketableToBbo:           false,
		ParentPermId:                   math.MaxInt,
		Duration:                       math.MaxInt32,
		PostToAts:                      math.MaxInt32,
		AdvancedErrorOverride:          "",
		ManualOrderTime:                "",
		MinTradeQty:                    math.MaxInt32,
		MinCompeteSize:                 math.MaxInt32,
		CompeteAgainstBestOffset:       math.MaxFloat64,
		MidOffsetAtWhole:               math.MaxFloat64,
		MidOffsetAtHalf:                math.MaxFloat64,
	}
}

func (o *Order) ToParamArray() []interface{} {
	paramList := []interface{}{
		o.Action,
		o.TotalQuantity,
		o.OrderType,
		o.LmtPrice, //
		o.AuxPrice, // 这两处忽略了c#版api中的'AddParameterMax'功能，留意
		o.Tif,
		o.OcaGroup,
		o.Account,
		o.OpenClose,
		o.Origin,
		o.OrderRef,
		o.Transmit,
		o.ParentId,
		o.BlockOrder,
		o.SweepToFill,
		o.DisplaySize,
		o.TriggerMethod,
		o.OutsideRth,
		o.Hidden,
		nil, // 这里跳过了一大段当contract.SecType=="BAG"时的数据
		"",
		o.DiscretionaryAmt,
		o.GoodAfterTime,
		o.GoodTillDate,
		o.FaGroup,
		o.FaMethod,
		o.FaPercentage,
		o.FaProfile,
		o.ModelCode,
		o.ShortSaleSlot,
		o.DesignatedLocation,
		o.ExemptCode,
		o.OcaType,
		o.Rule80A,
		o.SettlingFirm,
		o.AllOrNone,
		o.MinQty,
		o.PercentOffset,
		false,
		false,
		math.MaxFloat64,
		o.AuctionStrategy,
		o.StartingPrice,
		o.StockRefPrice,
		o.Delta,
		o.StockRangeLower,
		o.StockRangeUpper,
		o.OverridePercentageConstraints,
		o.Volatility,
		o.VolatilityType,
		o.toDeltaNeutralParams(),
		o.ContinuousUpdate,
		o.ReferencePriceType,
		o.TrailStopPrice,
		o.TrailingPercent,
		o.ScaleInitLevelSize,
		o.ScaleSubsLevelSize,
		o.toScalePriceIncrementParams(),
		o.ScaleTable,
		o.ActiveStartTime,
		o.ActiveStopTime,
		o.HedgeType,
		util.ValueIf[interface{}](len(o.HedgeType) > 0, o.HedgeParam, nil),
		o.OptOutSmartRouting,
		o.ClearingAccount,
		o.ClearingIntent,
		o.NotHeld,
		false, // 这里跳过了contract.DeltaNeutralContract的处理
		o.toAlgoStrategyParams(),
		o.WhatIf,
		tagValueListToString(o.OrderMiscOptions),
		o.Solicited,
		o.RandomizeSize,
		o.RandomizePrice,
		nil, // 这里跳过了o.OrderType == "PEG BENCH"这一分支
		0,   // 这里跳过o.Conditions功能
		o.AdjustedOrderType,
		o.TriggerPrice,
		o.LmtPriceOffset,
		o.AdjustedStopPrice,
		o.AdjustedStopLimitPrice,
		o.AdjustedTrailingAmount,
		o.AdjustableTrailingUnit,
		o.ExtOperator,
		"", "", // 这里跳过了o.Tier
		o.CashQty,
		o.Mifid2DecisionMaker,
		o.Mifid2DecisionAlgo,
		o.Mifid2ExecutionTrader,
		o.Mifid2ExecutionAlgo,
		o.DontUseAutoPriceForHedge,
		o.IsOmsContainer,
		o.DiscretionaryUpToLimitPrice,
		byte(0), // 这里跳过UsePriceMgmtAlgo
		o.Duration,
		o.PostToAts,
		o.AutoCancelParent,
		o.AdvancedErrorOverride,
		o.ManualOrderTime,
		nil, // 这里跳过了最后一段关于几个特殊交易所和特殊订单类型的判断
	}

	return paramList
}

func (o *Order) toDeltaNeutralParams() []interface{} {
	paramList := []interface{}{
		o.DeltaNeutralOrderType,
		o.DeltaNeutralAuxPrice,
	}

	if o.DeltaNeutralOrderType != "" {
		paramList = append(paramList,
			o.DeltaNeutralConId,
			o.DeltaNeutralSettlingFirm,
			o.DeltaNeutralClearingAccount,
			o.DeltaNeutralClearingIntent,
			o.DeltaNeutralOpenClose,
			o.DeltaNeutralShortSale,
			o.DeltaNeutralShortSaleSlot,
			o.DeltaNeutralDesignatedLocation)
	}

	return paramList
}

func (o *Order) toScalePriceIncrementParams() []interface{} {
	paramList := []interface{}{o.ScalePriceIncrement}
	if o.ScalePriceIncrement != 0 && o.ScalePriceIncrement != math.MaxFloat64 {
		paramList = append(paramList,
			o.ScalePriceAdjustValue,
			o.ScalePriceAdjustInterval,
			o.ScaleProfitOffset,
			o.ScaleAutoReset,
			o.ScaleInitPosition,
			o.ScaleInitFillQty,
			o.ScaleRandomPercent)
	}

	return paramList
}

func (o *Order) toAlgoStrategyParams() []interface{} {
	paramList := []interface{}{o.AlgoStrategy}
	if o.AlgoStrategy != "" {
		c := len(o.AlgoParams)
		paramList = append(paramList, c)
		for i := 0; i < c; i++ {
			paramList = append(paramList, o.AlgoParams[i].Tag, o.AlgoParams[i].Value)
		}
	}
	paramList = append(paramList, o.AlgoId)
	return paramList
}
