/*
 * @Author: aztec
 * @Date: 2022-12-13 10:07:58
 * @Description: 价格突破交易器。顾名思义，价格突破一定限制后直接开仓。价格回撤过大或者达到预期盈利时平仓。
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

// 交易参数
type BreakDealerConfig struct {
	TriggerPriceUpper decimal.Decimal // 上触发价格
	TriggerPriceLower decimal.Decimal // 下触发价格
	TargetSz          decimal.Decimal // 目标成交量
	MaxSlipPoint      decimal.Decimal // 每次吃单最多吃掉盘口多少价格比例

	// 只有当同时满足以下两个回撤条件后（比例值+绝对值），才触发平仓
	// 防止刚刚开仓，Best还不具备参考意义时，就被震出去了
	MaxRetreatRel decimal.Decimal // 可以承受的最大回撤相对比例（从Best到Start）
	MaxRetreatAbs decimal.Decimal // 可以承受的最大回撤绝对比例（以Best作为基准）

	// 一口价盈利率。当此参数有效时，则持续在此位置挂平仓单
	FixedProfitRate decimal.Decimal
}

func EmptyBreakDealerConfig() BreakDealerConfig {
	return BreakDealerConfig{
		TriggerPriceUpper: decimal.NewFromInt(-1),
		TriggerPriceLower: decimal.NewFromInt(-1),
		TargetSz:          decimal.NewFromInt(-1),
		MaxSlipPoint:      decimal.NewFromInt(-1),
		MaxRetreatRel:     decimal.NewFromInt(-1),
		MaxRetreatAbs:     decimal.NewFromInt(-1),
		FixedProfitRate:   decimal.NewFromInt(-1),
	}
}

func (c0 BreakDealerConfig) merge(c1 BreakDealerConfig) BreakDealerConfig {
	cfg := BreakDealerConfig{}

	if !c1.TriggerPriceUpper.IsPositive() {
		cfg.TriggerPriceUpper = c0.TriggerPriceUpper
	} else {
		cfg.TriggerPriceUpper = c1.TriggerPriceUpper
	}

	if !c1.TriggerPriceLower.IsPositive() {
		cfg.TriggerPriceLower = c0.TriggerPriceLower
	} else {
		cfg.TriggerPriceLower = c1.TriggerPriceLower
	}

	if !c1.TargetSz.IsPositive() {
		cfg.TargetSz = c0.TargetSz
	} else {
		cfg.TargetSz = c1.TargetSz
	}

	if !c1.MaxSlipPoint.IsPositive() {
		cfg.MaxSlipPoint = c0.MaxSlipPoint
	} else {
		cfg.MaxSlipPoint = c1.MaxSlipPoint
	}

	if !c1.MaxRetreatRel.IsPositive() {
		cfg.MaxRetreatRel = c0.MaxRetreatRel
	} else {
		cfg.MaxRetreatRel = c1.MaxRetreatRel
	}

	if !c1.MaxRetreatAbs.IsPositive() {
		cfg.MaxRetreatAbs = c0.MaxRetreatAbs
	} else {
		cfg.MaxRetreatAbs = c1.MaxRetreatAbs
	}

	if c1.FixedProfitRate.IsPositive() {
		cfg.FixedProfitRate = c0.FixedProfitRate
	} else {
		cfg.FixedProfitRate = c1.FixedProfitRate
	}

	return cfg
}

// 交易阶段
type BreakDealerPhase int

const (
	BreakDealerPhase_WaitOpen BreakDealerPhase = iota
	BreakDealerPhase_OpenBuy
	BreakDealerPhase_OpenSell
	BreakDealerPhase_Closing
)

func BreakDealerPhase2Str(s BreakDealerPhase) string {
	switch s {
	case BreakDealerPhase_WaitOpen:
		return "WaitOpen"
	case BreakDealerPhase_OpenBuy:
		return "OpenBuy"
	case BreakDealerPhase_OpenSell:
		return "OpenSell"
	case BreakDealerPhase_Closing:
		return "Closing"
	default:
		return ""
	}
}

// 状态
type BreakDealerStatus struct {
	InstId        string          `json:"instId"`
	Status        string          `json:"status"`
	UpperPx       decimal.Decimal `json:"upper_px"`
	LowerPx       decimal.Decimal `json:"lower_px"`
	BestPx        decimal.Decimal `json:"best_px"`
	CurrentPrice  decimal.Decimal `json:"cur_px"`
	MaxRetreatAbs decimal.Decimal `json:"max_retreat_abs"`
	CurRetreatAbs decimal.Decimal `json:"cur_retreat_abs"`
	MaxRetreatRel decimal.Decimal `json:"max_retreat_rel"`
	CurRetreatRel decimal.Decimal `json:"cur_retreat_rel"`
	TargetSize    decimal.Decimal `json:"target_sz"`
	LongSize      decimal.Decimal `json:"long_sz"`
	ShortSize     decimal.Decimal `json:"short_sz"`
	Pause         bool            `json:"paused"`
	Finished      bool            `json:"finished"`
	Quit          bool            `json:"quit"`
}

type BreakDealer struct {
	trader    common.FutureTrader
	logPrefix string

	// 交易配置
	cfg BreakDealerConfig

	// 交易阶段
	phase  BreakDealerPhase
	bestPx decimal.Decimal // 历史最佳价格（从首次成交后开始计算）

	// 下单器
	mkOpen  *Maker
	mkClose *Maker

	// taker冷却控制
	takerCD time.Time

	// 下单标志
	keepOpening bool

	finished bool // 交易是否结束
	pause    bool // 暂停交易
	quit     bool // 立即平仓
}

func (d *BreakDealer) Init(trader common.FutureTrader) {
	d.trader = trader
	d.logPrefix = fmt.Sprintf("breakdealer-%s", trader.Market().Type())
	d.cfg = EmptyBreakDealerConfig()
	d.mkOpen = new(Maker)
	d.mkClose = new(Maker)
	d.mkOpen.Init(trader, false, false, 0, 0, "open")
	d.mkClose.Init(trader, false, false, 0, 0, "close")
	d.mkOpen.Go()
	d.mkClose.Go()
	d.mkOpen.SetDealFn(d.onOpenDeal)
	d.mkClose.SetDealFn(d.onCloseDeal)
	d.bestPx = decimal.NewFromInt(-1)
	d.keepOpening = false
	d.finished = false
	d.pause = false
	d.quit = false

	// 当非空仓启动时，初始化参数
	l := d.long()
	s := d.short()
	if l.IsPositive() {
		d.cfg.TargetSz = l
		d.switchStatus(BreakDealerPhase_OpenBuy)
	} else if s.IsPositive() {
		d.cfg.TargetSz = s
		d.switchStatus(BreakDealerPhase_OpenSell)
	}

	go d.Update()
}

func (d *BreakDealer) IsFinished() bool {
	return d.finished
}

func (d *BreakDealer) Stop() {
	d.finished = true
	d.mkOpen.Stop()
	d.mkClose.Stop()
}

func (d *BreakDealer) GetTrader() common.FutureTrader {
	return d.trader
}

func (d *BreakDealer) switchStatus(s BreakDealerPhase) {
	if d.phase != s {
		logger.LogInfo(d.logPrefix, "status switched from %s to %s", BreakDealerPhase2Str(d.phase), BreakDealerPhase2Str(s))
		d.phase = s

		if d.phase == BreakDealerPhase_OpenBuy || d.phase == BreakDealerPhase_OpenSell {
			d.keepOpening = true
		}
	}
}

func (d *BreakDealer) Status() BreakDealerStatus {
	st := BreakDealerStatus{}
	st.InstId = d.trader.Market().Type()
	st.Status = BreakDealerPhase2Str(d.phase)
	st.UpperPx = d.cfg.TriggerPriceUpper
	st.LowerPx = d.cfg.TriggerPriceLower
	st.BestPx = d.bestPx
	st.CurrentPrice = d.px()
	st.MaxRetreatAbs = d.cfg.MaxRetreatAbs
	st.CurRetreatAbs = d.curRetreatAbs()
	st.MaxRetreatRel = d.cfg.MaxRetreatRel
	st.CurRetreatRel = d.curRetreatRel()
	st.TargetSize = d.cfg.TargetSz
	st.LongSize = d.long()
	st.ShortSize = d.short()
	st.Pause = d.pause
	st.Finished = d.finished
	st.Quit = d.quit
	return st
}

func (d *BreakDealer) TogglePause() {
	d.pause = !d.pause
	logger.LogInfo(d.logPrefix, "pause toggled to %v", d.pause)
}

func (d *BreakDealer) ToggleQuit() {
	d.quit = !d.quit
	logger.LogInfo(d.logPrefix, "quit toggled to %v", d.quit)
}

func (d *BreakDealer) Detach() {
	d.mkOpen.Cancel()
	d.mkClose.Cancel()
	d.finished = true
	logger.LogInfo(d.logPrefix, "detached")
}

func (d *BreakDealer) ModifyConfig(cfg BreakDealerConfig) (valid bool, msg string) {
	// 合并参数
	newCfg := d.cfg.merge(cfg)

	// 上下限价格至少要有一个
	if !newCfg.TriggerPriceLower.IsPositive() && !newCfg.TriggerPriceUpper.IsPositive() {
		return false, fmt.Sprintf("at least on trigger price must be specified")
	}

	// 如果上下限都有，则下限必须大于上限
	if newCfg.TriggerPriceLower.IsPositive() && newCfg.TriggerPriceUpper.IsPositive() {
		if newCfg.TriggerPriceLower.GreaterThanOrEqual(newCfg.TriggerPriceUpper) {
			return false, fmt.Sprintf("triggerPriceUpper(%v) must greater than triggerPriceLower(%v)", newCfg.TriggerPriceUpper, newCfg.TriggerPriceLower)
		}
	}

	// 数量必须大于0
	if !newCfg.TargetSz.IsPositive() {
		return false, fmt.Sprintf("size must be positive")
	}

	// 最大回撤必须在0~1之间
	if newCfg.MaxRetreatAbs.InexactFloat64() <= 0 || newCfg.MaxRetreatAbs.InexactFloat64() > 1 {
		return false, fmt.Sprintf("maxRetreatAbs must between 0~1")
	}

	if newCfg.MaxRetreatRel.InexactFloat64() <= 0 || newCfg.MaxRetreatRel.InexactFloat64() > 1 {
		return false, fmt.Sprintf("maxRetreatRel must between 0~1")
	}

	// 吃单滑点不应大于10%
	if newCfg.MaxSlipPoint.InexactFloat64() <= 0 || newCfg.MaxSlipPoint.InexactFloat64() > 0.1 {
		return false, fmt.Sprintf("maxSlipPoint must between 0~0.1")
	}

	// 价格对齐
	newCfg.TriggerPriceLower = d.trader.Market().AlignPriceNumber(newCfg.TriggerPriceLower)
	newCfg.TriggerPriceUpper = d.trader.Market().AlignPriceNumber(newCfg.TriggerPriceUpper)

	d.cfg = newCfg
	return true, ""
}

func (d *BreakDealer) onOpenDeal(deal MakerOrderDeal) {
	// 当达到目标仓位后，停止开仓标志
	if d.long().GreaterThanOrEqual(d.cfg.TargetSz) || d.short().GreaterThanOrEqual(d.cfg.TargetSz) {
		d.keepOpening = false
	}

	logger.LogInfo(
		d.logPrefix,
		"open order dealing, dir=%s, price=%v, amount=%v, longPos=%v, shortPos=%v, targetPos=%v, status=%s",
		common.OrderDir2Str(deal.Deal.O.GetDir()),
		deal.Deal.Price, deal.Deal.Amount,
		d.trader.Position().Long(),
		d.trader.Position().Short(),
		d.cfg.TargetSz,
		BreakDealerPhase2Str(d.phase),
	)
}

func (d *BreakDealer) onCloseDeal(deal MakerOrderDeal) {
	// 平仓单交易后应立即停止开仓，防止反复交易
	d.keepOpening = false

	// 仓位平完后，交易结束
	if deal.Deal.O.GetDir() == common.OrderDir_Buy && d.trader.Position().Short().IsZero() ||
		deal.Deal.O.GetDir() == common.OrderDir_Sell && d.trader.Position().Long().IsZero() {
		d.finished = true
		logger.LogInfo(d.logPrefix, "deal finished")
	}

	logger.LogInfo(
		d.logPrefix,
		"close order dealing, dir=%s, price=%v, amount=%v, longPos=%v, shortPos=%v, targetPos=%v, status=%s",
		common.OrderDir2Str(deal.Deal.O.GetDir()),
		deal.Deal.Price, deal.Deal.Amount,
		d.trader.Position().Long(),
		d.trader.Position().Short(),
		d.cfg.TargetSz,
		BreakDealerPhase2Str(d.phase),
	)
}

// 方便函数
func (d *BreakDealer) px() decimal.Decimal {
	return d.trader.FutureMarket().MarkPrice()
}

func (d *BreakDealer) long() decimal.Decimal {
	return d.trader.Position().Long()
}

func (d *BreakDealer) short() decimal.Decimal {
	return d.trader.Position().Short()
}

func (d *BreakDealer) takerPxBuy() decimal.Decimal {
	return d.trader.Market().OrderBook().Sell1().Mul(decimal.NewFromFloat(1.01))
}

func (d *BreakDealer) takerPxSell() decimal.Decimal {
	return d.trader.Market().OrderBook().Buy1().Mul(decimal.NewFromFloat(0.99))
}

func (d *BreakDealer) posPrice() decimal.Decimal {
	pos := d.trader.Position()
	if pos.Long().IsPositive() {
		return pos.LongAvgPx()
	} else if pos.Short().IsPositive() {
		return pos.ShortAvgPx()
	} else {
		return decimal.Zero
	}
}

func (d *BreakDealer) maxSlipPoint() decimal.Decimal {
	maxSlipPoint := d.cfg.MaxSlipPoint
	if !maxSlipPoint.IsPositive() {
		maxSlipPoint = decimal.NewFromFloat(0.0001)
	}
	return maxSlipPoint
}

func (d *BreakDealer) curRetreatAbs() decimal.Decimal {
	if d.bestPx.IsPositive() {
		return d.px().Sub(d.bestPx).Div(d.bestPx).Abs()
	} else {
		return decimal.NewFromInt(-1)
	}
}

func (d *BreakDealer) curRetreatRel() decimal.Decimal {
	if d.posPrice().IsPositive() {
		if d.posPrice().Equals(d.bestPx) {
			return decimal.Zero
		} else {
			return d.px().Sub(d.bestPx).Div(d.posPrice().Sub(d.bestPx)).Abs()
		}
	} else {
		return decimal.NewFromInt(-1)
	}
}

func (d *BreakDealer) ConfigStr() string {
	return fmt.Sprintf(
		"triggerPxUpper=%v, triggerPxLower=%v, targetSz=%v, fpr=%v%%, maxSlipPoint=%v%%, maxRetreatRel=%v%%, maxRetreadAbs=%v%%",
		d.cfg.TriggerPriceUpper,
		d.cfg.TriggerPriceLower,
		d.cfg.TargetSz,
		d.cfg.FixedProfitRate.Mul(decimal.NewFromInt(100)),
		d.cfg.MaxSlipPoint.Mul(decimal.NewFromInt(100)),
		d.cfg.MaxRetreatRel.Mul(decimal.NewFromInt(100)),
		d.cfg.MaxRetreatAbs.Mul(decimal.NewFromInt(100)),
	)
}

// 主逻辑
// 分为等待开仓、开仓、等待平仓、平仓四个状态
// 等待开仓时，检查开仓条件是否满足（价格突破）。满足条件后，进入开仓状态
// 开仓状态下，只要仓位未满，且价格优于StartPrice，则持续开仓
// 当存在仓位时，若满足平仓条件，则进入平仓状态
// 平仓状态下，直接吃单平仓直到仓位为空，结束
func (d *BreakDealer) Update() {
	logger.LogInfo(d.logPrefix, "started")
	defer logger.LogInfo(d.logPrefix, "finished")
	ticker := time.NewTicker(time.Millisecond * 100)
	for {
		<-ticker.C

		if d.finished {
			break
		}

		if d.pause {
			d.mkOpen.Cancel()
			d.mkClose.Cancel()
			continue
		}

		if d.quit {
			d.updateClosing()
			continue
		}

		// 否则根据状态进行刷新
		switch d.phase {
		case BreakDealerPhase_WaitOpen:
			d.updateWaitOpen()
		case BreakDealerPhase_OpenBuy:
			d.updateOpenBuy()
		case BreakDealerPhase_OpenSell:
			d.updateOpenSell()
		case BreakDealerPhase_Closing:
			d.updateClosing()
		}
	}
}

func (d *BreakDealer) updateWaitOpen() {
	// 检查价格是否突破
	if d.cfg.TriggerPriceLower.IsPositive() {
		if d.px().LessThan(d.cfg.TriggerPriceLower) {
			// 向下突破，开始开空
			logger.LogImportant(d.logPrefix, "mark-price(%v) lower than lpx(%v)", d.px(), d.cfg.TriggerPriceLower)
			d.switchStatus(BreakDealerPhase_OpenSell)
		}
	}

	if d.cfg.TriggerPriceUpper.IsPositive() {
		if d.px().GreaterThan(d.cfg.TriggerPriceUpper) {
			// 向上突破，开始开多
			logger.LogImportant(d.logPrefix, "mark-price(%v) greater than upx(%v), status switch to OpenSell", d.px(), d.cfg.TriggerPriceUpper)
			d.switchStatus(BreakDealerPhase_OpenBuy)
		}
	}

	d.mkOpen.Cancel()
	d.mkClose.Cancel()
}

func (d *BreakDealer) updateOpenBuy() {
	// 当多仓未达到目标仓位，且当前盘口价格优于初始价格，则持续开仓
	if d.keepOpening && d.long().LessThan(d.cfg.TargetSz) && time.Now().Unix() > d.takerCD.Unix() {
		if d.px().GreaterThan(d.posPrice()) || !d.posPrice().IsPositive() {
			max1 := d.trader.Market().OrderBook().MaxBuyAmountBySlipPoint(d.maxSlipPoint())
			max2 := d.cfg.TargetSz.Sub(d.long())
			size := decimal.Min(max1, max2) // 最大交易数量
			price := d.takerPxBuy()         // 吃单价格
			d.mkOpen.Modify(price, size, common.OrderDir_Buy, false)
			d.takerCD = time.Now().Add(time.Second)
		}
	}

	// 当存在多仓、存在一口价盈利率，则在既定盈利率挂单平仓
	if d.cfg.FixedProfitRate.IsPositive() {
		fpx := d.posPrice().Mul(util.DecimalOne.Add(d.cfg.FixedProfitRate))
		if d.long().IsPositive() && fpx.IsPositive() {
			d.mkClose.Modify(fpx, d.long(), common.OrderDir_Sell, true)
		} else {
			d.mkClose.Cancel()
		}
	}

	// 维护BestPrice
	if d.px().GreaterThan(d.bestPx) {
		d.bestPx = d.px()
	}

	// 当价格达到平仓条件，则切换进平仓状态
	if d.bestPx.IsPositive() {
		rta := d.curRetreatAbs()
		rtr := d.curRetreatRel()

		// 绝对回撤
		cond1 := false
		if rta.GreaterThan(d.cfg.MaxRetreatAbs) {
			cond1 = true
			logger.LogInfo(d.logPrefix, "rta(%.2f) is greater than max-rta(%.2f)", rta, d.cfg.MaxRetreatAbs)
		}

		// 相对回撤
		cond2 := false
		if rtr.GreaterThan(d.cfg.MaxRetreatRel) {
			cond2 = true
			logger.LogInfo(d.logPrefix, "rtr(%.2f) is greater than max-rtr(%.2f)", rtr, d.cfg.MaxRetreatRel)
		}

		if cond1 && cond2 {
			d.switchStatus(BreakDealerPhase_Closing)
		}
	}
}

func (d *BreakDealer) updateOpenSell() {
	// 当空仓未达到目标仓位，且当前盘口价格优于初始价格，则持续开仓
	if d.keepOpening && d.short().LessThan(d.cfg.TargetSz) && time.Now().Unix() > d.takerCD.Unix() {
		if d.px().LessThan(d.posPrice()) || !d.posPrice().IsPositive() {
			max1 := d.trader.Market().OrderBook().MaxSellAmountBySlipPoint(d.maxSlipPoint())
			max2 := d.cfg.TargetSz.Sub(d.short())
			size := decimal.Min(max1, max2) // 最大交易数量
			price := d.takerPxSell()        // 吃单价格
			d.mkOpen.Modify(price, size, common.OrderDir_Sell, false)
			d.takerCD = time.Now().Add(time.Second)
		}
	}

	// 当存在空仓、存在一口价盈利率，则在既定盈利率挂单平仓
	if d.cfg.FixedProfitRate.IsPositive() {
		fpx := d.posPrice().Mul(util.DecimalOne.Sub(d.cfg.FixedProfitRate))
		if d.short().IsPositive() && fpx.IsPositive() {
			d.mkClose.Modify(fpx, d.short(), common.OrderDir_Buy, true)
		} else {
			d.mkClose.Cancel()
		}
	}

	// 维护BestPrice
	if d.px().LessThan(d.bestPx) || d.bestPx.IsNegative() {
		d.bestPx = d.px()
	}

	// 当价格达到平仓条件，则切换进平仓状态
	if d.bestPx.IsPositive() {
		rta := d.curRetreatAbs()
		rtr := d.curRetreatRel()

		// 绝对回撤
		cond1 := false
		if rta.GreaterThan(d.cfg.MaxRetreatAbs) {
			cond1 = true
			logger.LogInfo(d.logPrefix, "rta(%.2f) is greater than max-rta(%.2f)", rta, d.cfg.MaxRetreatAbs)
		}

		// 相对回撤
		cond2 := false
		if rtr.GreaterThan(d.cfg.MaxRetreatRel) {
			cond2 = true
			logger.LogInfo(d.logPrefix, "rtr(%.2f) is greater than max-rtr(%.2f)", rtr, d.cfg.MaxRetreatRel)
		}

		if cond1 && cond2 {
			d.switchStatus(BreakDealerPhase_Closing)
		}
	}
}

func (d *BreakDealer) updateClosing() {
	// 吃单，清空所有仓位
	if d.long().IsPositive() {
		sz := decimal.Max(d.long(), d.trader.Market().OrderBook().MaxSellAmountBySlipPoint(d.maxSlipPoint()))
		px := d.takerPxSell()
		d.mkClose.Modify(px, sz, common.OrderDir_Sell, true)
		d.mkOpen.Cancel()
	} else if d.short().IsPositive() {
		sz := decimal.Max(d.short(), d.trader.Market().OrderBook().MaxBuyAmountBySlipPoint(d.maxSlipPoint()))
		px := d.takerPxBuy()
		d.mkClose.Modify(px, sz, common.OrderDir_Buy, true)
		d.mkOpen.Cancel()
	} else {
		d.finished = true
	}
}
