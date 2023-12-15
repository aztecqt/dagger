/*
- @Author: aztec
- @Date: 2023-12-14 11:16:55
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package adv

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi/binancefutureapi"
	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/binance"
	"github.com/aztecqt/dagger/cex/okexv5"
	"github.com/aztecqt/dagger/util"
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
	fees = make([]FundingFeeUnit, 0)

	// okx需要从t1到t0反向获取
	t := t1
	enough := false
	for !enough {
		if resp, err := okexv5api.GetFundingRateHistory(instId, 100, time.Time{}, t); err == nil {
			for _, frhr := range resp.Data {
				ffu := FundingFeeUnit{}
				ffu.FeeRate = util.String2DecimalPanic(frhr.FundingRate)
				ffu.Time = time.UnixMilli(util.String2Int64Panic(frhr.FundingTime))
				t = ffu.Time
				fees = append(fees, ffu)
				if ffu.Time.Unix() <= t0.Unix() {
					enough = true
					break
				}
			}
		} else {
			fmt.Printf("get funding fee from ok failed: %s\n", err.Error())
			time.Sleep(time.Second * 3)
		}
	}

	slices.Reverse(fees)

	// 获取这段时间的1小时k线，为所有费率记录配上价格
	kus := okexv5.GetKline(instId, t0.AddDate(0, 0, -1), t1.AddDate(0, 0, 1), 3600)
	time2Price := make(map[int64]decimal.Decimal)
	for _, k := range kus {
		time2Price[k.Time.Unix()] = k.OpenPrice
	}

	allPriceOk := true
	for i, ffu := range fees {
		if px, ok := time2Price[ffu.Time.Unix()]; ok {
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
	fees = make([]FundingFeeUnit, 0)

	// 转换为币安的symbol
	isUsdt := strings.Contains(instId, "USDT")
	ac := util.ValueIf(isUsdt, binancefutureapi.API_ClassicUsdt, binancefutureapi.API_ClassicUsd)
	if rawFees, err := binancefutureapi.GetHistoryFundingRate(instId, t0, t1, ac); err == nil {
		for _, ff := range *rawFees {
			fu := FundingFeeUnit{
				FeeRate: ff.FundingRate,
				Time:    time.UnixMilli(ff.FundingTimeStamp),
			}
			fees = append(fees, fu)
		}
		ok = true
	} else {
		fmt.Printf("get fundingfee from bn failed: %s\n", err.Error())
		time.Sleep(time.Second * 3)
	}

	// 获取这段时间的1小时k线，为所有费率记录配上价格
	klfn := util.ValueIf(isUsdt, binancefutureapi.GetKline_Usdt, binancefutureapi.GetKline_Usd)
	kus := binance.GetKline(instId, t0, t1, 3600, klfn)
	time2Price := make(map[int64]decimal.Decimal)
	for _, k := range kus {
		time2Price[k.Time.Unix()] = k.OpenPrice
	}

	allPriceOk := true
	for i, ffu := range fees {
		if px, ok := time2Price[ffu.Time.Unix()]; ok {
			fees[i].MarkPrice = px
		} else {
			allPriceOk = false
		}
	}

	ok = allPriceOk
	return
}
