/*
 * @Author: aztec
 * @Date: 2022-12-08 17:32:57
 * @Description:
 * 反向网格交易器。当价格上突破一个格子时，吃单做多，反之吃单做空
 * 当价格超出网格范围后，挂单平仓
 * 本质上是一种趋势策略
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package adv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/stratergy"
	"github.com/aztecqt/dagger/stratergy/datamanager"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

// 交易阶段
type ReverseGridPhase int

const (
	ReverseGridPhase_WaitActive ReverseGridPhase = iota
	ReverseGridPhase_Dealing
	ReverseGridPhase_Retreat
	ReverseGridPhase_Finished
)

func ReverseGridPhase2Str(phase ReverseGridPhase) string {
	switch phase {
	case ReverseGridPhase_WaitActive:
		return "wait_active"
	case ReverseGridPhase_Dealing:
		return "dealing"
	case ReverseGridPhase_Retreat:
		return "retreat"
	case ReverseGridPhase_Finished:
		return "finished"
	default:
		return "unknown"
	}
}

// 参数
type ReversedGridConfig struct {
	StartPriceAvgPeriod int64   `json:"start_px_avg_period"` // 起始平均价的计算时长
	MaxLongPos          int64   `json:"max_lpos"`            // 最大持仓数量（以usd计算）
	MaxShortPos         int64   `json:"max_spos"`            // 最大持仓数量（以usd计算）
	GridRange           float64 `json:"grid_range"`          // 网格范围（以单边计）
	GridStep            float64 `json:"grid_step"`           // 网格格子大小
	TPStep              int     `json:"tp_step"`             // 止盈价格子数。在超出网格范围多少格之后止盈
}

func (c *ReversedGridConfig) String() string {
	return fmt.Sprintf(
		"maxLong: %d, maxShort: %d, range:%.2f%%, step:%.2f%%, tp at %d step",
		c.MaxLongPos,
		c.MaxShortPos,
		c.GridRange*100,
		c.GridStep*100,
		c.TPStep)
}

// 状态
type ReverseGridStatus struct {
	InstId               string             `json:"inst_id"`
	Phase                string             `json:"phase"`
	ActiveTime           string             `json:"active_time"`
	BasePrice            float64            `json:"base_px"`
	CurrentPrice         float64            `json:"cur_px"`
	LongTakeProfitPrice  float64            `json:"long_tp_px"`
	ShortTakeProfitPrice float64            `json:"short_tp_px"`
	LongPos              float64            `json:"long_pos"`
	ShortPos             float64            `json:"short_pos"`
	OpenBuyRange         string             `json:"ob_range"`
	CloseBuyRange        string             `json:"cb_range"`
	OpenSellRange        string             `json:"os_range"`
	CloseSellRange       string             `json:"cs_range"`
	FrameIndex           int                `json:"frame_index"`
	Config               ReversedGridConfig `json:"config"`
}

type ReversedGrid struct {
	logPrefix            string
	dealType             string
	infContext           *datamanager.InfluxContext // 数据存储
	cfg                  ReversedGridConfig         // 配置
	cfgDirty             bool                       // 配置更新过
	trader               common.FutureTrader        // 交易器
	pm                   *PositionManager           // 这个用来执行吃单交易
	activeTime           time.Time                  // 自动激活时间，过了这个时间会自动激活
	onDeal               OnMakerOrderDeal           // 成交回调
	dlPrice              *stratergy.DataLine        // waitActive阶段用于计算MarkPrice平均值
	basePrice            float64
	longTakeProfitPrice  float64
	shortTakeProfitPrice float64
	p2pOpenBuy           *PriceMap2Position // 价格-仓位映射
	p2pCloseBuy          *PriceMap2Position
	p2pOpenSell          *PriceMap2Position
	p2pCloseSell         *PriceMap2Position
	phase                ReverseGridPhase // 交易阶段
	frameIndex           int
}

func (g *ReversedGrid) Init(
	trader common.FutureTrader,
	cfg ReversedGridConfig,
	onDeal OnMakerOrderDeal,
	activeTime time.Time,
	autoUpdate bool,
	dealType string) {
	if cfg.GridStep >= cfg.GridRange {
		logger.LogPanic(g.logPrefix, "invalid param")
		return
	}

	g.dealType = dealType
	g.logPrefix = fmt.Sprintf("rGrid-%s_%s", trader.FutureMarket().Symbol(), trader.FutureMarket().ContractType())
	g.cfg = cfg
	g.trader = trader
	g.onDeal = onDeal
	g.infContext = datamanager.NewInfluxContext("rgrid", trader.Market().Type())
	g.activeTime = activeTime
	g.dlPrice = new(stratergy.DataLine)
	g.dlPrice.Init("price", 120, 1000, 0)

	g.pm = new(PositionManager)
	g.pm.Init(g.trader, g.onOrderDeal, g.logPrefix, false, false)
	g.pm.SetTaker(true, true)

	g.p2pOpenBuy = NewPriceMap2Position(trader.Market())
	g.p2pCloseBuy = NewPriceMap2Position(trader.Market())
	g.p2pOpenSell = NewPriceMap2Position(trader.Market())
	g.p2pCloseSell = NewPriceMap2Position(trader.Market())

	g.generatePlan(true)
	g.phase = ReverseGridPhase_WaitActive

	if autoUpdate {
		go g.autoUpdate()
	}

	go g.autoSaveData()
}

func (g *ReversedGrid) uninit() {
	g.pm.Uninit()
}

func (g *ReversedGrid) GetConfig() ReversedGridConfig {
	return g.cfg
}

func (g *ReversedGrid) SetConfig(cfg ReversedGridConfig) {
	g.cfg = cfg
	g.cfgDirty = true
}

func (g *ReversedGrid) Status() ReverseGridStatus {
	status := ReverseGridStatus{}
	status.InstId = g.GetMarket().Type()
	status.Phase = ReverseGridPhase2Str(g.phase)
	status.ActiveTime = g.activeTime.Format("2006-01-02 15:04:05")
	status.BasePrice = g.basePrice
	status.CurrentPrice = g.px()
	status.LongTakeProfitPrice = g.longTakeProfitPrice
	status.ShortTakeProfitPrice = g.shortTakeProfitPrice
	status.LongPos = g.long()
	status.ShortPos = g.short()
	status.FrameIndex = g.frameIndex
	status.Config = g.cfg
	if g.longValid() {
		status.OpenBuyRange = g.p2pOpenBuy.String()
		status.CloseBuyRange = g.p2pCloseBuy.String()
	} else {
		status.OpenBuyRange = "n/a"
		status.CloseBuyRange = "n/a"
	}

	if g.shortValid() {
		status.OpenSellRange = g.p2pOpenSell.String()
		status.CloseSellRange = g.p2pCloseSell.String()
	} else {
		status.OpenSellRange = "n/a"
		status.CloseSellRange = "n/a"
	}

	return status
}

func (g *ReversedGrid) StatusStr() string {
	s := g.Status()
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func (g *ReversedGrid) GetTrader() common.FutureTrader {
	return g.trader
}

func (g *ReversedGrid) GetMarket() common.FutureMarket {
	return g.trader.FutureMarket()
}

func (g *ReversedGrid) onOrderDeal(deal MakerOrderDeal) {
	logger.LogInfo(g.logPrefix, "dealing: %s", deal.Deal.O.String())

	// 保存数据
	if g.infContext != nil {
		g.infContext.AddDeal(deal.Deal, g.dealType)
	}

	// 回调外部
	if g.onDeal != nil {
		g.onDeal(deal)
	}
}

func (g *ReversedGrid) Active() {
	if g.phase == ReverseGridPhase_WaitActive {
		// 进入交易状态
		g.switchPhase(ReverseGridPhase_Dealing)
	}
}

func (g *ReversedGrid) Stop() {
	g.pm.Cancel()
}

func (g *ReversedGrid) Retreat() {
	g.switchPhase(ReverseGridPhase_Retreat)
}

func (g *ReversedGrid) Detach() {
	g.switchPhase(ReverseGridPhase_Finished)
}

func (g *ReversedGrid) Finished() bool {
	return g.phase == ReverseGridPhase_Finished
}

func (g *ReversedGrid) autoUpdate() {
	ticker := time.NewTicker(time.Millisecond * 10)
	for !g.Finished() {
		<-ticker.C
		g.Update()
	}
	g.uninit()
}

func (g *ReversedGrid) Update() {
	switch g.phase {
	case ReverseGridPhase_WaitActive:
		g.update_WaitActive()
	case ReverseGridPhase_Dealing:
		g.update_Dealing()
	case ReverseGridPhase_Retreat:
		g.update_Retreat()
	}

	g.pm.Update()
	g.frameIndex++
}

func (g *ReversedGrid) autoSaveData() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		points := make(map[string]float64)
		points["px"] = g.px()
		points["long"] = g.long()
		points["short"] = g.short()
		g.infContext.AddDataPoints(points, time.Now())

		if g.Finished() {
			break
		}
	}
}

func (g *ReversedGrid) update_WaitActive() {
	g.basePrice = g.calcuAvgPrice()
	g.generatePlan(true)

	if time.Now().After(g.activeTime) {
		g.Active()
	}
}

func (g *ReversedGrid) update_Dealing() {
	// 适时更新计划表
	if g.cfgDirty {
		g.generatePlan(true)
		g.cfgDirty = false
	} else {
		g.generatePlan(false)
	}

	// 根据计划表进行交易
	markPrice := g.px()
	if g.longValid() && markPrice > g.longTakeProfitPrice {
		// 价格达标，切换到平仓状态
		g.switchPhase(ReverseGridPhase_Retreat)
		return
	}

	if g.shortValid() && markPrice < g.shortTakeProfitPrice {
		// 价格达标，切换到平仓状态
		g.switchPhase(ReverseGridPhase_Retreat)
		return
	}

	// 价格不达标，继续交易
	// 根据价格计算出当前应持仓位，然后吃单开平仓
	targetDir := common.OrderDir_None
	if g.longValid() && markPrice > g.basePrice {
		// 尝试做多
		targetDir = common.OrderDir_Buy
		posopen := g.p2pOpenBuy.GetPosition(markPrice)
		posclose := g.p2pCloseBuy.GetPosition(markPrice)
		if posopen > g.long() {
			g.pm.ModifyTargetSizeWithDealDir(decimal.NewFromFloat(posopen), targetDir, DealDir_Open)
			logger.LogInfo(g.logPrefix, "target size modified up to %v(%s)", posopen, common.OrderDir2Str(targetDir))
		} else if posclose < g.long() {
			g.pm.ModifyTargetSizeWithDealDir(decimal.NewFromFloat(posclose), targetDir, DealDir_Close)
			logger.LogInfo(g.logPrefix, "target size modified down to %v(%s)", posclose, common.OrderDir2Str(targetDir))
		}
	} else if g.shortValid() && markPrice < g.basePrice {
		// 尝试做空
		targetDir = common.OrderDir_Sell
		posopen := g.p2pOpenSell.GetPosition(markPrice)
		posclose := g.p2pCloseSell.GetPosition(markPrice)
		if posopen > g.short() {
			g.pm.ModifyTargetSizeWithDealDir(decimal.NewFromFloat(posopen), targetDir, DealDir_Open)
			logger.LogInfo(g.logPrefix, "target size modified up to %v(%s)", posopen, common.OrderDir2Str(targetDir))
		} else if posclose < g.short() {
			g.pm.ModifyTargetSizeWithDealDir(decimal.NewFromFloat(posclose), targetDir, DealDir_Close)
			logger.LogInfo(g.logPrefix, "target size modified down to %v(%s)", posclose, common.OrderDir2Str(targetDir))
		}
	}

	if g.long() > 0 && markPrice < g.basePrice || g.short() > 0 && markPrice > g.basePrice {
		g.pm.ModifyTargetSizeWithDealDir(decimal.Zero, common.OrderDir_None, DealDir_Close)
	}
}

func (g *ReversedGrid) update_Retreat() {
	// 结束条件
	if g.long() == 0 && g.short() == 0 {
		logger.LogInfo(g.logPrefix, "no position in retreat status? finish")
		g.switchPhase(ReverseGridPhase_Finished)
	}
}

func (g *ReversedGrid) generatePlan(forceRegen bool) {
	// 计算中点价格和最大仓位
	needGen := false
	if g.basePrice == 0 {
		g.basePrice = g.GetMarket().OrderBook().MiddlePrice().InexactFloat64()
		needGen = true
	} else {
		if g.longOnly() {
			if g.long() == 0 {
				sell1 := g.trader.Market().OrderBook().Sell1Price().InexactFloat64()
				if sell1 < g.basePrice {
					g.basePrice = sell1
					needGen = true
				}
			}
		} else if g.shortOnly() {
			if g.short() == 0 {
				buy1 := g.trader.Market().OrderBook().Buy1Price().InexactFloat64()
				if buy1 > g.basePrice {
					g.basePrice = buy1
					needGen = true
				}
			}
		}
	}

	if !needGen && !forceRegen {
		return
	}

	maxLong :=
		common.USDT2ContractAmount(
			decimal.NewFromInt(g.cfg.MaxLongPos),
			g.trader.FutureMarket()).InexactFloat64()

	maxShort :=
		common.USDT2ContractAmount(
			decimal.NewFromInt(g.cfg.MaxShortPos),
			g.trader.FutureMarket()).InexactFloat64()

	// 设置价格-仓位映射
	if maxLong > 0 {
		g.p2pOpenBuy.SetPriceRange(g.basePrice*(1+g.cfg.GridStep), g.basePrice*(1+g.cfg.GridRange))
		g.p2pCloseBuy.SetPriceRange(g.basePrice, g.basePrice*(1+g.cfg.GridRange-g.cfg.GridStep))
		g.p2pOpenBuy.SetPositionRange(0, maxLong)
		g.p2pCloseBuy.SetPositionRange(0, maxLong)
		g.longTakeProfitPrice = g.basePrice * (1 + g.cfg.GridRange + g.cfg.GridStep*float64(g.cfg.TPStep))
		g.longTakeProfitPrice = g.trader.Market().AlignPriceNumber(decimal.NewFromFloat(g.longTakeProfitPrice)).InexactFloat64()
	}

	if maxShort > 0 {
		g.p2pOpenSell.SetPriceRange(g.basePrice*(1-g.cfg.GridStep), g.basePrice*(1-g.cfg.GridRange))
		g.p2pCloseSell.SetPriceRange(g.basePrice, g.basePrice*(1-g.cfg.GridRange+g.cfg.GridStep))
		g.p2pOpenSell.SetPositionRange(0, maxShort)
		g.p2pCloseSell.SetPositionRange(0, maxShort)
		g.shortTakeProfitPrice = g.basePrice * (1 - g.cfg.GridRange - g.cfg.GridStep*float64(g.cfg.TPStep))
		g.shortTakeProfitPrice = g.trader.Market().AlignPriceNumber(decimal.NewFromFloat(g.shortTakeProfitPrice)).InexactFloat64()
	}
}

func (g *ReversedGrid) switchPhase(phase ReverseGridPhase) {
	if g.phase != phase {
		// 状态切换逻辑
		if phase == ReverseGridPhase_Retreat {
			g.pm.SetTaker(false, false)
			g.pm.ModifyTargetSize(decimal.Zero, common.OrderDir_None)
		}

		logger.LogInfo(
			g.logPrefix,
			"phase switching from %s to %s",
			ReverseGridPhase2Str(g.phase),
			ReverseGridPhase2Str(phase))
		g.phase = phase
	}
}

// #region 内部函数
func (d *ReversedGrid) px() float64 {
	return d.trader.FutureMarket().MarkPrice().InexactFloat64()
}

func (d *ReversedGrid) long() float64 {
	return d.trader.Position().Long().InexactFloat64()
}

func (d *ReversedGrid) short() float64 {
	return d.trader.Position().Short().InexactFloat64()
}

func (d *ReversedGrid) longAvgPrice() float64 {
	return d.trader.Position().LongAvgPx().InexactFloat64()
}

func (d *ReversedGrid) shortAvgPrice() float64 {
	return d.trader.Position().ShortAvgPx().InexactFloat64()
}

func (d *ReversedGrid) longOnly() bool {
	return d.longValid() && !d.shortValid()
}

func (d *ReversedGrid) shortOnly() bool {
	return d.shortValid() && !d.longValid()
}

func (d *ReversedGrid) longValid() bool {
	return d.p2pOpenBuy.Valid() && d.p2pCloseBuy.Valid()
}

func (d *ReversedGrid) shortValid() bool {
	return d.p2pOpenSell.Valid() && d.p2pCloseSell.Valid()
}

// 计算近期价格平均值
func (d *ReversedGrid) calcuAvgPrice() float64 {
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
