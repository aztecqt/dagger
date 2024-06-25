/*
- @Author: aztec
- @Date: 2023-12-20 16:29:11
- @Description: 资金费率历史。按月为单位保存缓存数据。数据量不大，直接存明文
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedbn

import (
	"fmt"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancefutureapi"
	"github.com/aztecqt/dagger/util"
)

func GetFundingFees(instId string, t0, t1 time.Time, fnprg fnprg) (fees []binanceapi.FundingFee) {
	dt0 := util.MonthOfTime(t0)
	dt1 := util.MonthOfTime(t1).AddDate(0, 1, 0)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]binanceapi.FundingFee, 0)
	for dt := dt0; dt.Before(dt1); dt = dt.AddDate(0, 1, 0) {
		if v, ok := loadFundingFeesOfMonth(instId, dt); ok {
			if !apit0.IsZero() && !apit1.IsZero() {
				// 加载在线数据/保存到结果/保存到缓存
				ffs := getFundingFeesFromApi(instId, apit0, apit1, fnprg)
				result = append(result, ffs...)
				saveFundingFees(instId, ffs)
				apit0 = time.Time{}
				apit1 = time.Time{}
			}

			if fnprg != nil {
				fnprg(dt)
			}
			result = append(result, v...)
		} else {
			if apit0.IsZero() {
				apit0 = dt
			}
			apit1 = dt.AddDate(0, 1, 0).Add(-time.Second)
		}
	}

	if !apit0.IsZero() && !apit1.IsZero() {
		// 加载在线数据/保存到结果/保存到缓存
		ffs := getFundingFeesFromApi(instId, apit0, apit1, fnprg)
		result = append(result, ffs...)
		saveFundingFees(instId, ffs)
	}

	// 对result进行截断
	for i, ff := range result {
		if ff.FundingTimeStamp >= t0.UnixMilli() {
			result = result[i:]
			break
		}
	}

	for i, ff := range result {
		if ff.FundingTimeStamp >= t1.UnixMilli() {
			result = result[:i]
			break
		}
	}

	return result
}

func pathOfFundingFees(instId string, dt time.Time) string {
	return fmt.Sprintf("%s/dagger/binance/fundingfees/%s/%s.kline", util.SystemCachePath(), instId, dt.Format("2006-01"))
}

func loadFundingFeesOfMonth(instId string, dt time.Time) ([]binanceapi.FundingFee, bool) {
	if DisableCached {
		return nil, false
	}

	path := pathOfFundingFees(instId, dt)
	result := make([]binanceapi.FundingFee, 0)
	return result, util.ObjectFromFile(path, &result)
}

func saveFundingFees(instId string, fees []binanceapi.FundingFee) {
	// 当月数据不要保存，因为还不全
	thisMonth := util.MonthOfTime(time.Now())
	dataByPath := make(map[string][]binanceapi.FundingFee)
	for _, ff := range fees {
		if ff.FundingTimeStamp < thisMonth.UnixMilli() {
			path := pathOfFundingFees(instId, time.UnixMilli(ff.FundingTimeStamp))
			ffs := dataByPath[path]
			ffs = append(ffs, ff)
			dataByPath[path] = ffs
		}
	}

	// 执行保存
	for path, v := range dataByPath {
		util.ObjectToFile(path, v)
	}
}

func getFundingFeesFromApi(instId string, t0, t1 time.Time, fnprg fnprg) []binanceapi.FundingFee {
	isUsdt := strings.Contains(instId, "USDT")
	ac := util.ValueIf(isUsdt, binancefutureapi.API_ClassicUsdt, binancefutureapi.API_ClassicUsd)
	result := make([]binanceapi.FundingFee, 0)
	t := t0
	enough := false
	for !enough {
		if resp, err := binancefutureapi.GetHistoryFundingRate(instId, t, time.Time{}, 1000, ac); err == nil {
			time.Sleep(time.Millisecond * 1200) // 频率限制：5分钟500次
			if len(*resp) == 0 {
				enough = true
			} else {
				for _, fr := range *resp {
					if fr.FundingTimeStamp > t1.UnixMilli() {
						enough = true
						break
					}
					t = time.UnixMilli(fr.FundingTimeStamp).Add(time.Second)
					result = append(result, fr)
				}
				if fnprg != nil {
					fnprg(t)
				}
			}
		} else {
			fmt.Printf("get funding fee from bn failed: %s\n", err.Error())
			time.Sleep(time.Second * 10)
		}
	}

	return result
}
