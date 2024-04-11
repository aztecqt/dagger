/*
 * @Author: aztec
 * @Date: 2022-12-31 12:02:28
 * @Description: 按时间开仓。ma平仓
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

	"github.com/aztecqt/dagger/util/indacators"
	"github.com/shopspring/decimal"
)

// 参数
type TWDealerConfig struct {
	OpenTimeLength int     `json:"open_time_len"` // 开仓时长
	MaxSizeUsd     int     `json:"max_size_usd"`  // 最大持仓量
	MA             int     `json:"ma"`            // 平仓ma
	SafeGap        float64 `json:"safe_gap"`      // 安全区。价格距离ma多远就不再开仓了
}

func (c *TWDealerConfig) String() string {
	return fmt.Sprintf("max_size: %dusd, max_time:%dsec, ma:%d", c.MaxSizeUsd, c.OpenTimeLength, c.MA)
}

// 状态
type TWDealerStatus struct {
	InstID    string         `json:"inst_id"`
	LongPos   float64        `json:"long_pos"`
	ShortPos  float64        `json:"short_pos"`
	MaxPos    float64        `json:"max_pos"`
	TargetPos float64        `json:"target_pos"`
	Price     float64        `json:"price"`
	MAPrice   float64        `json:"ma_price"`
	PosPrice  float64        `json:"pos_price"`
	Config    TWDealerConfig `json:"config"`
}

// 交易器
type TWDealer struct {
	logPrefix  string
	cfg        TWDealerConfig
	dealType   string
	trader     common.FutureTrader
	exchange   common.CEx
	infContext *datamanager.InfluxContext

	// 开仓逻辑
	thisUpdateTime time.Time
	lastUpdateTime time.Time
	dir            common.OrderDir // 交易方向
	needOpen       float64         // 需要开这么多（随时间增长）
	alreadyOpen    float64         // 已经开了这么多（随成交增长）
	mkOpen         *Maker          // 开仓订单
	mkClose        *Maker          // 平仓订单
	retreating     bool
	finished       bool

	// 指标计算
	dlPrice       stratergy.DataLine
	ma            *indacators.SMA
	needRebuild   bool
	needRefreshDB bool
}

func (b *TWDealer) Init(
	exchange common.CEx,
	trader common.FutureTrader,
	autoUpdate bool,
	dealType string) {
	b.exchange = exchange
	b.trader = trader
	b.dealType = dealType
	b.infContext = datamanager.NewInfluxContext("twdealer", trader.Market().Type())
	b.dlPrice.Init("price", 86400, 1000, 0)
	b.logPrefix = fmt.Sprintf("twdealer-%s", b.trader.Market().Type())
	b.dir = common.OrderDir_None
	b.mkOpen = new(Maker)
	b.mkClose = new(Maker)
	b.mkOpen.Init(trader, true, false, true, 0, 0, "open")
	b.mkClose.Init(trader, true, false, true, 0, 0, "close")
	b.mkOpen.SetDealFn(b.onOpenDeal)
	b.mkClose.SetDealFn(b.onCloseDeal)

	// 加载历史价格数据
	t1 := time.Now()
	t0 := t1.Add(-time.Hour * 6)
	symbol := b.trader.FutureMarket().Symbol()
	cttype := b.trader.FutureMarket().ContractType()
	kunis := b.exchange.GetFutureKline(symbol, cttype, t0, t1, 60)
	for _, ku := range kunis {
		b.dlPrice.UpdateDecimal(ku.Time.UnixMilli(), ku.ClosePrice)
	}

	if b.dlPrice.Length() == 0 {
		logger.LogPanic(b.logPrefix, "load history price failed!")
	}

	if autoUpdate {
		go b.autoUpdate()
	}

	go b.autoSaveData()
}

func (b *TWDealer) uninit() {
	b.mkOpen.Stop()
	b.mkClose.Stop()
}

func (b *TWDealer) Retreat() {
	logger.LogImportant(b.logPrefix, "retreating")
	b.mkOpen.Cancel()
	b.retreating = true
}

func (b *TWDealer) Detach() {
	logger.LogImportant(b.logPrefix, "detaching")
	b.mkOpen.Cancel()
	b.finished = true
}

func (b *TWDealer) Status() TWDealerStatus {
	st := TWDealerStatus{}
	st.InstID = b.trader.Market().Type()
	st.LongPos = b.long()
	st.ShortPos = b.short()
	st.MaxPos = b.maxPos()
	st.TargetPos = b.needOpen
	st.Price = b.markPrice()
	st.MAPrice, _ = b.ma.Value().LastValue()
	st.PosPrice = b.posPrice()
	st.Config = b.cfg
	return st
}

func (b *TWDealer) StatusStr() string {
	d, _ := json.Marshal(b.Status())
	return string(d)
}

func (b *TWDealer) GetTrader() common.FutureTrader {
	return b.trader
}

func (b *TWDealer) SetConfig(cfg TWDealerConfig, refreshDB bool) {
	b.cfg = cfg
	b.ma = indacators.NewSMA(&b.dlPrice, b.cfg.MA)
	b.needRebuild = true
	b.needRefreshDB = refreshDB
}

// 根据价格，重新构建布林带数据
func (b *TWDealer) rebuildLines(refreshDB bool) {
	// 重建数据线条
	b.ma.Rebuild()

	if refreshDB {
		// 清空数据库
		b.infContext.ClearDataPoints()

		// 重新保存数据
		for i := 0; i < b.dlPrice.Length(); i++ {
			b.savePoints(i)
		}
	}
}

func (b *TWDealer) savePoints(index int) {
	if b.ma != nil {
		points := make(map[string]float64)
		px, _ := b.dlPrice.GetData(index)
		ma, _ := b.ma.Value().GetData(index)
		points["price"] = px.V
		points["ma"] = ma.V
		points["long"] = b.long()
		points["short"] = b.short()

		t := time.Now()
		if index != b.dlPrice.Length()-1 {
			t = time.UnixMilli(px.MS)
		}

		b.infContext.AddDataPoints(points, t)
	}
}

func (b *TWDealer) autoUpdate() {
	ticker := time.NewTicker(time.Millisecond * 100)
	for !b.Finished() {
		<-ticker.C
		b.Update()
	}
}

func (b *TWDealer) Update() {
	b.thisUpdateTime = time.Now()

	if b.needRebuild {
		b.rebuildLines(b.needRefreshDB)
		b.needRebuild = false
		logger.LogImportant(b.logPrefix, "rebuilded")
	}

	// 更新数据线
	b.dlPrice.Update(time.Now().UnixMilli(), b.lastPrice())
	b.ma.Update()

	b.updateDeal_Open()
	b.updateDeal_Close()

	b.lastUpdateTime = b.thisUpdateTime
}

// 开仓逻辑
func (b *TWDealer) updateDeal_Open() {
	if b.retreating {
		return
	}

	if b.lastUpdateTime.IsZero() {
		return
	}

	if ma, ok := b.ma.Value().LastValue(); ok {
		if b.dir == common.OrderDir_None {
			if b.lastPrice() > ma {
				b.dir = common.OrderDir_Sell
			} else {
				b.dir = common.OrderDir_Buy
			}
		}

		// 计算需要开仓的大小
		deltaSec := float64(b.thisUpdateTime.UnixNano()-b.lastUpdateTime.UnixNano()) * time.Nanosecond.Seconds()
		posPerSec := b.maxPos() / float64(b.cfg.OpenTimeLength)
		b.needOpen += posPerSec * deltaSec

		// 下单大小
		sz := 0.0
		px := ma
		if b.dir == common.OrderDir_Sell {
			sz = b.needOpen - b.short()
			px = b.buy1() * 0.99
		} else if b.dir == common.OrderDir_Buy {
			sz = b.needOpen - b.long()
			px = b.sell1() * 1.01
		}

		// safe gap
		if b.dir == common.OrderDir_Buy {
			gap := (ma - b.buy1()) / ma
			if gap < b.cfg.SafeGap {
				sz = 0
			}
		} else if b.dir == common.OrderDir_Sell {
			gap := (b.sell1() - ma) / ma
			if gap < b.cfg.SafeGap {
				sz = 0
			}
		}

		// 保险
		if b.short() > b.maxPos() || b.long() >= b.maxPos() {
			sz = 0
		}

		b.mkOpen.Modify(decimal.NewFromFloat(px), decimal.NewFromFloat(sz), b.dir, false)
	}
}

// 平仓逻辑
// 只要有仓位就在ma上平仓
func (b *TWDealer) updateDeal_Close() {
	if b.retreating {
		b.mkOpen.Cancel()
		if b.long() > 0 {
			px := b.buy1() * 0.99
			sz := b.long()
			b.mkClose.Modify(
				decimal.NewFromFloat(px),
				decimal.NewFromFloat(sz),
				common.OrderDir_Sell, true)
		} else if b.short() > 0 {
			px := b.sell1() * 1.01
			sz := b.short()
			b.mkClose.Modify(
				decimal.NewFromFloat(px),
				decimal.NewFromFloat(sz),
				common.OrderDir_Buy, true)
		}
	} else {
		ma, maok := b.ma.Value().LastValue()
		if maok {
			px := ma
			if b.long() > 0 {
				sz := b.long()
				b.mkClose.Modify(
					decimal.NewFromFloat(px),
					decimal.NewFromFloat(sz),
					common.OrderDir_Sell, true)
			} else if b.short() > 0 {
				sz := b.short()
				b.mkClose.Modify(
					decimal.NewFromFloat(px),
					decimal.NewFromFloat(sz),
					common.OrderDir_Buy,
					true)
			}
		} else {
			logger.LogImportant(b.logPrefix, "get ma failed?")
		}
	}

}

func (b *TWDealer) onOpenDeal(deal MakerOrderDeal) {
	b.infContext.AddDeal(deal.Deal, b.dealType)
	b.alreadyOpen += deal.Deal.Amount.InexactFloat64()
}

func (b *TWDealer) onCloseDeal(deal MakerOrderDeal) {
	b.infContext.AddDeal(deal.Deal, b.dealType)
	b.Retreat() // 平仓单一旦成交，直接进入平仓模式不再开仓
	if b.long() == 0 && b.short() == 0 {
		b.finished = true
		logger.LogImportant(b.logPrefix, "position cleared, finish")
	}
}

func (b *TWDealer) autoSaveData() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C

		b.savePoints(b.dlPrice.Length() - 1)

		if b.Finished() {
			break
		}
	}
}

func (b *TWDealer) Finished() bool {
	return b.finished
}

// 内部函数
func (b *TWDealer) lastPrice() float64 {
	return b.trader.Market().LatestPrice().InexactFloat64()
}

func (b *TWDealer) markPrice() float64 {
	return b.trader.FutureMarket().MarkPrice().InexactFloat64()
}

func (b *TWDealer) buy1() float64 {
	return b.trader.Market().OrderBook().Buy1Price().InexactFloat64()
}

func (b *TWDealer) sell1() float64 {
	return b.trader.Market().OrderBook().Sell1Price().InexactFloat64()
}

func (b *TWDealer) posPrice() float64 {
	if b.trader.Position().Long().IsPositive() {
		return b.trader.Position().LongAvgPx().InexactFloat64()
	} else {
		return b.trader.Position().ShortAvgPx().InexactFloat64()
	}
}

func (b *TWDealer) long() float64 {
	return b.trader.Position().Long().InexactFloat64()
}

func (b *TWDealer) short() float64 {
	return b.trader.Position().Short().InexactFloat64()
}

func (b *TWDealer) maxPos() float64 {
	return common.USDT2ContractAmountAtLeast1(
		decimal.NewFromInt32(int32(b.cfg.MaxSizeUsd)), b.trader.FutureMarket()).InexactFloat64()
}
