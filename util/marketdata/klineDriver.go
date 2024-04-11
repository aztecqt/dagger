/*
 * @Author: aztec
 * @Date: 2023-09-01 15:12:59
 * @Description: 用K线数据作为驱动来源
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package marketdata

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/aztecqt/dagger/util"
	"golang.org/x/exp/slices"
)

// k线单位
type KlineUnit struct {
	Ts         int64
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

func (k *KlineUnit) Deserialize(r io.Reader) bool {
	if e := binary.Read(r, binary.LittleEndian, &k.Ts); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &k.OpenPrice); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &k.ClosePrice); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &k.LowPrice); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &k.HighPrice); e != nil {
		return false
	}

	if e := binary.Read(r, binary.LittleEndian, &k.Volume); e != nil {
		return false
	}

	return true
}

type KLine struct {
	Symbol string
	Units  []KlineUnit
}

func LoadKLine(rootDir string, t0, t1 time.Time, symbol string) *KLine {
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1)
	kline := &KLine{Symbol: symbol}
	for d := dt0; d.Unix() <= dt1.Unix(); d = d.AddDate(0, 0, 1) {
		path := fmt.Sprintf("%s/%s/%s.1min.kline", rootDir, symbol, d.Format(time.DateOnly))
		util.FileDeserializeToObjects(
			path,
			func() *KlineUnit { return &KlineUnit{} },
			func(ku *KlineUnit) bool {
				kline.Units = append(kline.Units, *ku)
				return true
			})
	}
	return kline
}

type KLineDriver struct {
	t0, t1   time.Time
	klines   []*KLine
	progress float64
}

func (d *KLineDriver) Clone() Driver {
	clone := &KLineDriver{}
	clone.t0 = d.t0
	clone.t1 = d.t1
	clone.klines = slices.Clone(d.klines)
	clone.progress = 0
	return clone
}

func (d *KLineDriver) StartTime() time.Time {
	return d.t0
}

func (d *KLineDriver) EndTime() time.Time {
	return d.t1
}

func (d *KLineDriver) Init(rootDir string, t0, t1 time.Time, symbols ...string) bool {
	// 加载数据
	for _, symbol := range symbols {
		kline := LoadKLine(rootDir, t0, t1, symbol)

		if len(kline.Units) > 0 {
			_t0 := time.UnixMilli(kline.Units[0].Ts)
			_t1 := time.UnixMilli(kline.Units[len(kline.Units)-1].Ts)
			if _t0.After(d.t0) {
				d.t0 = _t0
			}
			if _t1.Before(d.t1) {
				d.t1 = _t1
			}
		}

		d.klines = append(d.klines, kline)
	}

	// 验证数据
	ts0 := int64(-1)
	ts1 := int64(-1)
	length := -1
	valid := true
	for i, kl := range d.klines {
		if len(kl.Units) == 0 {
			valid = false
			break
		}
		_ts0 := kl.Units[0].Ts
		_ts1 := kl.Units[len(kl.Units)-1].Ts
		_length := len(kl.Units)
		fmt.Printf("%s: %d data loaded, ts0=%d, ts1=%d\n", kl.Symbol, _length, _ts0, _ts1)

		if i == 0 {
			ts0 = kl.Units[0].Ts
			ts1 = kl.Units[len(kl.Units)-1].Ts
			length = len(kl.Units)
		} else {
			if _length == 0 {
				valid = false
			}

			if ts0 != kl.Units[0].Ts || ts1 != kl.Units[len(kl.Units)-1].Ts || length != len(kl.Units) {
				valid = false
			}
		}
	}

	return valid
}

func (d *KLineDriver) Run(fnUpdate func(now time.Time, tickers []Ticker)) {
	l := len(d.klines[0].Units)
	for i := 0; i < l; i++ {
		tickers := make([]Ticker, 0)
		d.progress = float64(i) / float64(l)
		for _, kl := range d.klines {
			ticker := Ticker{}
			ticker.Symbol = kl.Symbol
			ticker.Buy1 = kl.Units[i].OpenPrice
			ticker.Sell1 = kl.Units[i].OpenPrice
			tickers = append(tickers, ticker)
		}

		fnUpdate(time.UnixMilli(d.klines[0].Units[i].Ts), tickers)
	}

	d.progress = 1
}

func (d *KLineDriver) ShowProgress() {
	go func() {
		for d.progress < 1 {
			fmt.Printf("%.2f%%\n", d.progress*100)
			time.Sleep(time.Millisecond * 500)
		}
		fmt.Println("100.00%")
	}()
}
