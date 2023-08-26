/*
 * @Author: aztec
 * @Date: 2023-02-23 09:33:59
 * @Description: 定义一个通用的订单，放一些最基础、所有订单都有的东西
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package common

import (
	"fmt"
	"strconv"
	"time"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

type OrderImpl struct {
	LogPrefix     string
	InstrumentMgr *InstrumentMgr  // 用于对齐价格、数量
	Trader        CommonTrader    // 所属Traqder
	InstId        string          // 现货交易对、合约Id等类型标识
	OrderId       int64           // 订单Id
	CltOrderId    string          // 自定义订单Id
	Price         decimal.Decimal // 订单价格
	Size          decimal.Decimal // 订单数量
	Dir           OrderDir        // 订单方向
	ReduceOnly    bool            // 只减仓(仅合约有效)
	MakeOnly      bool            // 只挂单
	Purpose       string          // 订单目的（调试用）
	Filled        decimal.Decimal // 已成交数量
	AvgPrice      decimal.Decimal // 平均成交价格
	Borntime      time.Time       // 本地创建时间
	UpdateTime    time.Time       // 最近刷新时间
	Status        string          // 根据不同交易所的规则，一般有不同的定义
	Finished      bool            // 是否完结
	ErrMsg        string          // 最近的错误消息（仅用于记录，不用于判断订单是否失败）
	FatalError    bool            // 是否出现致命错误

	// 成交回调
	Observers []OrderObserver
}

// 初始化订单，矫正价格、数量
func (o *OrderImpl) Init(
	trader CommonTrader,
	instrumentMgr *InstrumentMgr,
	instId string,
	price, amount decimal.Decimal,
	dir OrderDir,
	makeOnly, reduceOnly bool,
	purpose string) bool {
	o.Trader = trader
	o.InstrumentMgr = instrumentMgr
	o.InstId = instId
	o.Dir = dir
	o.ReduceOnly = reduceOnly
	o.MakeOnly = makeOnly
	o.Purpose = purpose

	o.Price = instrumentMgr.AlignPrice(
		instId,
		price,
		dir,
		makeOnly,
		trader.Market().OrderBook().Buy1(),
		trader.Market().OrderBook().Sell1())
	o.Size = amount
	max := o.Trader.AvilableAmount(o.Dir, o.Price)
	o.Size = util.ClampDecimal(o.Size, decimal.Zero, max) // 受AvilableAmount的制约
	o.Size = instrumentMgr.AlignSize(instId, o.Size)      // 对齐
	minSize := instrumentMgr.MinSize(instId, price)

	if o.Size.GreaterThanOrEqual(minSize) {
		o.CltOrderId = NewClientOrderId(o.Purpose)
		o.LogPrefix = fmt.Sprintf("Order-%s-%s", o.InstId, o.CltOrderId)
		o.Status = "born"
		o.Borntime = time.Now()
		o.Observers = make([]OrderObserver, 0)
		return true
	} else {
		logger.LogInfo(o.LogPrefix, "creating order failed, size too small(instId=%s, raw amount=%v, aligned size=%v, minSize=%v)",
			instId,
			amount,
			o.Size,
			minSize)
		return false
	}
}

// #region 实现common.Order
func (o *OrderImpl) AddObserver(obs OrderObserver) {
	o.Observers = append(o.Observers, obs)
}

func (o *OrderImpl) GetID() (string, string) {
	return strconv.FormatInt(o.OrderId, 10), o.CltOrderId
}

func (o *OrderImpl) GetType() string {
	return o.InstId
}

func (o *OrderImpl) String() string {
	if o == nil {
		return "nil"
	}

	return fmt.Sprintf(
		"[instID:%v purpose:%v status:%v id:%v cltid:%v price:%v amount:%v dir:%v reduceOnly:%v makeOnly:%v filled:%v avgPrice:%v msg:%v]",
		o.InstId,
		o.Purpose,
		o.Status,
		o.OrderId,
		o.CltOrderId,
		o.Price,
		o.Size,
		OrderDir2Str(o.Dir),
		o.ReduceOnly,
		o.MakeOnly,
		o.Filled,
		o.AvgPrice,
		o.ErrMsg)
}

func (o *OrderImpl) GetStatus() string {
	return o.Status
}

func (o *OrderImpl) GetExtend() string {
	return "" // 子类覆盖
}

func (o *OrderImpl) GetDir() OrderDir {
	return o.Dir
}

func (o *OrderImpl) GetPrice() decimal.Decimal {
	return o.Price
}

func (o *OrderImpl) GetSize() decimal.Decimal {
	return o.Size
}

func (o *OrderImpl) GetFilled() decimal.Decimal {
	return o.Filled
}

func (o *OrderImpl) GetUnfilled() decimal.Decimal {
	return o.Size.Sub(o.Filled)
}

func (o *OrderImpl) GetAvgPrice() decimal.Decimal {
	return o.AvgPrice
}

func (o *OrderImpl) GetBornTime() time.Time {
	return o.Borntime
}

func (o *OrderImpl) GetUpdateTime() time.Time {
	return o.UpdateTime
}

func (o *OrderImpl) IsAlive() bool {
	return o.OrderId > 0 && !o.IsFinished()
}

func (o *OrderImpl) IsFinished() bool {
	return o.Finished || o.FatalError
}

func (o *OrderImpl) HasFatalError() bool {
	return o.FatalError
}

// #endregion
