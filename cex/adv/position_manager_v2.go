/*
- @Author: aztec
- @Date: 2024-05-03 15:51:09
- @Description: 另一个版本的交易器
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package adv

import (
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type PositionManagerV2 struct {
	trader       common.FutureTrader
	fnOnDeal     func(deal common.Deal)
	makerMode    bool
	mkDeal       *Maker
	tkDeal       *Taker
	valAmountUsd decimal.Decimal // 每张合约代表的usd价值。为了防止价格浮动导致频繁开平，仅在无仓位时更新此数值。带仓不更新。
	lastPos      decimal.Decimal // 用于监控仓位清空事件
}

func NewPositionManagerV2(t common.FutureTrader, makerMode bool, fnOnDeal func(deal common.Deal)) *PositionManagerV2 {
	d := &PositionManagerV2{}
	d.trader = t
	d.fnOnDeal = fnOnDeal
	d.makerMode = makerMode
	d.mkDeal = &Maker{}
	d.mkDeal.Init(t, true, false, true, 0, 0, "deal")
	d.mkDeal.SetDealFn(func(deal MakerOrderDeal) { d.onDeal(deal.Deal) })
	d.mkDeal.Go()
	return d
}

// 以U的数量进行交易
func (d *PositionManagerV2) UpdateTargetPosUsd(targetPosUsd decimal.Decimal, canOpenLong, canOpenShort bool) {
	realPos := d.trader.Position().Net()

	// 更新仓位价值
	if realPos.IsZero() || d.valAmountUsd.IsZero() {
		usdt := decimal.NewFromFloat(1000)
		pos := common.USDT2ContractAmountFloatUnRounded(usdt, d.trader.FutureMarket())
		d.valAmountUsd = usdt.Div(pos)
	}

	// 计算目标仓位
	targetPos := targetPosUsd.Div(d.valAmountUsd)

	if d.makerMode {
		d.dealMaker(realPos, targetPos, canOpenLong, canOpenShort)
	} else {
		d.dealTaker(realPos, targetPos, canOpenLong, canOpenShort)
	}
}

// 以币的数量进行交易
func (d *PositionManagerV2) UpdateTargetPos(targetPos decimal.Decimal, canOpenLong, canOpenShort bool) {
	realPos := d.trader.Position().Net()

	// targetPos转换成张数
	targetPos = targetPos.Div(d.trader.FutureMarket().ValueAmount())

	if d.makerMode {
		d.dealMaker(realPos, targetPos, canOpenLong, canOpenShort)
	} else {
		d.dealTaker(realPos, targetPos, canOpenLong, canOpenShort)
	}
}

func (d *PositionManagerV2) onDeal(deal common.Deal) {
	curPos := d.trader.Position().Net()
	if curPos.IsZero() && !d.lastPos.IsZero() {
		// 发生了PositionCleared事件
	}
	d.lastPos = curPos

	if d.fnOnDeal != nil {
		d.fnOnDeal(deal)
	}
}

func (d *PositionManagerV2) dealMaker(realPos, targetPos decimal.Decimal, canOpenLong, canOpenShort bool) {
	if targetPos.Sub(realPos).GreaterThanOrEqual(util.DecimalOne) {
		// 需要买入
		size := targetPos.Sub(realPos)
		if realPos.IsPositive() || realPos.IsZero() {
			if canOpenLong {
				// 开仓
				d.mkDeal.Modify(
					d.trader.Market().OrderBook().Buy1Price(),
					size,
					common.OrderDir_Buy,
					false,
				)
			} else {
				d.mkDeal.Cancel()
			}
		} else {
			// 平仓
			d.mkDeal.Modify(
				d.trader.Market().OrderBook().Buy1Price(),
				decimal.Min(size, realPos.Neg()),
				common.OrderDir_Buy,
				false,
			)
		}
	} else if realPos.Sub(targetPos).GreaterThanOrEqual(util.DecimalOne) {
		// 需要卖出
		size := realPos.Sub(targetPos)
		if realPos.IsNegative() || realPos.IsZero() {
			if canOpenShort {
				// 开仓
				d.mkDeal.Modify(
					d.trader.Market().OrderBook().Sell1Price(),
					size,
					common.OrderDir_Sell,
					false,
				)
			} else {
				d.mkDeal.Cancel()
			}
		} else {
			// 平仓
			d.mkDeal.Modify(
				d.trader.Market().OrderBook().Sell1Price(),
				decimal.Min(size, realPos),
				common.OrderDir_Sell,
				false,
			)
		}
	} else {
		// 无需交易
		d.mkDeal.Cancel()
	}
}

func (d *PositionManagerV2) dealTaker(realPos, targetPos decimal.Decimal, canOpenLong, canOpenShort bool) {
	if d.tkDeal != nil && d.tkDeal.Finished() {
		d.tkDeal = nil
	}

	if d.tkDeal != nil {
		return
	}

	d.tkDeal = &Taker{}
	d.tkDeal.Init(d.trader, decimal.Zero, common.OrderDir_Sell, true, "deal", nil)
	d.tkDeal.SetDealFn(func(tkDeal TakerDeal) { d.onDeal(tkDeal.Deal) })
	d.tkDeal.Go()
	d.tkDeal.Finished()

	if targetPos.Sub(realPos).GreaterThanOrEqual(util.DecimalOne) {
		// 需要买入
		size := targetPos.Sub(realPos)
		if realPos.IsPositive() || realPos.IsZero() {
			// 开仓
			if canOpenLong {
				d.tkDeal = &Taker{}
				d.tkDeal.Init(d.trader, size, common.OrderDir_Buy, false, "deal", nil)
			}
		} else {
			// 平仓
			d.tkDeal = &Taker{}
			d.tkDeal.Init(d.trader, decimal.Min(size, realPos.Neg()), common.OrderDir_Buy, true, "deal", nil)
		}
	} else if realPos.Sub(targetPos).GreaterThanOrEqual(util.DecimalOne) {
		// 需要卖出
		size := realPos.Sub(targetPos)
		if realPos.IsNegative() || realPos.IsZero() {
			if canOpenShort {
				// 开仓
				d.tkDeal = &Taker{}
				d.tkDeal.Init(d.trader, size, common.OrderDir_Sell, false, "deal", nil)
			}
		} else {
			// 平仓
			d.tkDeal = &Taker{}
			d.tkDeal.Init(d.trader, decimal.Min(size, realPos), common.OrderDir_Sell, true, "deal", nil)
		}
	}

	if d.tkDeal != nil {
		d.tkDeal.Go()
	}
}
