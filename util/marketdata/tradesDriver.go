/*
 * @Author: aztec
 * @Date: 2023-09-04 08:36:40
 * @Description: 使用市场成交数据作为驱动源
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package marketdata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/aztecqt/dagger/util"
	"golang.org/x/exp/slices"
)

type marketTrade struct {
	Price     float64
	Quantity  float64
	TimeStamp int64
	IsSell    bool
}

func (t *marketTrade) Deserialize(r io.Reader) bool {
	if e := binary.Read(r, binary.LittleEndian, &t.Price); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.Quantity); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.TimeStamp); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.IsSell); e != nil {
		return false
	}

	return true
}

type marketTradeList struct {
	symbol string
	index  int
	list   []marketTrade
}

type TradeDriver struct {
	t0, t1              time.Time
	maxIntervalMS       int64
	allMarketTradeLists []marketTradeList
	progress            float64
}

func (d *TradeDriver) Clone() Driver {
	clone := &TradeDriver{}
	clone.t0 = d.t0
	clone.t1 = d.t1
	clone.maxIntervalMS = d.maxIntervalMS
	clone.allMarketTradeLists = slices.Clone(d.allMarketTradeLists)
	clone.progress = d.progress
	for i := range clone.allMarketTradeLists {
		clone.allMarketTradeLists[i].index = 0
	}
	return clone
}

func (d *TradeDriver) StartTime() time.Time {
	return d.t0
}

func (d *TradeDriver) EndTime() time.Time {
	return d.t1
}

func (d *TradeDriver) Init(rootDir string, t0, t1 time.Time, maxIntervalMS int64, symbols ...string) {
	// 加载数据
	d.allMarketTradeLists = make([]marketTradeList, 0)
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1)
	d.t0 = t0
	d.t1 = t1
	d.maxIntervalMS = maxIntervalMS

	// 加载指定时间区间内的所有trades，并组织整齐
	for _, symbol := range symbols {
		trades := make([]marketTrade, 0)
		for d := dt0; d.Unix() <= dt1.Unix(); d = d.AddDate(0, 0, 1) {
			path := fmt.Sprintf("%s/%s/%s.trades", rootDir, symbol, d.Format(time.DateOnly))
			util.FileDeserializeToObjects(
				path,
				func() *marketTrade { return &marketTrade{} },
				func(t *marketTrade) bool {
					trades = append(trades, *t)
					return true
				})
		}

		if len(trades) > 0 {
			_t0 := time.UnixMilli(trades[0].TimeStamp)
			_t1 := time.UnixMilli(trades[len(trades)-1].TimeStamp)
			if _t0.After(d.t0) {
				d.t0 = _t0
			}
			if _t1.Before(d.t1) {
				d.t1 = _t1
			}
		}

		d.allMarketTradeLists = append(d.allMarketTradeLists, marketTradeList{symbol: symbol, index: 0, list: trades})
	}
}

func (d *TradeDriver) Run(fnUpdate func(now time.Time, tickers []Ticker)) {
	// 先初始化Ticker数组，symbol填好，价格按原始数据中的第一个元素。
	// 然后找出最早的那个时间作为起始时间。
	tickers := make([]Ticker, 0)
	ms := int64(math.MaxInt64)
	for _, tl := range d.allMarketTradeLists {
		if len(tl.list) == 0 {
			fmt.Printf("no data for symbol %s\n", tl.symbol)
			return
		}

		// 找出起始时间ms
		if tl.list[0].TimeStamp < ms {
			ms = tl.list[0].TimeStamp
		}

		// 每个币种行情的初始值
		tickers = append(tickers, Ticker{Symbol: tl.symbol, Buy1: tl.list[0].Price, Sell1: tl.list[0].Price})
	}

	// 首次更新
	fnUpdate(time.UnixMilli(ms), tickers)

	// 从起始时间开始，向后遍历整个时间段
	// 当interval范围内存在tick数据，则用最新的ticker时间进行更新。否则推进一个maxInterval
	for ms < d.t1.UnixMilli() {
		/*ms += d.maxIntervalMS
		d.progress = float64(ms-d.t0.UnixMilli()) / float64(d.t1.UnixMilli()-d.t0.UnixMilli())

		// 对于每一组交易序列，由当前位置向下寻找。当下一个trade的时间超过ms时停止寻找
		for i, mtl := range d.allMarketTradeLists {
			for index := mtl.index; index < len(mtl.list)-1; index++ {
				if mtl.list[index+1].TimeStamp > ms || index == len(mtl.list)-2 {
					tickers[i].Buy1 = mtl.list[index].Price
					tickers[i].Sell1 = mtl.list[index].Price
					d.allMarketTradeLists[i].index = index
					break
				}
			}
		}*/

		// 找出最近的那个ticker
		minTs := int64(math.MaxInt64)
		minTsTickerIndex := -1
		for i, rtl := range d.allMarketTradeLists {
			if rtl.index < len(rtl.list)-1 {
				nextIndex := rtl.index + 1
				if rtl.list[nextIndex].TimeStamp < minTs {
					minTs = rtl.list[nextIndex].TimeStamp
					minTsTickerIndex = i
				}
			}
		}

		// 如果最近的ticker在一个MaxInterval之内，则更新到这个Ticker，否则仅推进一个maxInterval。
		if minTsTickerIndex >= 0 && minTs <= ms+d.maxIntervalMS {
			tl := &d.allMarketTradeLists[minTsTickerIndex]
			tk := &tickers[minTsTickerIndex]
			tl.index++
			tk.Buy1 = tl.list[tl.index].Price
			tk.Sell1 = tl.list[tl.index].Price
			ms = minTs
		} else {
			ms += d.maxIntervalMS
		}

		// 本帧输出
		fnUpdate(time.UnixMilli(ms), tickers)
	}

	d.progress = 1
}

func (d *TradeDriver) ShowProgress() {
	go func() {
		for d.progress < 1 {
			fmt.Printf("%.2f%%\n", d.progress*100)
			time.Sleep(time.Millisecond * 500)
		}
		fmt.Println("100.00%")
	}()
}
