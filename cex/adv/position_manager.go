/*
 * @Author: aztec
 * @Date: 2022-11-15 11:31:47
 * @Description: 以仓位的角度，管理订单，进行开平仓，适用于合约。很多单市场策略都可以复用它
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package adv

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

type DealDir int

const (
	DealDir_None DealDir = iota
	DealDir_Open
	DealDir_Close
)

type PositionManagerStatus struct {
	InstId       string          `json:"instId"`
	TargetSz     decimal.Decimal `json:"targetSz"`
	TargetDir    string          `json:"targetDir"`
	Taker        bool            `json:"taker"`
	SuperTaker   bool            `json:"super_taker"`
	MaxSlipPoint decimal.Decimal `json:"max_slip"`
	Quiting      bool            `json:"quiting"`
}

type PositionManager struct {
	logPrefix string
	trader    common.FutureTrader

	targetSz  decimal.Decimal // 目标仓位数量
	targetDir common.OrderDir // 目标仓位方向
	stepSz    decimal.Decimal // 交易步长
	dealDir   DealDir         // 开仓/平仓

	// 开仓方式
	maker        bool            // 吃单/挂单开仓
	makeOnly     bool            // 挂单时仅挂单
	superTaker   bool            // 吃单时不计滑点
	maxSlipPoint decimal.Decimal // 吃单开仓的单次最大滑点
	maxBuyPrice  decimal.Decimal // 最高买入价（用户设定，非交易所限制）
	minSellPrice decimal.Decimal // 最低卖出价

	// 挂单
	mkOpen  *Maker
	mkClose *Maker

	// 吃单
	tkOpen  *Maker
	tkClose *Maker

	// 允许开仓
	canOpen bool

	// 交易自锁，达到仓位目标后锁定，即使仓位变化也不再交易
	dealDone bool

	// 使能
	enabled bool

	// 立即平仓
	quiting bool

	// 吃单冷却
	takerCD time.Time

	// 外部成交回调
	onDeal OnMakerOrderDeal
}

func (p *PositionManager) Init(trader common.FutureTrader, onDeal OnMakerOrderDeal, logPrefix string, maker, makeOnly bool) {
	p.trader = trader
	p.onDeal = onDeal
	if len(logPrefix) == 0 {
		p.logPrefix = fmt.Sprintf("posmgr-%s", trader.Market().Type())
	} else {
		p.logPrefix = logPrefix // 日志跟随外部
	}
	p.maker = maker
	p.makeOnly = makeOnly
	p.maxSlipPoint = decimal.NewFromFloat(0.005)
	p.mkOpen = new(Maker)
	p.mkClose = new(Maker)
	p.tkOpen = new(Maker)
	p.tkClose = new(Maker)
	p.mkOpen.Init(trader, p.makeOnly, false, true, 0, 0, "mkOpen")
	p.mkClose.Init(trader, p.makeOnly, false, true, 0, 0, "mkClose")
	p.tkOpen.Init(trader, false, false, true, 0, 0, "tkOpen")
	p.tkClose.Init(trader, false, false, true, 0, 0, "tkClose")
	p.mkOpen.SetDealFn(p.onOpenDeal)
	p.mkClose.SetDealFn(p.onCloseDeal)
	p.tkOpen.SetDealFn(p.onOpenDeal)
	p.tkClose.SetDealFn(p.onCloseDeal)
	p.canOpen = true
	p.enabled = true
	p.quiting = false
	p.dealDir = DealDir_None

	// 带仓位启动
	if p.long().IsPositive() {
		p.targetSz = p.long()
		p.targetDir = common.OrderDir_Buy
		logger.LogInfo(p.logPrefix, "start with long pos %v", p.long())
	} else if p.short().IsPositive() {
		p.targetSz = p.short()
		p.targetDir = common.OrderDir_Sell
		logger.LogInfo(p.logPrefix, "start with short pos %v", p.short())
	} else {
		logger.LogInfo(p.logPrefix, "start with no pos")
	}
}

func (p *PositionManager) Uninit() {
	p.mkOpen.Stop()
	p.mkClose.Stop()
	p.tkOpen.Stop()
	p.tkOpen.Stop()
}

func (p *PositionManager) Trader() common.FutureTrader {
	return p.trader
}

func (p *PositionManager) Status() PositionManagerStatus {
	s := PositionManagerStatus{}
	s.InstId = p.trader.Market().Type()
	s.TargetSz = p.targetSz
	s.TargetDir = common.OrderDir2Str(p.targetDir)
	s.Taker = !p.maker
	s.SuperTaker = p.superTaker
	s.MaxSlipPoint = p.maxSlipPoint
	s.Quiting = p.quiting
	return s
}

func (p *PositionManager) TargetSize() decimal.Decimal {
	return p.targetSz
}

func (p *PositionManager) TargetDir() common.OrderDir {
	return p.targetDir
}

func (p *PositionManager) ModifyTargetSize(targetSz decimal.Decimal, targetDir common.OrderDir) {
	// 确定好交易方向，避免反复开平仓
	if targetDir != p.targetDir {
		p.dealDir = DealDir_Open
	} else {
		if targetDir == common.OrderDir_Buy && targetSz.GreaterThan(p.long()) ||
			targetDir == common.OrderDir_Sell && targetSz.GreaterThan(p.short()) {
			p.dealDir = DealDir_Open
		} else {
			p.dealDir = DealDir_Close
		}
	}

	p.targetSz = p.trader.Market().AlignSize(targetSz)
	p.targetDir = targetDir
	p.dealDone = false
}

func (p *PositionManager) ModifyTargetSizeWithDealDir(targetSz decimal.Decimal, targetDir common.OrderDir, dealDir DealDir) {
	// 确定好交易方向，避免反复开平仓
	if targetDir != p.targetDir {
		p.dealDir = DealDir_Open
	} else {
		p.dealDir = dealDir
	}

	p.targetSz = p.trader.Market().AlignSize(targetSz)
	p.targetDir = targetDir
	p.dealDone = false
}

func (p *PositionManager) SetEnabled(enabled bool) {
	if p.enabled != enabled {
		p.enabled = enabled
		logger.LogInfo(p.logPrefix, "enabled set to %v", enabled)
	}
}

func (p *PositionManager) ModifyTargetSizeRatio(ratio decimal.Decimal) {
	if p.long().IsPositive() {
		p.ModifyTargetSize(p.long().Mul(ratio), common.OrderDir_Buy)
	} else if p.short().IsPositive() {
		p.ModifyTargetSize(p.short().Mul(ratio), common.OrderDir_Sell)
	}
}

func (p *PositionManager) SetCanOpen(canOpen bool) {
	p.canOpen = canOpen
}

func (p *PositionManager) SetTaker(taker, superTaker bool) {
	maker := !taker
	if p.maker != maker {
		p.maker = maker
		logger.LogInfo(p.logPrefix, "`maker` set to %v", p.maker)
	}

	if p.superTaker != superTaker {
		p.superTaker = superTaker
		logger.LogInfo(p.logPrefix, "`super taker` set to %v", p.superTaker)
	}
}

func (p *PositionManager) SetStepSize(stepSz decimal.Decimal) {
	if p.stepSz != stepSz {
		p.stepSz = stepSz
		logger.LogInfo(p.logPrefix, "`stepSz` set to %v", p.stepSz)
	}
}

func (p *PositionManager) SetMaxSlipPoint(slp decimal.Decimal) {
	if p.maxSlipPoint != slp {
		p.maxSlipPoint = slp
		logger.LogInfo(p.logPrefix, "`max slp` set to %v", p.maxSlipPoint)
	}
}

func (p *PositionManager) SetMaxBuyPrice(px decimal.Decimal) {
	p.maxBuyPrice = px
}

func (p *PositionManager) SetMinSellPrice(px decimal.Decimal) {
	p.maxBuyPrice = px
}

func (p *PositionManager) Quit() {
	p.quiting = true
	p.targetDir = common.OrderDir_None
	p.targetSz = decimal.Zero
	p.maker = true
	logger.LogInfo(p.logPrefix, "quit toggled to %v", p.quiting)
}

func (p *PositionManager) Cancel() {
	p.mkOpen.Cancel()
	p.mkClose.Cancel()
	p.tkOpen.Cancel()
	p.tkClose.Cancel()
	logger.LogInfo(p.logPrefix, "all order canceled")
}

func (p *PositionManager) Update() {
	if p.dealDone {
		return
	}

	if !p.enabled {
		return
	}

	dir := common.OrderDir_None
	sz := decimal.Zero
	isOpen := true
	if p.targetDir != common.OrderDir_Sell && p.short().IsPositive() {
		// 平掉空仓
		dir = common.OrderDir_Buy
		sz = p.short()
		isOpen = false
	} else if p.targetDir != common.OrderDir_Buy && p.long().IsPositive() {
		// 平掉多仓
		dir = common.OrderDir_Sell
		sz = p.long()
		isOpen = false
	} else if p.targetDir == common.OrderDir_Buy {
		if p.long().Sub(p.targetSz).GreaterThanOrEqual(util.DecimalOne) && p.dealDir == DealDir_Close {
			// 平掉相应的张数
			dir = common.OrderDir_Sell
			sz = p.long().Sub(p.targetSz)
			isOpen = false
		} else if p.targetSz.Sub(p.long()).GreaterThanOrEqual(util.DecimalOne) && p.dealDir == DealDir_Open && p.canOpen {
			// 补开仓位
			dir = common.OrderDir_Buy
			sz = p.targetSz.Sub(p.long())
			isOpen = true
		}
	} else if p.targetDir == common.OrderDir_Sell {
		if p.short().Sub(p.targetSz).GreaterThanOrEqual(util.DecimalOne) && p.dealDir == DealDir_Close {
			// 平掉相应的张数
			dir = common.OrderDir_Buy
			sz = p.short().Sub(p.targetSz)
			isOpen = false
		} else if p.targetSz.Sub(p.short()).GreaterThanOrEqual(util.DecimalOne) && p.dealDir == DealDir_Open && p.canOpen {
			// 补开仓位
			dir = common.OrderDir_Sell
			sz = p.targetSz.Sub(p.short())
			isOpen = true
		}
	}

	if dir != common.OrderDir_None && sz.IsPositive() {
		//lint:ignore SA4006
		price := decimal.Zero
		if dir == common.OrderDir_Buy {
			price = p.trader.Market().OrderBook().Sell1Price().Mul(decimal.NewFromFloat(1.01))
			if p.maxBuyPrice.IsPositive() && price.GreaterThan(p.maxBuyPrice) {
				price = p.maxBuyPrice
			}
		} else {
			price = p.trader.Market().OrderBook().Buy1Price().Mul(decimal.NewFromFloat(0.99))
			if p.minSellPrice.IsPositive() && price.LessThan(p.minSellPrice) {
				price = p.minSellPrice
			}
		}
		if p.maker {
			// 挂单交易
			if p.stepSz.IsPositive() {
				sz = decimal.Min(sz, p.stepSz)
			}

			if isOpen {
				p.mkOpen.Modify(price, sz, dir, false)
				p.mkClose.Cancel()
			} else {
				p.mkClose.Modify(price, sz, dir, true)
				p.mkOpen.Cancel()
			}

			p.tkOpen.Cancel()
			p.tkClose.Cancel()
		} else if time.Now().Unix() > p.takerCD.Unix() || p.superTaker {
			// 吃单交易
			// 吃单数量受最大滑点的影响
			// 但superTaker订单不受影响
			if !p.superTaker {
				maxTk := decimal.Zero
				if dir == common.OrderDir_Buy {
					maxTk = p.trader.Market().OrderBook().MaxBuyAmountBySlipPoint(p.maxSlipPoint)
				} else {
					maxTk = p.trader.Market().OrderBook().MaxSellAmountBySlipPoint(p.maxSlipPoint)
				}
				sz = decimal.Min(sz, maxTk)
			}

			// 吃单数量受交易步长限制
			if p.stepSz.IsPositive() {
				sz = decimal.Min(sz, p.stepSz)
			}

			if isOpen {
				p.tkOpen.ModifyWithoutOrderModify(price, sz, dir, false) // 吃单交易不要修改订单，否则容易出错（猜）
				p.tkClose.Cancel()
				p.takerCD = time.Now().Add(time.Second)
			} else {
				p.tkClose.ModifyWithoutOrderModify(price, sz, dir, true) // 吃单交易不要修改订单，否则容易出错（猜）
				p.tkOpen.Cancel()
				p.takerCD = time.Now().Add(time.Second)
			}

			p.mkClose.Cancel()
			p.mkOpen.Cancel()
		}
	} else {
		p.dealDone = true
	}
}

// #region 内部逻辑
func (p *PositionManager) long() decimal.Decimal {
	return p.trader.Position().Long()
}

func (p *PositionManager) short() decimal.Decimal {
	return p.trader.Position().Short()
}

func (p *PositionManager) onOpenDeal(deal MakerOrderDeal) {
	if p.onDeal != nil {
		p.onDeal(deal)
	}

	// 日志
	logger.LogInfo(
		p.logPrefix,
		"open order dealing, dir=%s, price=%v, amount=%v, longPos=%v, shortPos=%v, targetPos=%v",
		common.OrderDir2Str(deal.Deal.O.GetDir()),
		deal.Deal.Price, deal.Deal.Amount,
		p.trader.Position().Long(),
		p.trader.Position().Short(),
		p.targetSz,
	)
}

func (p *PositionManager) onCloseDeal(deal MakerOrderDeal) {
	if p.onDeal != nil {
		p.onDeal(deal)
	}

	// 日志
	logger.LogInfo(
		p.logPrefix,
		"close order dealing, dir=%s, price=%v, amount=%v, longPos=%v, shortPos=%v, targetPos=%v",
		common.OrderDir2Str(deal.Deal.O.GetDir()),
		deal.Deal.Price, deal.Deal.Amount,
		p.trader.Position().Long(),
		p.trader.Position().Short(),
		p.targetSz,
	)
}

// #endregion
