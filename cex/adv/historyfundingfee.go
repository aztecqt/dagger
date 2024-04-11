/*
- @Author: aztec
- @Date: 2023-12-14 11:16:55
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package adv

import (
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi/cachedbn"
	"github.com/aztecqt/dagger/api/okexv5api/cachedok"
	"github.com/aztecqt/dagger/cex/binance"
	"github.com/aztecqt/dagger/cex/okexv5"
	"github.com/shopspring/decimal"
)

type FundingFeeUnit struct {
	FeeRate   decimal.Decimal
	MarkPrice decimal.Decimal
	Time      time.Time
}

// ex: binance, okx
func GetHistoryFundingFeeRate(ex, symbol, contractType string, t0, t1 time.Time) (fees []FundingFeeUnit, ok bool) {
	ex = strings.ToLower(ex)
	if ex == "binance" {
		return getBinanceFundingFee(t0, t1, symbol, contractType)
	} else if ex == "okx" {
		return getOkxFundingFee(t0, t1, symbol, contractType)
	} else {
		return []FundingFeeUnit{}, false
	}
}

func GetHistoryFundingFeeRateByInstId(ex, instId string, t0, t1 time.Time) (fees []FundingFeeUnit, ok bool) {
	ex = strings.ToLower(ex)
	if ex == "binance" {
		return getBinanceFundingFeeByInstId(t0, t1, instId)
	} else if ex == "okx" {
		return getOkxFundingFeeByInstId(t0, t1, instId)
	} else {
		return []FundingFeeUnit{}, false
	}
}

func getOkxFundingFee(t0, t1 time.Time, symbol, contractType string) (fees []FundingFeeUnit, ok bool) {
	fees = make([]FundingFeeUnit, 0)

	// 转换为okx的instId
	instId := okexv5.CCyCttypeToInstId(symbol, contractType)
	return getOkxFundingFeeByInstId(t0, t1, instId)
}

func getOkxFundingFeeByInstId(t0, t1 time.Time, instId string) (fees []FundingFeeUnit, ok bool) {
	// 加载费率
	rawFees := cachedok.GetFundingFees(instId, t0, t1, nil)
	fees = make([]FundingFeeUnit, 0)
	for _, ffr := range rawFees {
		ffu := FundingFeeUnit{}
		ffu.FeeRate = ffr.FundingRate
		ffu.Time = time.UnixMilli(ffr.FundingTimeStamp)
		fees = append(fees, ffu)
	}

	// 获取这段时间的1小时k线，为所有费率记录配上价格
	kus, _ := cachedok.GetKline(instId, t0.AddDate(0, 0, -1), t1.AddDate(0, 0, 1), 3600, nil)
	time2Price := make(map[int64]decimal.Decimal)
	for _, k := range kus {
		time2Price[k.Time.Unix()/3600] = k.Open
	}

	allPriceOk := true
	for i, ffu := range fees {
		if px, ok := time2Price[ffu.Time.Unix()/3600]; ok {
			fees[i].MarkPrice = px
		} else {
			allPriceOk = false
		}
	}

	return fees, allPriceOk
}

func getBinanceFundingFee(t0, t1 time.Time, symbol, contractType string) (fees []FundingFeeUnit, ok bool) {
	fees = make([]FundingFeeUnit, 0)

	// 转换为币安的symbol
	instId := binance.CCyCttypeToInstId(symbol, contractType)
	return getBinanceFundingFeeByInstId(t0, t1, instId)
}

func getBinanceFundingFeeByInstId(t0, t1 time.Time, instId string) (fees []FundingFeeUnit, ok bool) {
	// 加载费率
	rawFees := cachedbn.GetFundingFees(instId, t0, t1, nil)
	fees = make([]FundingFeeUnit, 0)
	for _, ffr := range rawFees {
		ffu := FundingFeeUnit{}
		ffu.FeeRate = ffr.FundingRate
		ffu.Time = time.UnixMilli(ffr.FundingTimeStamp)
		fees = append(fees, ffu)
	}

	if len(fees) == 0 {
		return fees, true
	}

	// 获取这段时间的1小时k线，为所有费率记录配上价格
	kus, _ := cachedbn.GetFutureKline(instId, t0, t1, 3600, nil)
	time2Price := make(map[int64]decimal.Decimal)
	for _, k := range kus {
		time2Price[k.Time.Unix()/3600] = k.Open
	}

	allPriceOk := true
	for i, ffu := range fees {
		if px, ok := time2Price[ffu.Time.Unix()/3600]; ok {
			fees[i].MarkPrice = px
		} else {
			allPriceOk = false
		}
	}

	ok = allPriceOk
	return
}
