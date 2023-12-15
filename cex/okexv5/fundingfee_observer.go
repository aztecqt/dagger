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

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

type FundingFeeInfo struct {
	common.FundingFeeInfo
	inst common.Instruments
}

type FundingFeeObserver struct {
	muMain     sync.Mutex
	muRestFreq sync.Mutex

	ex          *Exchange
	logPrefix   string
	fundingFees map[string]*FundingFeeInfo
}

func (f *FundingFeeObserver) init(ex *Exchange) {
	f.ex = ex
	f.logPrefix = "funding-fee-observer"
	f.fundingFees = make(map[string]*FundingFeeInfo)
}

func (f *FundingFeeObserver) AddType(t string) {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	if _, ok := f.fundingFees[t]; !ok {
		f.fundingFees[t] = nil

		go func() {
			// 循环拉取交易对信息（ticker、fee）
			ticker := time.NewTicker(time.Second)
			for {
				<-ticker.C
				f.refresh(t)
				ticker.Reset(time.Second * 60)
			}
		}()
	}
}

func (f *FundingFeeObserver) GetFeeInfo(t string) (common.FundingFeeInfo, bool) {
	f.muMain.Lock()
	defer f.muMain.Unlock()

	if fi, ok := f.fundingFees[t]; ok {
		return fi.FundingFeeInfo, true
	} else {
		return common.FundingFeeInfo{}, false
	}
}

func (f *FundingFeeObserver) AllTypes() []string {
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
		if v != nil {
			vals = append(vals, v.FundingFeeInfo)
		}
	}
	return vals
}

func (f *FundingFeeObserver) refresh(instId string) {
	f.muRestFreq.Lock()
	defer f.muRestFreq.Unlock()
	defer time.Sleep(time.Millisecond * 200) // 限定频率最多1秒5次左右

	finfo := FundingFeeInfo{inst: *f.ex.instrumentMgr.Get(instId)}
	finfo.TradeType = instId

	// 拉取现货ticker（获取现货价格）
	{
		ccy := InstIdToCcy(instId)
		instIdSpot := SpotTypeToInstId(ccy, "usdt")
		resp, err := okexv5api.GetTicker(instIdSpot)
		if err == nil && resp.Code == "0" {
			spotPrice, _ := util.String2Decimal(resp.Data[0].Last)

			// 拉取ticker（获取24小时成交量）
			resp, err := okexv5api.GetTicker(instId)
			if err == nil && resp.Code == "0" {
				swapPrice, _ := util.String2Decimal(resp.Data[0].Last)
				volCcy24h, _ := util.String2Decimal(resp.Data[0].VolCcy24h)
				finfo.PriceRatio = spotPrice.Div(swapPrice)
				finfo.VolUSD24h = volCcy24h.Mul(spotPrice)
			} else {
				logger.LogInfo(f.logPrefix, "refresh %s failed, can't get swap ticker, err=%s", instId, err.Error())
				return
			}
		} else {
			logger.LogInfo(f.logPrefix, "refresh %s failed, can't get spot ticker, err=%s", instId, err.Error())
			return
		}
	}

	// 拉取当前费率
	{
		resp, err := okexv5api.GetFundingRate(instId)
		if err == nil && resp.Code == "0" {
			finfo.FeeRate = util.String2DecimalPanic(resp.Data[0].FundingRate)
			finfo.NextFeeRate = util.String2DecimalPanic(resp.Data[0].NextFundingRate)
			finfo.FeeTime = util.ConvetUnix13StrToTimePanic(resp.Data[0].FundingTime)
			finfo.NextFeeTime = util.ConvetUnix13StrToTimePanic(resp.Data[0].NextFundingTime)
		} else {
			logger.LogInfo(f.logPrefix, "refresh %s failed, can't get fee-rate, err=%s", instId, err.Error())
			return
		}
	}

	// 拉取历史费率
	{
		resp, err := okexv5api.GetFundingRateHistory(instId, 30, time.Time{}, time.Time{})
		if err == nil && resp.Code == "0" {
			finfo.FeeHistory = make(map[time.Time]decimal.Decimal)
			for _, data := range resp.Data {
				fr := util.String2DecimalPanic(data.FundingRate)
				tm := util.ConvetUnix13StrToTimePanic(data.FundingTime)
				finfo.FeeHistory[tm] = fr
			}
		} else {
			logger.LogInfo(f.logPrefix, "refresh %s failed, can't get fee-rate-history, err=%s", instId, err.Error())
			return
		}
	}

	finfo.RefreshTime = time.Now()
	f.fundingFees[instId] = &finfo
	logger.LogInfo(f.logPrefix, "refresh %s success", instId)
}
