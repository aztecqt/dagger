/*
 * @Author: aztec
 * @Date: 2022-12-14 10:07:58
 * @Description: 价格突破交易器。顾名思义，价格突破一定限制后直接开仓。价格回撤过大或者达到预期盈利时平仓。
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package adv

import (
	"fmt"
	"math"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/framework"
	"github.com/aztecqt/dagger/stratergy/datamanager"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

// 交易器参数
type BreakDealerV2Config struct {
	StartPriceAvgPeriod int64   `json:"start_px_avg_period"`  // 起始平均价的计算时长
	BreakObservePeriod  int64   `json:"break_observe_period"` // 突破观察期。超过这个时间还没突破则结束
	MaxLongPos          int64   `json:"max_lpos"`             // 最大持仓数量（以usd计算）
	MaxShortPos         int64   `json:"max_spos"`             // 最大持仓数量（以usd计算）
	DealStep            int64   `json:"step"`                 // 交易步长。默认为一步到位
	BreakPR             float64 `json:"break_pr"`             // 价格突破阈值。价格突破后决定交易方向
	DealPR              float64 `json:"deal_pr"`              // 交易价格阈值。价格离开起始价格过远后不再交易，以降提高胜率。
	StopProfitPR0       float64 `json:"stop_pft0"`            // 止盈价起始
	StopProfitPR1       float64 `json:"stop_pft1"`            // 止盈价结束
	MaxRetreatAbs       float64 `json:"rta"`                  // 最大回撤率-绝对值（从最佳价格起算）
	MaxRetreatRel       float64 `json:"rtr"`                  // 最大回撤率-相对值（从最佳价格算到仓位价格）
}

// 交易器状态
type BreakDealerV2Status struct {
	InstId       string              `json:"inst_id"`
	Phase        string              `json:"phase"`
	StartPrice   float64             `json:"start_px"`
	CurrentPrice float64             `json:"cur_px"`
	BestPrice    float64             `json:"best_px"`
	ActiveTime   string              `json:"active_time"`
	CurDir       string              `json:"dir"`
	MaxRealPos   int                 `json:"max_real_pos"`
	CurRealPos   int                 `json:"cur_real_pos"`
	PosAvgPrice  float64             `json:"pos_avg_px"`
	ProfitRate   float64             `json:"profit_rate"`
	Rta          float64             `json:"rta"`
	Rtr          float64             `json:"rtr"`
	Config       BreakDealerV2Config `json:"config"`
}

// 交易阶段
type dealerPhase int

const (
	// 前置条件未达成，等待激活
	// 此时只需要收集最新价格，计算近期平均价格即可
	// 该平均价格，会在状态切换时，成为起始价格
	dealerPhase_WaitActive dealerPhase = iota

	// 等待价格突破阈值
	// 突破的基准价格，应为激活前N秒的平均价格
	// 突破后，进入下一状态
	dealerPhase_WaitBreak

	// 交易状态。当仓位未满，且当前价格离开起始价格没有超出范围时，持续吃单交易。
	// 同时记录一个“MaxRealPosition”数值，然后根据MaxRealPosition、CurrentPosition、以及止盈利润率范围，尝试挂单平掉当前的仓位
	// 另外要检测“BestPrice”以及最大回撤率。当回撤过大时，进入强制平仓状态
	dealerPhase_Dealing

	// 尽快平调所有仓位(挂单or吃单?)
	dealerPhase_Retreat

	// 因出错或者交易完成而结束
	dealerPhase_Finished
)

func dealerPhase2Str(s dealerPhase) string {
	switch s {
	case dealerPhase_WaitActive:
		return "wait_active"
	case dealerPhase_WaitBreak:
		return "wait_break"
	case dealerPhase_Dealing:
		return "dealing"
	case dealerPhase_Retreat:
		return "retreat"
	case dealerPhase_Finished:
		return "finished"
	default:
		return "unknown"
	}
}

type BreakDealerV2 struct {
	logPrefix  string
	dealType   string
	trader     common.FutureTrader
	cfg        BreakDealerV2Config
	phase      dealerPhase
	pm         *PositionManager
	infContext *datamanager.InfluxContext

	dlPrice             *framework.DataLine // waitActive阶段用于计算MarkPrice平均值
	startPrice          float64             // 起始价格
	bestPrice           float64             // 最佳价格
	activeTime          time.Time           // 激活时间
	maxRealPositionSize float64             // 真实最大仓位，用于计算平仓价格和数量等
	rta                 float64             // 当前绝对回撤率
	rtr                 float64             // 当前相对回撤比例
	mkTakeProfit        *Maker              // 止盈平仓订单
	mkRetreat           *Maker              // 止损平仓订单
}

func (d *BreakDealerV2) Init(
	trader common.FutureTrader,
	cfg BreakDealerV2Config,
	activeTime time.Time,
	autoupdate bool,
	dealType string) {
	d.dealType = dealType
	d.trader = trader
	d.cfg = cfg
	d.logPrefix = fmt.Sprintf("dealer-%s_%s", d.trader.FutureMarket().Symbol(), d.trader.FutureMarket().ContractType())
	d.activeTime = activeTime
	d.dlPrice = new(framework.DataLine)
	d.dlPrice.Init("price", 120, 1000, 0)
	d.infContext = datamanager.NewInfluxContext("break_dealer", d.trader.Market().Type())

	// 检查参数
	if d.cfg.StopProfitPR0 >= d.cfg.StopProfitPR1 {
		logger.LogPanic(d.logPrefix, "stop_pft1 must greater than stop_pft0")
	}

	if d.cfg.DealPR > d.cfg.StopProfitPR0 || d.cfg.DealPR > d.cfg.StopProfitPR1 {
		logger.LogPanic(d.logPrefix, "deal_pr must lesser than stop_pft")
	}

	if d.cfg.DealPR < d.cfg.BreakPR {
		logger.LogPanic(d.logPrefix, "deal_pr must greater than break_pr")
	}

	// 交易相关
	d.pm = new(PositionManager)
	d.pm.Init(d.trader, d.onDeal, d.logPrefix, false, false)
	d.pm.SetTaker(true, true)
	d.mkTakeProfit = new(Maker)
	d.mkTakeProfit.Init(d.trader, true, false, true, 0, 0, "take-profit")
	d.mkTakeProfit.SetDealFn(d.onDeal)
	d.mkRetreat = new(Maker)
	d.mkRetreat.Init(d.trader, true, false, true, 0, 0, "retreat")
	d.mkRetreat.SetDealFn(d.onDeal)

	for !d.trader.Ready() {
		time.Sleep(time.Millisecond * 100)
	}

	// 检查账户资金是否足够
	maxl := float64(d.maxLong())
	maxs := float64(d.maxShort())
	aval := d.trader.AvailableAmount(common.OrderDir_Buy, decimal.Zero).InexactFloat64()
	avas := d.trader.AvailableAmount(common.OrderDir_Sell, decimal.Zero).InexactFloat64()

	if maxl > aval {
		logger.LogPanic(d.logPrefix, "can't open %v long, only %v avilable", maxl, aval)
	}

	if maxs > avas {
		logger.LogPanic(d.logPrefix, "can't open %v short, only %v avilable", maxs, avas)
	}

	// 自动更新
	if autoupdate {
		go func() {
			ticker := time.NewTicker(time.Millisecond * 10)
			for !d.finished() {
				<-ticker.C
				d.Update()
			}
		}()
	}

	// 数据保存
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			<-ticker.C
			points := make(map[string]float64)
			points["px"] = d.markPrice()
			points["long"] = d.long()
			points["short"] = d.short()
			d.infContext.AddDataPoints(points, time.Now())

			if d.finished() {
				break
			}
		}
	}()

	logger.LogInfo(d.logPrefix, "created")
}

func (d *BreakDealerV2) Stop() {
	// 撤销全部订单
	d.pm.Cancel()
	logger.LogInfo(d.logPrefix, "stop")
}

func (d *BreakDealerV2) Retreat() {
	d.switchPhase(dealerPhase_Retreat)
	logger.LogInfo(d.logPrefix, "retreat")
}

func (d *BreakDealerV2) GetTrader() common.FutureTrader {
	return d.trader
}

func (d *BreakDealerV2) GetMarket() common.FutureMarket {
	return d.trader.FutureMarket()
}

func (d *BreakDealerV2) GetConfig() BreakDealerV2Config {
	return d.cfg
}

func (d *BreakDealerV2) SetConfig(cfg BreakDealerV2Config) {
	d.cfg = cfg
	logger.LogInfo(d.logPrefix, "config modified")
}

func (d *BreakDealerV2) switchPhase(s dealerPhase) {
	if d.phase != s {
		logger.LogInfo(d.logPrefix, "phase switching from %s to %s", dealerPhase2Str(d.phase), dealerPhase2Str(s))
		d.phase = s
	}
}

func (d *BreakDealerV2) finished() bool {
	return d.phase == dealerPhase_Finished
}

func (d *BreakDealerV2) Update() {
	switch d.phase {
	case dealerPhase_WaitActive:
		d.update_WaitActive()
	case dealerPhase_WaitBreak:
		d.update_WaitBreak()
	case dealerPhase_Dealing:
		d.update_Dealing()
	case dealerPhase_Retreat:
		d.update_Retreat()
	}
}

func (d *BreakDealerV2) update_WaitActive() {
	// 收集价格，计算近期价格平均值
	d.dlPrice.Update(time.Now().UnixMilli(), d.middlePrice())
	d.startPrice = d.calcuAvgPrice()

	// 到时间后，切换到下一状态
	if time.Now().After(d.activeTime) {
		logger.LogInfo(d.logPrefix, "dealer active! start price: %.4f", d.startPrice)
		d.switchPhase(dealerPhase_WaitBreak)
	}
}

func (d *BreakDealerV2) update_WaitBreak() {
	// 此时必须已经计算好起始价格
	if d.startPrice == 0 {
		logger.LogImportant(d.logPrefix, "no start price! finishing")
		d.switchPhase(dealerPhase_Finished)
		return
	}

	// 如果超过了观察窗口期还没有突破，则结束
	now := time.Now()
	if now.Unix()-d.activeTime.Unix() > d.cfg.BreakObservePeriod {
		logger.LogImportant(d.logPrefix, "didn't break within %d s, finishing", d.cfg.BreakObservePeriod)
		d.switchPhase(dealerPhase_Finished)
		return
	}

	// 如果价格产生了足够大的突破，则设定目标仓位、进入交易阶段
	breakpr := (d.middlePrice() - d.startPrice) / d.startPrice
	breakprLimit := d.cfg.BreakPR
	if breakpr > breakprLimit {
		targetSz := decimal.NewFromInt(d.maxLong())
		stepSz := decimal.NewFromInt(d.dealStep())
		maxBuyPrice := decimal.NewFromFloat(d.startPrice * (1 + d.cfg.DealPR))
		logger.LogInfo(d.logPrefix, "up breaking! curpx: %f, startpx: %f, pr: %f, prlimit: %f",
			d.middlePrice(),
			d.startPrice,
			breakpr,
			breakprLimit)
		logger.LogInfo(d.logPrefix, "targetSz set to %v, max buy price:%v, stepSz:%v", targetSz, maxBuyPrice, stepSz)
		d.pm.ModifyTargetSize(targetSz, common.OrderDir_Buy)
		d.pm.SetMaxBuyPrice(maxBuyPrice)
		d.pm.SetStepSize(stepSz)
		d.switchPhase(dealerPhase_Dealing)
	} else if breakpr < -breakprLimit {
		targetSz := decimal.NewFromInt(d.maxShort())
		stepSz := decimal.NewFromInt(d.dealStep())
		minSellPrice := decimal.NewFromFloat(d.startPrice * (1 - d.cfg.DealPR))
		logger.LogInfo(d.logPrefix, "down breaking! curpx: %f, startpx: %f, pr: %f, prlimit: %f",
			d.middlePrice(),
			d.startPrice,
			breakpr,
			breakprLimit)
		logger.LogInfo(d.logPrefix, "targetSz set to %v, min sell price:%v, stepSz:%v", targetSz, minSellPrice, stepSz)
		d.pm.ModifyTargetSize(targetSz, common.OrderDir_Sell)
		d.pm.SetMinSellPrice(minSellPrice)
		d.pm.SetStepSize(stepSz)
		d.switchPhase(dealerPhase_Dealing)
	}
}

func (d *BreakDealerV2) update_Dealing() {
	// 交易阶段，position manager会执行开仓
	d.pm.Update()

	// 同时要计算平仓价格，挂单平仓
	d.refreshMaxRealPosition()
	if d.isLong() && d.maxRealPositionSize > 0 {
		dir := common.OrderDir_Sell
		rsz := d.long()
		sz := math.Min(d.maxRealPositionSize/10, rsz)
		sz = math.Max(sz, 1)
		px0 := d.startPrice * (1 + d.cfg.StopProfitPR0)
		px1 := d.startPrice * (1 + d.cfg.StopProfitPR1)
		px := util.LerpFloat(px0, px1, rsz/d.maxRealPositionSize)
		d.mkTakeProfit.Modify(decimal.NewFromFloat(px), decimal.NewFromFloat(sz), dir, true)
	} else if d.isShort() && d.maxRealPositionSize > 0 {
		dir := common.OrderDir_Buy
		rsz := d.short()
		sz := math.Min(d.maxRealPositionSize/10, rsz)
		sz = math.Max(sz, 1)
		px0 := d.startPrice * (1 - d.cfg.StopProfitPR0)
		px1 := d.startPrice * (1 - d.cfg.StopProfitPR1)
		px := util.LerpFloat(px0, px1, rsz/d.maxRealPositionSize)
		d.mkTakeProfit.Modify(decimal.NewFromFloat(px), decimal.NewFromFloat(sz), dir, true)
	} else {
		d.mkTakeProfit.Cancel()
	}
	d.mkRetreat.Cancel()

	// 最后要监控最大回撤，如果回撤过大则直接进入撤退状态
	d.refreshBestPrice()
	if d.isLong() {
		d.rta = (d.bestPrice - d.middlePrice()) / d.bestPrice
		d.rtr = (d.bestPrice - d.middlePrice()) / (d.bestPrice - d.startPrice)
	} else if d.isShort() {
		d.rta = (d.middlePrice() - d.bestPrice) / d.bestPrice
		d.rtr = (d.middlePrice() - d.bestPrice) / (d.startPrice - d.bestPrice)
	}

	if d.rta > d.cfg.MaxRetreatAbs && d.rtr > d.cfg.MaxRetreatRel {
		logger.LogInfo(
			d.logPrefix,
			"rta=%.2f%%, rtr=%.2f%%, startpx=%.4f, bestpx=%.4f, curpx=%.4f, retreating",
			d.rta*100,
			d.rtr*100,
			d.startPrice,
			d.bestPrice,
			d.middlePrice())
		d.pm.Cancel()
		d.switchPhase(dealerPhase_Retreat)
	}
}

func (d *BreakDealerV2) update_Retreat() {
	if d.long() == 0 && d.short() == 0 {
		logger.LogInfo(d.logPrefix, "no position in retreat status? finish")
		d.switchPhase(dealerPhase_Finished)
		return
	}

	// 盘口挂单平仓
	if d.isLong() {
		px := d.trader.Market().OrderBook().Buy1Price().Mul(decimal.NewFromFloat(0.99))
		rsz := d.long()
		sz := math.Min(d.maxRealPositionSize/5, rsz)
		sz = math.Max(sz, 1)
		d.mkRetreat.Modify(px, decimal.NewFromFloat(sz), common.OrderDir_Sell, true)
	} else if d.isShort() {
		px := d.trader.Market().OrderBook().Sell1Price().Mul(decimal.NewFromFloat(1.01))
		rsz := d.short()
		sz := math.Min(d.maxRealPositionSize/5, rsz)
		sz = math.Max(sz, 1)
		d.mkRetreat.Modify(px, decimal.NewFromFloat(sz), common.OrderDir_Buy, true)
	}
	d.mkTakeProfit.Cancel()
}

// #region 参数转换
func (d *BreakDealerV2) maxLong() int64 {
	cfgvalue := decimal.NewFromInt(d.cfg.MaxLongPos)
	maxLong := common.USDT2ContractAmountAtLeast1(cfgvalue, d.trader.FutureMarket())
	return maxLong.IntPart()
}

func (d *BreakDealerV2) maxShort() int64 {
	cfgvalue := decimal.NewFromInt(d.cfg.MaxShortPos)
	maxShort := common.USDT2ContractAmountAtLeast1(cfgvalue, d.trader.FutureMarket())
	return maxShort.IntPart()
}

func (d *BreakDealerV2) dealStep() int64 {
	cfgvalue := decimal.NewFromInt(d.cfg.DealStep)
	dealStep := common.USDT2ContractAmountAtLeast1(cfgvalue, d.trader.FutureMarket())
	return dealStep.IntPart()
}

// #endregion

// #region 内部函数
func (d *BreakDealerV2) middlePrice() float64 {
	return d.trader.Market().OrderBook().MiddlePrice().InexactFloat64()
}

func (d *BreakDealerV2) markPrice() float64 {
	return d.trader.FutureMarket().MarkPrice().InexactFloat64()
}

func (d *BreakDealerV2) long() float64 {
	return d.trader.Position().Long().InexactFloat64()
}

func (d *BreakDealerV2) short() float64 {
	return d.trader.Position().Short().InexactFloat64()
}

func (d *BreakDealerV2) longAvgPrice() float64 {
	return d.trader.Position().LongAvgPx().InexactFloat64()
}

func (d *BreakDealerV2) shortAvgPrice() float64 {
	return d.trader.Position().ShortAvgPx().InexactFloat64()
}

// 交易方向是否为做多
func (d *BreakDealerV2) isLong() bool {
	return d.pm.TargetDir() == common.OrderDir_Buy
}

// 交易方向是否为做空
func (d *BreakDealerV2) isShort() bool {
	return d.pm.TargetDir() == common.OrderDir_Sell
}

// 计算近期价格平均值
func (d *BreakDealerV2) calcuAvgPrice() float64 {
	nowMs := time.Now().UnixMilli()
	total := 0.0
	count := 0
	for i := d.dlPrice.Length() - 1; i >= 0; i-- {
		if du, ok := d.dlPrice.GetData(i); ok {
			if nowMs-du.MS <= d.cfg.StartPriceAvgPeriod*1000 {
				total += du.V
				count++
			} else {
				break
			}
		}
	}

	if count > 0 {
		avg := total / float64(count)
		avgAligned := d.trader.Market().AlignPrice(
			decimal.NewFromFloat(avg),
			common.OrderDir_Buy,
			false).InexactFloat64()
		return avgAligned
	} else {
		return 0
	}
}

// 刷新最大真实仓位
func (d *BreakDealerV2) refreshMaxRealPosition() {
	if d.isLong() && d.long() > d.maxRealPositionSize {
		d.maxRealPositionSize = d.long()
		logger.LogInfo(d.logPrefix, "maxRealPosition(long) grow to %d", int(d.maxRealPositionSize))
	} else if d.isShort() && d.short() > d.maxRealPositionSize {
		d.maxRealPositionSize = d.short()
		logger.LogInfo(d.logPrefix, "maxRealPosition(short) grow to %d", int(d.maxRealPositionSize))
	}
}

// 刷新最佳价格
func (d *BreakDealerV2) refreshBestPrice() {
	if d.bestPrice == 0 {
		d.bestPrice = d.middlePrice()
		logger.LogInfo(d.logPrefix, "best price set to %f", d.bestPrice)
	} else {
		if d.isLong() && d.middlePrice() > d.bestPrice {
			d.bestPrice = d.middlePrice()
			logger.LogInfo(d.logPrefix, "best price grow to %f", d.bestPrice)
		} else if d.isShort() && d.middlePrice() < d.bestPrice {
			d.bestPrice = d.middlePrice()
			logger.LogInfo(d.logPrefix, "best price drop to %f", d.bestPrice)
		}
	}
}

// 成交回调
func (d *BreakDealerV2) onDeal(deal MakerOrderDeal) {
	// 保存数据
	d.infContext.AddDeal(deal.Deal, d.dealType)

	// 检测finish
	if d.isLong() && d.long() == 0 || d.isShort() && d.short() == 0 {
		logger.LogInfo(d.logPrefix, "position cleared, finish")
		d.switchPhase(dealerPhase_Finished)
	}
}

// 状态
func (d *BreakDealerV2) Status() BreakDealerV2Status {
	status := BreakDealerV2Status{}
	status.InstId = d.trader.Market().Type()
	status.Phase = dealerPhase2Str(d.phase)
	status.StartPrice = d.startPrice
	status.CurrentPrice = d.markPrice()
	status.BestPrice = d.bestPrice
	status.ActiveTime = d.activeTime.Format("2006-01-02 15:04:05")
	status.CurDir = common.OrderDir2Str(d.pm.TargetDir())
	status.MaxRealPos = int(d.maxRealPositionSize)
	status.Rta = d.rta
	status.Rtr = d.rtr
	status.Config = d.cfg
	status.Config.MaxLongPos = d.maxLong() // 转换为张数，下同
	status.Config.MaxShortPos = d.maxShort()
	status.Config.DealStep = d.dealStep()

	if d.long() > 0 {
		status.CurRealPos = int(d.long())
		if d.longAvgPrice() > 0 {
			status.PosAvgPrice = d.longAvgPrice()
			status.ProfitRate = (d.markPrice() - d.longAvgPrice()) / d.longAvgPrice()
		}
	} else if d.short() > 0 {
		status.CurRealPos = -int(d.short())
		if d.shortAvgPrice() > 0 {
			status.PosAvgPrice = d.shortAvgPrice()
			status.ProfitRate = (d.shortAvgPrice() - d.markPrice()) / d.shortAvgPrice()
		}
	}

	return status
}

// #endregion
