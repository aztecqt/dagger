/*
 * @Author: aztec
 * @Date: 2023-08-24 11:33:53
 * @Description: 使用ticker数据作为行情驱动器
 * ticker的数据结构参考market_collector.collectorv2
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package marketdata

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/aztecqt/dagger/util"
	"golang.org/x/exp/slices"
)

type rawTicker struct {
	TimeStamp int64
	Price     float64
	Buy1      float64
	Sell1     float64
}

func (t *rawTicker) Deserialize(r io.Reader) bool {
	if r == nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.TimeStamp); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.Price); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.Buy1); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &t.Sell1); e != nil {
		return false
	}

	return true
}

type rawTickerList struct {
	symbol string
	index  int
	list   []rawTicker
}

type TickerDriver struct {
	t0, t1        time.Time
	intervalMS    int64
	allTickerList []rawTickerList
	fastMode      bool
	progress      float64
}

func (d *TickerDriver) Clone() Driver {
	clone := &TickerDriver{}
	clone.t0 = d.t0
	clone.t1 = d.t1
	clone.intervalMS = d.intervalMS
	clone.allTickerList = slices.Clone(d.allTickerList)
	for i := range clone.allTickerList {
		clone.allTickerList[i].index = 0
	}
	return clone
}

func (d *TickerDriver) StartTime() time.Time {
	return d.t0
}

func (d *TickerDriver) EndTime() time.Time {
	return d.t1
}

func (d *TickerDriver) Init(fastMode bool, rootDir string, t0, t1 time.Time, intervalMS int64, symbols ...string) {
	// 加载数据
	d.fastMode = fastMode
	d.allTickerList = make([]rawTickerList, 0)
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1)
	d.t0 = t0
	d.t1 = t1
	d.intervalMS = intervalMS

	// 加载指定时间区间内的所有ticker，并组织整齐
	for _, symbol := range symbols {
		tickers := make([]rawTicker, 0)
		for d := dt0; d.Unix() <= dt1.Unix(); d = d.AddDate(0, 0, 1) {
			path := fmt.Sprintf("%s/%s/%s.ticker", rootDir, symbol, d.Format(time.DateOnly))
			pathZipped := path + ".zlib"
			var file io.ReadCloser

			if f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm); err == nil {
				file = f
				defer file.Close()
				fmt.Printf("ticker driver loaded %s\n", path)
			} else {
				if f, err := util.OpenCompressedFile_Zlib(pathZipped); err == nil {
					file = f
					defer file.Close()
					fmt.Printf("ticker driver loaded %s\n", pathZipped)
				}
			}

			lastTimeStamp := int64(0)
			lastBuy := -1.0
			lastSell := -1.0
			util.DeserializeToObjects(
				file,
				func() *rawTicker { return &rawTicker{} },
				func(t *rawTicker) bool {
					timeok := false
					if fastMode {
						timeok = t.TimeStamp-lastTimeStamp >= intervalMS && t.TimeStamp >= t0.UnixMilli() && t.TimeStamp < t1.UnixMilli()
					} else {
						timeok = t.TimeStamp > lastTimeStamp && t.TimeStamp >= t0.UnixMilli() && t.TimeStamp < t1.UnixMilli()
					}
					if (t.Buy1 != lastBuy || t.Sell1 != lastSell) && timeok {
						tickers = append(tickers, *t)
						lastTimeStamp = t.TimeStamp
						lastBuy = t.Buy1
						lastSell = t.Sell1
					}
					return true
				})
		}

		if len(tickers) > 0 {
			_t0 := time.UnixMilli(tickers[0].TimeStamp)
			_t1 := time.UnixMilli(tickers[len(tickers)-1].TimeStamp)
			if _t0.After(d.t0) {
				d.t0 = _t0
			}
			if _t1.Before(d.t1) {
				d.t1 = _t1
			}
		}

		d.allTickerList = append(d.allTickerList, rawTickerList{symbol: symbol, list: tickers, index: 0})
	}
}

func (d *TickerDriver) Run(fnUpdate func(now time.Time, tickers []Ticker)) {
	if d.fastMode {
		d.run_fast(fnUpdate)
	} else {
		d.run_slow(fnUpdate)
	}
}

func (d *TickerDriver) run_slow(fnUpdate func(now time.Time, tickers []Ticker)) {
	// 先初始化Ticker数组，symbol填好，价格按原始数据中的第一个元素。
	// 然后找出最早的那个时间作为起始时间。
	tickers := make([]Ticker, 0)
	ms := int64(math.MaxInt64)
	for _, tl := range d.allTickerList {
		if len(tl.list) == 0 {
			fmt.Printf("no data for symbol %s\n", tl.symbol)
			return
		}

		if tl.list[0].TimeStamp < ms {
			ms = tl.list[0].TimeStamp
		}
		tickers = append(tickers, Ticker{Symbol: tl.symbol, Buy1: tl.list[0].Buy1, Sell1: tl.list[0].Sell1})
	}

	// 首次更新
	fnUpdate(time.UnixMilli(ms), tickers)

	// 从起始时间开始，向后遍历整个时间段
	// 当interval范围内存在tick数据，则用最新的ticker时间进行更新。否则推进一个maxInterval
	for ms < d.t1.UnixMilli() {
		d.progress = float64(ms-d.t0.UnixMilli()) / float64(d.t1.UnixMilli()-d.t0.UnixMilli())

		// 找出最近的那个ticker
		minTs := int64(math.MaxInt64)
		minTsTickerIndex := -1
		allFinished := true
		for i, rtl := range d.allTickerList {
			if rtl.index < len(rtl.list)-1 {
				nextIndex := rtl.index + 1
				if rtl.list[nextIndex].TimeStamp < minTs {
					minTs = rtl.list[nextIndex].TimeStamp
					minTsTickerIndex = i
				}
			}

			if rtl.index < len(rtl.list)-1 {
				allFinished = false
			}
		}

		// 如果最近的ticker在一个MaxInterval之内，则更新到这个Ticker，否则仅推进一个maxInterval。
		if minTsTickerIndex >= 0 && minTs <= ms+d.intervalMS {
			tl := &d.allTickerList[minTsTickerIndex]
			tk := &tickers[minTsTickerIndex]
			tl.index++
			tk.Buy1 = tl.list[tl.index].Buy1
			tk.Sell1 = tl.list[tl.index].Sell1
			ms = minTs
		} else {
			ms += d.intervalMS
		}

		// 本帧输出
		fnUpdate(time.UnixMilli(ms), tickers)

		if allFinished {
			break
		}
	}

	d.progress = 1
}

func (d *TickerDriver) run_fast(fnUpdate func(now time.Time, tickers []Ticker)) {
	// 先初始化Ticker数组，symbol填好，价格按原始数据中的第一个元素。
	// 然后找出最早的那个时间作为起始时间。
	tickers := make([]Ticker, 0)
	ms := int64(math.MaxInt64)
	for _, tl := range d.allTickerList {
		if len(tl.list) == 0 {
			fmt.Printf("no data for symbol %s\n", tl.symbol)
			return
		}

		if tl.list[0].TimeStamp < ms {
			ms = tl.list[0].TimeStamp
		}
		tickers = append(tickers, Ticker{Symbol: tl.symbol, Buy1: tl.list[0].Buy1, Sell1: tl.list[0].Sell1})
	}

	// 首次更新
	fnUpdate(time.UnixMilli(ms), tickers)

	// 从起始时间开始，向后遍历整个时间段
	// 当interval范围内存在tick数据，则用最新的ticker时间进行更新。否则推进一个maxInterval
	for ms < d.t1.UnixMilli() {
		ms += d.intervalMS
		d.progress = float64(ms-d.t0.UnixMilli()) / float64(d.t1.UnixMilli()-d.t0.UnixMilli())

		// 对于每一组ticker序列，由当前位置向下寻找。当下一个ticker的时间超过ms时停止寻找
		allFinished := true
		for i, tl := range d.allTickerList {
			for index := tl.index; index < len(tl.list)-1; index++ {
				if tl.list[index+1].TimeStamp > ms || index == len(tl.list)-2 {
					tickers[i].Buy1 = tl.list[index].Buy1
					tickers[i].Sell1 = tl.list[index].Sell1
					d.allTickerList[i].index = index
					break
				}
			}

			if tl.index < len(tl.list)-1 {
				allFinished = false
			}
		}

		// 本帧输出
		fnUpdate(time.UnixMilli(ms), tickers)

		if allFinished {
			break
		}
	}

	d.progress = 1
}

func (d *TickerDriver) ShowProgress() {
	go func() {
		for d.progress < 1 {
			fmt.Printf("%.2f%%\n", d.progress*100)
			time.Sleep(time.Millisecond * 500)
		}
		fmt.Println("100.00%")
	}()
}
