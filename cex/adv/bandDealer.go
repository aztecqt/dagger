/*
 * @Author: aztec
 * @Date: 2022-12-28
 * @Description:
 * 布林带交易策略，超出布林带上轨逐步做多，回归后平仓
 * 震荡市使用
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
type BBandDealerConfig struct {
	// 交易参数
	MaxPos          int `json:"max_pos"`            // 最大持仓数量（以usd计算)
	MaxOpenTimeSec  int `json:"max_open_time_sec"`  // 最大开仓时间（秒）
	MaxCloseTimeSec int `json:"max_close_time_sec"` // 最大平仓时间（秒）

	// BBand参数
	BBand_NMA  int     `json:"bband_nma"`
	BBand_NStd float64 `json:"bband_nstd"`

	// MmwBand参数
	MmwBand_N  int `json:"mmwband_n"`
	MmwBand_N1 int `json:"mmwband_n1"`

	//（1，1）表示上轨开下轨平
	//（1，0）表示上轨开中轨平
	//（1.2，0.5）表示上轨外20%开，中轨偏下轨50%平。
	//（1，-0.8）表示上轨开，中轨偏上轨80%平
	OpenPitch  float64 `json:"open_pitch"`  // 开仓偏移。正浮点数。
	ClosePitch float64 `json:"close_pitch"` // 平仓偏移。正浮点数。
}

func (c *BBandDealerConfig) String() string {
	strband := "invalid"
	if c.BBand_NMA > 0 && c.BBand_NStd > 0 {
		strband = fmt.Sprintf("boll(ma=%d, nstd=%f)", c.BBand_NMA, c.BBand_NStd)
	} else if c.MmwBand_N > 0 && c.MmwBand_N1 > 0 {
		strband = fmt.Sprintf("mmw(n=%d, n1=%d)", c.MmwBand_N, c.MmwBand_N1)
	}

	return fmt.Sprintf("maxpos:%d, maxOpenTime:%ds, maxCloseTime:%ds, band=[%s], openPitch:%.2f%%, closePitch%.2f%%",
		c.MaxPos,
		c.MaxOpenTimeSec,
		c.MaxCloseTimeSec,
		strband,
		c.OpenPitch*100,
		c.ClosePitch*100)
}

// 状态
type BBandDealerStatus struct {
	InstID      string            `json:"inst_id"`
	LongPos     float64           `json:"long_pos"`
	ShortPos    float64           `json:"short_pos"`
	MaxLongPos  float64           `json:"max_long_pos"`
	MaxShortPos float64           `json:"max_short_pos"`
	BollUpper   float64           `json:"boll_u"`
	BollMiddle  float64           `json:"boll_m"`
	BollLower   float64           `json:"boll_d"`
	Config      BBandDealerConfig `json:"config"`
}

// 交易器对象
type BBandDealer struct {
	logPrefix  string
	cfg        BBandDealerConfig
	dealType   string
	trader     common.FutureTrader
	infContext *datamanager.InfluxContext // 数据存储

	// 指标计算
	dlPrice       stratergy.DataLine
	band          indacators.Band
	needRebuild   bool
	needRefreshDB bool
	finished      bool
}

func (b *BBandDealer) Init(
	trader common.FutureTrader,
	autoUpdate bool,
	dealType string) {
	b.trader = trader
	b.dealType = dealType
	b.infContext = datamanager.NewInfluxContext("bband", trader.Market().Type())
	b.dlPrice.Init("price", 86400, 1000, 0)

	// 加载历史价格数据
	t1 := time.Now()
	t0 := t1.Add(-time.Hour * 12)
	datamanager.GetHistoryFuturePrice(
		trader.FutureMarket().Symbol(),
		trader.FutureMarket().ContractType(),
		t0, t1, &b.dlPrice)

	if autoUpdate {
		go b.autoUpdate()
	}

	go b.autoSaveData()
}

func (b *BBandDealer) uninit() {

}

func (b *BBandDealer) Retreat() {
	panic("no impl")
}

func (b *BBandDealer) Detach() {
	b.finished = true
}

func (b *BBandDealer) Status() BBandDealerStatus {
	st := BBandDealerStatus{}
	st.InstID = b.trader.Market().Type()
	st.LongPos = b.long()
	st.ShortPos = b.short()
	st.MaxLongPos = b.maxLong()
	st.MaxShortPos = b.maxShort()
	st.BollMiddle, _ = b.band.Middle().LastValue()
	st.BollUpper, _ = b.band.Upper().LastValue()
	st.BollLower, _ = b.band.Lower().LastValue()
	st.Config = b.cfg
	return st
}

func (b *BBandDealer) StatusStr() string {
	d, _ := json.Marshal(b.Status())
	return string(d)
}

func (b *BBandDealer) GetTrader() common.FutureTrader {
	return b.trader
}

func (b *BBandDealer) SetConfig(cfg BBandDealerConfig, refreshDB bool) {
	b.cfg = cfg

	if cfg.BBand_NMA > 0 && cfg.BBand_NStd > 0 {
		b.band = indacators.NewBBand(&b.dlPrice, cfg.BBand_NMA, cfg.BBand_NStd)
	} else if cfg.MmwBand_N > 0 && cfg.MmwBand_N1 > 0 {
		b.band = indacators.NewMmwBand(&b.dlPrice, cfg.MmwBand_N, cfg.MmwBand_N1)
	} else {
		logger.LogPanic(b.logPrefix, "invalid band param")
	}

	b.needRebuild = true
	b.needRefreshDB = refreshDB
}

// 根据价格，重新构建布林带数据
func (b *BBandDealer) rebuildLines(refreshDB bool) {
	// 重建数据线条
	b.band.Rebuild()

	if refreshDB {
		// 清空数据库
		b.infContext.ClearDataPoints()

		// 重新保存数据
		for i := 0; i < b.dlPrice.Length(); i++ {
			b.savePoints(i)
		}
	}
}

func (b *BBandDealer) savePoints(index int) {
	points := make(map[string]float64)
	px, _ := b.dlPrice.GetData(index)
	middle, _ := b.band.Middle().GetData(index)
	upper, _ := b.band.Upper().GetData(index)
	lower, _ := b.band.Lower().GetData(index)
	points["price"] = px.V
	points["middle"] = middle.V
	points["upper"] = upper.V
	points["lower"] = lower.V
	points["long"] = b.long()
	points["short"] = b.short()

	t := time.Now()
	if index != b.dlPrice.Length()-1 {
		t = time.UnixMilli(px.MS)
	}

	b.infContext.AddDataPoints(points, t)
}

func (b *BBandDealer) autoUpdate() {
	ticker := time.NewTicker(time.Millisecond * 100)
	for !b.Finished() {
		<-ticker.C
		b.Update()
	}
	b.uninit()
}

func (b *BBandDealer) Update() {
	if b.needRebuild {
		b.rebuildLines(b.needRefreshDB)
		b.needRebuild = false
		logger.LogImportant(b.logPrefix, "rebuilded")
	}

	// 更新数据线
	b.dlPrice.Update(time.Now().UnixMilli(), b.middlePx())
	b.band.Update()
}

func (b *BBandDealer) autoSaveData() {
	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C

		b.savePoints(b.dlPrice.Length() - 1)

		if b.Finished() {
			break
		}
	}
}

func (b *BBandDealer) Finished() bool {
	return b.finished
}

// 内部函数
func (b *BBandDealer) middlePx() float64 {
	return b.trader.Market().OrderBook().MiddlePrice().InexactFloat64()
}

func (b *BBandDealer) long() float64 {
	return b.trader.Position().Long().InexactFloat64()
}

func (b *BBandDealer) short() float64 {
	return b.trader.Position().Short().InexactFloat64()
}

func (b *BBandDealer) maxLong() float64 {
	return common.USDT2ContractAmountAtLeast1(
		decimal.NewFromInt32(int32(b.cfg.MaxPos)), b.trader.FutureMarket()).InexactFloat64()
}

func (b *BBandDealer) maxShort() float64 {
	return common.USDT2ContractAmountAtLeast1(
		decimal.NewFromInt32(int32(b.cfg.MaxPos)), b.trader.FutureMarket()).InexactFloat64()
}
