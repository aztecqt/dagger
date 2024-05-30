/*
 * @Author: aztec
 * @Date: 2022-05-24 15:31:00
  - @LastEditors: Please set LastEditors
 * @FilePath: \dagger\cex\okexv5\fundingfee_observer.go
 * @Description: okx的费率观察器。实现common.FundingFeeObserver接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5

import (
	"sync"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type FundingFeeObserver struct {
	logPrefix  string
	muMain     sync.Mutex
	muRestFreq sync.Mutex

	ex            *Exchange
	usdtSwap      bool
	spotPrices    map[string]decimal.Decimal
	fundingFees   map[string]common.FundingFeeInfo
	mainReady     bool
	historyReady  bool
	progressTotal float64
	progress      float64
}

func (f *FundingFeeObserver) init(ex *Exchange) {
	f.ex = ex
	f.usdtSwap = f.ex.excfg.FundingFeeObserver.UsdtSwap
	f.logPrefix = "funding-fee-observer"
	f.fundingFees = make(map[string]common.FundingFeeInfo)
	f.spotPrices = make(map[string]decimal.Decimal)
	ex.registerTickerCallbackOfInstType("SWAP", f.onSwapTickers)
	ex.registerTickerCallbackOfInstType("SPOT", f.onSpotTickers)

	// 先等待fundingfees队列构建完毕(由ticker创建)
	for {
		if len(f.fundingFees) > 0 {
			break
		} else {
			time.Sleep(time.Millisecond * 100)
		}
	}

	f.progressTotal = float64(len(f.fundingFees) * 3)
	go f.updateMain()
	go f.updateHistory()
}

func (f *FundingFeeObserver) onSwapTickers(tks []okexv5api.TickerResp) {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	validInstIds := map[string]int{}
	for _, tr := range tks {
		validInstIds[tr.InstId] = 1

		// 找到对应现货价格
		spotInstId := FutureInstId2SpotInstId(tr.InstId)
		if spotPx, ok := f.spotPrices[spotInstId]; ok {
			if _, ok := f.fundingFees[tr.InstId]; !ok {
				// 创建新的fundingfee对象
				f.fundingFees[tr.InstId] = common.FundingFeeInfo{}
			}

			// 刷新部分信息
			ffi := f.fundingFees[tr.InstId]
			ffi.InstId = tr.InstId
			ffi.SwapPrice = tr.Buy1.Add(tr.Sell1).Div(util.DecimalTwo)
			ffi.SpotPrice = spotPx
			ffi.VolUSD24h = tr.VolCcy24h
			f.fundingFees[tr.InstId] = ffi
		}
	}

	// 删除无效数据
	for k := range f.fundingFees {
		if _, ok := validInstIds[k]; !ok {
			delete(f.fundingFees, k)
		}
	}
}

func (f *FundingFeeObserver) onSpotTickers(tks []okexv5api.TickerResp) {
	f.muMain.Lock()
	defer f.muMain.Unlock()
	for _, tr := range tks {
		f.spotPrices[tr.InstId] = tr.Buy1.Add(tr.Sell1).Div(util.DecimalTwo)
	}
}

func (f *FundingFeeObserver) updateMain() {
	// 主循环：以每秒1个的速度，刷新所有品种的当前费率
	for {
		instIds := make([]string, 0, len(f.fundingFees))
		f.muMain.Lock()
		for k := range f.fundingFees {
			instIds = append(instIds, k)
		}
		f.muMain.Unlock()

		for i := 0; i < len(instIds); i++ {
			instId := instIds[i]
			resp, err := okexv5api.GetFundingRate(instId)
			if okexv5api.CheckRestResp(resp.CommonRestResp, err, "get funding fee of "+instId, f.logPrefix) && len(resp.Data) > 0 {
				fr := resp.Data[0]
				f.muMain.Lock()
				if v, ok := f.fundingFees[instId]; ok {
					v.FeeRate = fr.FundingRate
					v.FeeTime = fr.FundingTime
					v.NextFeeRate = fr.NextFundingRate
					v.NextFeeTime = fr.NextFundingTime
					f.fundingFees[instId] = v
				}
				f.muMain.Unlock()
			}

			if f.mainReady {
				time.Sleep(time.Second)
			} else {
				f.progress += 1
				time.Sleep(time.Millisecond * 100)
			}
		}
		f.mainReady = true
	}
}

func (f *FundingFeeObserver) updateHistory() {
	// 次要循环：每个整点启动一次历史费率刷新，1秒1次刷新所有品种的历史费率
	lastTime := time.Time{}

	for {
		now := time.Now()
		if now.Hour() != lastTime.Hour() || lastTime.IsZero() {
			lastTime = now
			if f.historyReady {
				time.Sleep(time.Minute)
			}

			// 整点，启动历史费率刷新
			instIds := make([]string, 0, len(f.fundingFees))
			f.muMain.Lock()
			for k := range f.fundingFees {
				instIds = append(instIds, k)
			}
			f.muMain.Unlock()

			// 先缓存结果，然后一次性更新
			apiResults := map[string][]okexv5api.FundingRateHistory{}
			for i := 0; i < len(instIds); i++ {
				instId := instIds[i]
				resp, err := okexv5api.GetFundingRateHistory(instId, 100, time.Time{}, time.Time{})
				if okexv5api.CheckRestResp(resp.CommonRestResp, err, "get fundingfee history of "+instId, f.logPrefix) {
					apiResults[instId] = resp.Data
				}

				if f.historyReady {
					time.Sleep(time.Second)
				} else {
					f.progress += 2
					time.Sleep(time.Millisecond * 200)
				}
			}

			// 执行更新
			f.muMain.Lock()
			for instId, frhs := range apiResults {
				if v, ok := f.fundingFees[instId]; ok {
					v.FeeHistory = make(map[time.Time]decimal.Decimal)
					for _, frh := range frhs {
						v.FeeHistory[time.UnixMilli(frh.FundingTimeStamp)] = frh.FundingRate
					}
					f.fundingFees[instId] = v
				}
			}
			f.historyReady = true
			f.muMain.Unlock()
		}
		time.Sleep(time.Second * 10)
	}
}

func (f *FundingFeeObserver) GetFeeInfo(instId string) (common.FundingFeeInfo, bool) {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	if fi, ok := f.fundingFees[instId]; ok {
		return fi, true
	} else {
		return common.FundingFeeInfo{}, false
	}
}

func (f *FundingFeeObserver) AllInstIds() []string {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	keys := make([]string, 0, len(f.fundingFees))
	for k := range f.fundingFees {
		keys = append(keys, k)
	}
	return keys
}

func (f *FundingFeeObserver) AllFeeInfo() []common.FundingFeeInfo {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	vals := make([]common.FundingFeeInfo, 0, len(f.fundingFees))
	for _, v := range f.fundingFees {
		vals = append(vals, v)
	}
	return vals
}

func (f *FundingFeeObserver) Ready() (float64, bool) {
	return f.progress / f.progressTotal, f.mainReady && f.historyReady
}
