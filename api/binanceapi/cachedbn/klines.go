/*
- @Author: aztec
- @Date: 2023-12-20 10:27:58
- @Description: 以缓存的方式获取k线数据。范围从1min-1d。更大级别的k线也不用缓存了
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedbn

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancefutureapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancespotapi"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

func GetSpotKline(instId string, t0, t1 time.Time, intervalSec int, fnPrg fnprg) ([]binanceapi.KLineUnit, bool) {
	if bar, ok := getBar(intervalSec); ok {
		return getKline("spot", instId, t0, t1, bar, false, binancespotapi.GetKline, fnPrg)
	} else {
		return nil, false
	}
}

func GetFutureKline(instId string, t0, t1 time.Time, intervalSec int, fnPrg fnprg) ([]binanceapi.KLineUnit, bool) {
	if bar, ok := getBar(intervalSec); ok {
		if strings.Contains(instId, "USDT") {
			return getKline("future", instId, t0, t1, bar, false, binancefutureapi.GetKline_Usdt, fnPrg)
		} else {
			return getKline("future", instId, t0, t1, bar, true, binancefutureapi.GetKline_Usd, fnPrg)
		}
	} else {
		return nil, false
	}
}

func getBar(intervalSec int) (string, bool) {
	bar := ""
	switch intervalSec {
	case 60:
		bar = "1m"
	case 60 * 3:
		bar = "3m"
	case 60 * 5:
		bar = "5m"
	case 60 * 15:
		bar = "15m"
	case 60 * 30:
		bar = "30m"
	case 3600:
		bar = "1h"
	case 3600 * 2:
		bar = "2h"
	case 3600 * 4:
		bar = "4h"
	case 86400:
		bar = "1d"
	default:
		logger.LogPanic(logPrefix, "invalid kline intervalsec for okx: %d", intervalSec)
	}
	return bar, len(bar) > 0
}

func getKline(instType, instId string, t0, t1 time.Time, bar string, reversed bool, fnApi fnKlineRaw, fnPrg fnprg) ([]binanceapi.KLineUnit, bool) {
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1).AddDate(0, 0, 1)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]binanceapi.KLineUnit, 0)

	for dt := dt0; dt.Before(dt1); dt = dt.AddDate(0, 0, 1) {
		if v, ok := loadKlineOfDate(instType, instId, bar, dt); ok {
			if !apit0.IsZero() && !apit1.IsZero() {
				// 加载在线数据/保存到结果/保存到缓存
				kus := getKlineFromApi(instId, apit0, apit1, bar, reversed, fnApi, fnPrg)
				result = append(result, kus...)
				saveKlines(instType, instId, bar, kus)
				apit0 = time.Time{}
				apit1 = time.Time{}
			}

			if fnPrg != nil {
				fnPrg(dt)
			}
			result = append(result, v...)
		} else {
			if apit0.IsZero() {
				apit0 = dt
			}
			apit1 = dt.AddDate(0, 0, 1)
		}
	}

	if !apit0.IsZero() && !apit1.IsZero() {
		// 加载在线数据/保存到结果/保存到缓存
		kus := getKlineFromApi(instId, apit0, apit1, bar, reversed, fnApi, fnPrg)
		result = append(result, kus...)
		saveKlines(instType, instId, bar, kus)
	}

	// 对result进行截断
	for i, ku := range result {
		if ku.Time.UnixMilli() >= t0.UnixMilli() {
			result = result[i:]
			break
		}
	}

	for i, ku := range result {
		if ku.Time.UnixMilli() >= t1.UnixMilli() {
			result = result[:i]
			break
		}
	}

	return result, true
}

func klineCachePath(instType, instId, bar string, dt time.Time) string {
	appDataPath := os.Getenv("APPDATA")
	return fmt.Sprintf("%s/dagger/binance/klines/%s/%s/%s/%s.kline", appDataPath, instType, instId, bar, dt.Format(time.DateOnly))
}

// 加载某一日的k线数据
func loadKlineOfDate(instType, instId, bar string, dt time.Time) ([]binanceapi.KLineUnit, bool) {
	if DisableCached {
		return nil, false
	}
	path := klineCachePath(instType, instId, bar, dt)
	result := make([]binanceapi.KLineUnit, 0)
	ok := util.FileDeserializeToObjects(
		path,
		func() *binanceapi.KLineUnit { return &binanceapi.KLineUnit{} },
		func(o *binanceapi.KLineUnit) bool { result = append(result, *o); return true },
	)
	return result, ok
}

// 保证kl按时间排序
func saveKlines(instType, instId, bar string, kl []binanceapi.KLineUnit) {
	// 当日数据不要保存，因为还不全
	today := util.DateOfTime(time.Now())
	dataByPath := make(map[string][]binanceapi.KLineUnit)
	for _, ku := range kl {
		if ku.Time.UnixMilli() < today.UnixMilli() {
			path := klineCachePath(instType, instId, bar, ku.Time)
			kus := dataByPath[path]
			kus = append(kus, ku)
			dataByPath[path] = kus
		}
	}

	// 执行保存
	for path, v := range dataByPath {
		buf := &bytes.Buffer{}
		for _, ku := range v {
			ku.Serialize(buf)
		}

		util.BytesToFile(path, buf.Bytes())
	}
}

// 正序。从t1往t0方向取
func getKlineFromApi(instId string, t0, t1 time.Time, bar string, reversed bool, fnKlineApi fnKlineRaw, fnPrg fnprg) []binanceapi.KLineUnit {
	if reversed {
		tEnd := t1
		kus := make([]binanceapi.KLineUnit, 0)
		finished := false
		errCount := 0
		for !finished && errCount < 5 {
			resp, err := fnKlineApi(instId, bar, time.Time{}, tEnd, 1000)
			if err == nil {
				if len(*resp) == 0 {
					finished = true
				} else {
					temp := make([]binanceapi.KLineUnit, 0)
					for i, v := range *resp {
						ku := binanceapi.KLineUnit{}
						ku.FromRaw(v)
						if ku.Time.UnixMilli() <= t0.UnixMilli() {
							finished = true
							break
						}

						if i == 0 {
							tEnd = ku.Time.Add(-time.Second)
						}
						temp = append(temp, ku)
					}
					kus = append(temp, kus...)

					if fnPrg != nil {
						fnPrg(tEnd)
					}
				}
			} else {
				logger.LogImportant(logPrefix, "get kline from ex failed: %s", err.Error())
				time.Sleep(time.Second * 10)
				errCount++
			}
		}

		return kus
	} else {
		tStart := t0
		kus := make([]binanceapi.KLineUnit, 0)
		finished := false
		errCount := 0
		for !finished && errCount < 5 {
			resp, err := fnKlineApi(instId, bar, tStart, time.Time{}, 1000)
			if err == nil {
				if len(*resp) == 0 {
					finished = true
				} else {
					for _, v := range *resp {
						ku := binanceapi.KLineUnit{}
						ku.FromRaw(v)
						if ku.Time.UnixMilli() >= t1.UnixMilli() {
							finished = true
							break
						}

						tStart = ku.Time.Add(time.Second)
						kus = append(kus, ku)
					}

					if fnPrg != nil {
						fnPrg(tStart)
					}
				}
			} else {
				logger.LogImportant(logPrefix, "get kline from ex failed: %s", err.Error())
				time.Sleep(time.Second * 10)
				errCount++
			}
		}

		return kus
	}
}
