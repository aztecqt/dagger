/*
- @Author: aztec
- @Date: 2023-12-20 15:11:46
- @Description: 资金费率历史。按月为单位保存缓存数据。数据量不大，直接存明文（OKX最多仅能获取最近3个月的数据）
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedok

import (
	"fmt"
	"slices"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"
)

func GetFundingFees(instId string, t0, t1 time.Time, fnprg fnprg) (fees []okexv5api.FundingRateHistory) {
	dt0 := util.MonthOfTime(t0)
	dt1 := util.MonthOfTime(t1).AddDate(0, 1, 0)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]okexv5api.FundingRateHistory, 0)
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
	return fmt.Sprintf("%s/dagger/okx/fundingfees/%s/%s.kline", util.SystemCachePath(), instId, dt.Format("2006-01"))
}

func loadFundingFeesOfMonth(instId string, dt time.Time) ([]okexv5api.FundingRateHistory, bool) {
	if DisableCached {
		return nil, false
	}

	path := pathOfFundingFees(instId, dt)
	result := make([]okexv5api.FundingRateHistory, 0)
	return result, util.ObjectFromFile(path, &result)
}

func saveFundingFees(instId string, fees []okexv5api.FundingRateHistory) {
	// 当月数据不要保存，因为还不全
	thisMonth := util.MonthOfTime(time.Now())
	dataByPath := make(map[string][]okexv5api.FundingRateHistory)
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

func getFundingFeesFromApi(instId string, t0, t1 time.Time, fnprg fnprg) []okexv5api.FundingRateHistory {
	// okx需要从t1到t0反向获取
	result := make([]okexv5api.FundingRateHistory, 0)
	t := t1
	enough := false
	for !enough {
		if resp, err := okexv5api.GetFundingRateHistory(instId, 100, time.Time{}, t); err == nil {
			if len(resp.Data) == 0 {
				enough = true
			} else {
				for _, frhr := range resp.Data {
					if frhr.FundingTimeStamp < t0.UnixMilli() {
						enough = true
						break
					}
					t = time.UnixMilli(frhr.FundingTimeStamp)
					result = append(result, frhr)
				}
				if fnprg != nil {
					fnprg(t)
				}
			}
		} else {
			fmt.Printf("get funding fee from ok failed: %s\n", err.Error())
			time.Sleep(time.Second * 3)
		}
	}

	slices.Reverse(result)
	return result
}
