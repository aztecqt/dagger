/*
 * @Author: aztec
 * @Date: 2023-01-07 09:31:45
 * @Description: 正向网格交易器，固定范围，带止损
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package adv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/stratergy/datamanager"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

// 交易阶段
type GridPhase int

const (
	GridPhase_WaitActive GridPhase = iota
	GridPhase_Dealing
	GridPhase_Retreat
	GridPhase_Finished
)

func GridPhase2Str(phase GridPhase) string {
	switch phase {
	case GridPhase_WaitActive:
		return "wait_active"
	case GridPhase_Dealing:
		return "dealing"
	case GridPhase_Retreat:
		return "retreat"
	case GridPhase_Finished:
		return "finished"
	default:
		return "unknown"
	}
}

// 参数
type GridConfig struct {
	MaxLongPos  int64   `json:"max_lpos"`   // 最大持仓数量（以usd计算）
	MaxShortPos int64   `json:"max_spos"`   // 最大持仓数量（以usd计算）
	BasePrice   float64 `json:"base_px"`    // 中心点价格
	GridRange   float64 `json:"grid_range"` // 网格范围（以单边计）
	GridStep    float64 `json:"grid_step"`  // 网格格子大小
	SLStep      int     `json:"tp_step"`    // 止损价格子数。在超出网格范围多少格之后止损
}

func (c *GridConfig) String() string {
	return fmt.Sprintf(
		"maxLong: %d, maxShort: %d, range:%.2f%%, step:%.2f%%, sl at %d step",
		c.MaxLongPos,
		c.MaxShortPos,
		c.GridRange*100,
		c.GridStep*100,
		c.SLStep)
}

// 状态
type GridStatus struct {
	InstId             string     `json:"inst_id"`
	Phase              string     `json:"phase"`
	TryRetreat         bool       `json:"try_retreat"`
	ActiveTime         string     `json:"active_time"`
	BasePrice          float64    `json:"base_px"`
	CurrentPrice       float64    `json:"cur_px"`
	LongStopLossPrice  float64    `json:"long_ls_px"`
	ShortStopLossPrice float64    `json:"short_ls_px"`
	LongPos            float64    `json:"long_pos"`
	ShortPos           float64    `json:"short_pos"`
	OpenBuyRange       string     `json:"ob_range"`
	CloseBuyRange      string     `json:"cb_range"`
	OpenSellRange      string     `json:"os_range"`
	CloseSellRange     string     `json:"cs_range"`
	FrameIndex         int        `json:"frame_index"`
	Config             GridConfig `json:"config"`
}

type GridPlanUnit struct {
	price    float64
	position float64
}

type Grid struct {
	logPrefix              string
	dealType               string
	infContext             *datamanager.InfluxContext // 数据存储
	cfg                    GridConfig                 // 配置
	cfgDirty               bool                       // 配置更新过
	trader                 common.FutureTrader        // 交易器
	pm                     *PositionManager           // 这个用来执行挂单交易
	activeTime             time.Time                  // 自动激活时间，过了这个时间会自动激活
	onDeal                 OnMakerOrderDeal           // 成交回调
	longStopLossPrice      float64
	shortStopLossPrice     float64
	longStopLossStartTime  time.Time
	shortStopLossStartTime time.Time
	p2pOpenBuy             *PriceMap2Position // 价格-仓位映射
	p2pCloseBuy            *PriceMap2Position
	p2pOpenSell            *PriceMap2Position
	p2pCloseSell           *PriceMap2Position
	phase                  GridPhase // 交易阶段
	tryRetreat             bool      // 下次平仓后撤退
	frameIndex             int
}

func (g *Grid) Init(
	trader common.FutureTrader,
	cfg GridConfig,
	onDeal OnMakerOrderDeal,
	activeTime time.Time,
	autoUpdate bool,
	dealType string) {
	g.dealType = dealType
	g.logPrefix = fmt.Sprintf("grid-%s_%s", trader.FutureMarket().Symbol(), trader.FutureMarket().ContractType())
	g.cfg = cfg
	g.trader = trader
	g.onDeal = onDeal
	g.infContext = datamanager.NewInfluxContext("grid", trader.Market().Type())
	g.activeTime = activeTime

	g.pm = new(PositionManager)
	g.pm.Init(g.trader, g.onOrderDeal, g.logPrefix)
	g.pm.SetTaker(false, false)

	g.p2pOpenBuy = NewPriceMap2Position(trader.Market())
	g.p2pCloseBuy = NewPriceMap2Position(trader.Market())
	g.p2pOpenSell = NewPriceMap2Position(trader.Market())
	g.p2pCloseSell = NewPriceMap2Position(trader.Market())

	g.generatePlan()
	g.phase = GridPhase_WaitActive

	if autoUpdate {
		go g.autoUpdate()
	}

	go g.autoSaveData()
}

func (g *Grid) uninit() {
	g.pm.Uninit()
}

func (g *Grid) GetConfig() GridConfig {
	return g.cfg
}

func (g *Grid) SetConfig(cfg GridConfig) {
	g.cfg = cfg
	g.cfgDirty = true
}

func (g *Grid) Status() GridStatus {
	status := GridStatus{}
	status.InstId = g.GetMarket().Type()
	status.Phase = GridPhase2Str(g.phase)
	status.TryRetreat = g.tryRetreat
	status.ActiveTime = g.activeTime.Format("2006-01-02 15:04:05")
	status.BasePrice = g.cfg.BasePrice
	status.CurrentPrice = g.px()
	status.LongStopLossPrice = g.longStopLossPrice
	status.ShortStopLossPrice = g.shortStopLossPrice
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

func (g *Grid) StatusStr() string {
	s := g.Status()
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func (g *Grid) GetTrader() common.FutureTrader {
	return g.trader
}

func (g *Grid) GetMarket() common.FutureMarket {
	return g.trader.FutureMarket()
}

func (g *Grid) onOrderDeal(deal MakerOrderDeal) {
	logger.LogInfo(g.logPrefix, "dealing: %s", deal.Deal.O.String())

	// 撤退
	if g.tryRetreat && g.long() == 0 && g.short() == 0 {
		g.switchPhase(GridPhase_Retreat)
	}

	// 保存数据
	if g.infContext != nil {
		g.infContext.AddDeal(deal.Deal, g.dealType)
	}

	// 回调外部
	if g.onDeal != nil {
		g.onDeal(deal)
	}
}

func (g *Grid) Active() {
	if g.phase == GridPhase_WaitActive {
		// 进入交易状态
		g.switchPhase(GridPhase_Dealing)
	}
}

func (g *Grid) Stop() {
	g.pm.Cancel()
}

func (g *Grid) Retreat() {
	g.switchPhase(GridPhase_Retreat)
}

func (g *Grid) TryRetreat() {
	g.tryRetreat = true

	// 立刻判断一次
	if g.tryRetreat && g.long() == 0 && g.short() == 0 {
		g.switchPhase(GridPhase_Retreat)
	}
}

func (g *Grid) Detach() {
	g.switchPhase(GridPhase_Finished)
}

func (g *Grid) Finished() bool {
	return g.phase == GridPhase_Finished
}

func (g *Grid) autoUpdate() {
	ticker := time.NewTicker(time.Millisecond * 10)
	for !g.Finished() {
		<-ticker.C
		g.Update()
	}
	g.uninit()
}

func (g *Grid) Update() {
	switch g.phase {
	case GridPhase_WaitActive:
		g.update_WaitActive()
	case GridPhase_Dealing:
		g.update_Dealing()
	case GridPhase_Retreat:
		g.update_Retreat()
	}

	g.pm.Update()
	g.frameIndex++
}

func (g *Grid) autoSaveData() {
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

func (g *Grid) update_WaitActive() {
	if time.Now().After(g.activeTime) {
		g.Active()
	}
}

func (g *Grid) update_Dealing() {
	// 适时更新计划表
	if g.cfgDirty {
		g.generatePlan()
		g.cfgDirty = false
	}

	// 根据计划表进行交易
	markPrice := g.px()
	if g.longValid() {
		if markPrice < g.longStopLossPrice {
			if g.longStopLossStartTime.IsZero() {
				g.longStopLossStartTime = time.Now()
			} else if time.Now().Unix()-g.longStopLossStartTime.Unix() > 10 {
				// 价格超标，切换到平仓状态
				g.switchPhase(GridPhase_Retreat)
				return
			}
		} else {
			if !g.longStopLossStartTime.IsZero() {
				g.longStopLossStartTime = time.Time{}
			}
		}
	}

	if g.shortValid() {
		if markPrice > g.shortStopLossPrice {
			if g.shortStopLossStartTime.IsZero() {
				g.shortStopLossStartTime = time.Now()
			} else if time.Now().Unix()-g.shortStopLossStartTime.Unix() > 10 {
				// 价格超标，切换到平仓状态
				g.switchPhase(GridPhase_Retreat)
				return
			}
		} else {
			if !g.shortStopLossStartTime.IsZero() {
				g.shortStopLossStartTime = time.Time{}
			}
		}
	}

	// 价格不达标，继续交易
	// 根据价格计算出当前应持仓位，然后挂单开平仓
	targetDir := common.OrderDir_None
	basePx := g.cfg.BasePrice
	if g.longValid() && markPrice < basePx {
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
	} else if g.shortValid() && markPrice > basePx {
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

	if g.long() > 0 && markPrice > basePx || g.short() > 0 && markPrice < basePx {
		g.pm.ModifyTargetSizeWithDealDir(decimal.Zero, common.OrderDir_None, DealDir_Close)
	}
}

func (g *Grid) update_Retreat() {
	// 结束条件
	if g.long() == 0 && g.short() == 0 {
		logger.LogInfo(g.logPrefix, "no position in retreat status? finish")
		g.switchPhase(GridPhase_Finished)
	}
}

func (g *Grid) generatePlan() {
	// 计算中点价格和最大仓位
	if g.cfg.BasePrice == 0 {
		g.cfg.BasePrice = g.GetMarket().OrderBook().MiddlePrice().InexactFloat64()
	}

	basePx := g.cfg.BasePrice

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
		g.p2pOpenBuy.SetPriceRange(basePx*(1-g.cfg.GridStep), basePx*(1-g.cfg.GridRange))
		g.p2pCloseBuy.SetPriceRange(basePx, basePx*(1-g.cfg.GridRange+g.cfg.GridStep))
		g.p2pOpenBuy.SetPositionRange(0, maxLong)
		g.p2pCloseBuy.SetPositionRange(0, maxLong)
		g.longStopLossPrice = basePx * (1 - g.cfg.GridRange - g.cfg.GridStep*float64(g.cfg.SLStep))
		g.longStopLossPrice = g.trader.Market().AlignPriceNumber(decimal.NewFromFloat(g.longStopLossPrice)).InexactFloat64()
	}

	if maxShort > 0 {
		g.p2pOpenSell.SetPriceRange(basePx*(1+g.cfg.GridStep), basePx*(1+g.cfg.GridRange))
		g.p2pCloseSell.SetPriceRange(basePx, basePx*(1+g.cfg.GridRange-g.cfg.GridStep))
		g.p2pOpenSell.SetPositionRange(0, maxShort)
		g.p2pCloseSell.SetPositionRange(0, maxShort)
		g.shortStopLossPrice = basePx * (1 + g.cfg.GridRange + g.cfg.GridStep*float64(g.cfg.SLStep))
		g.shortStopLossPrice = g.trader.Market().AlignPriceNumber(decimal.NewFromFloat(g.shortStopLossPrice)).InexactFloat64()
	}

	logger.LogImportant(g.logPrefix, "plan generated:")
	logger.LogImportant(g.logPrefix, g.StatusStr())
}

func (g *Grid) switchPhase(phase GridPhase) {
	if g.phase != phase {
		// 状态切换逻辑
		if phase == GridPhase_Retreat {
			g.pm.ModifyTargetSize(decimal.Zero, common.OrderDir_None)
		}

		logger.LogInfo(
			g.logPrefix,
			"phase switching from %s to %s",
			GridPhase2Str(g.phase),
			GridPhase2Str(phase))
		g.phase = phase
	}
}

// #region 内部函数
func (d *Grid) px() float64 {
	return d.trader.FutureMarket().MarkPrice().InexactFloat64()
}

func (d *Grid) long() float64 {
	return d.trader.Position().Long().InexactFloat64()
}

func (d *Grid) short() float64 {
	return d.trader.Position().Short().InexactFloat64()
}

func (d *Grid) longAvgPrice() float64 {
	return d.trader.Position().LongAvgPx().InexactFloat64()
}

func (d *Grid) shortAvgPrice() float64 {
	return d.trader.Position().ShortAvgPx().InexactFloat64()
}

func (d *Grid) longOnly() bool {
	return d.longValid() && !d.shortValid()
}

func (d *Grid) shortOnly() bool {
	return d.shortValid() && !d.longValid()
}

func (d *Grid) longValid() bool {
	return d.p2pOpenBuy.Valid() && d.p2pCloseBuy.Valid()
}

func (d *Grid) shortValid() bool {
	return d.p2pOpenSell.Valid() && d.p2pCloseSell.Valid()
}
