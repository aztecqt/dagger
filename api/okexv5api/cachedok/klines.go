/*
- @Author: aztec
- @Date: 2023-12-19 11:57:56
- @Description: 以缓存的方式获取k线数据。范围从1min-1d。更大级别的k线也不用缓存了
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedok

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

func GetKline(instId string, t0, t1 time.Time, intervalSec int, fnprg fnprg) ([]okexv5api.KLineUnit, bool) {
	bar, barOk := getBar(intervalSec)
	if !barOk {
		return nil, false
	}

	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1).AddDate(0, 0, 1)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]okexv5api.KLineUnit, 0)
	for dt := dt0; dt.Before(dt1); dt = dt.AddDate(0, 0, 1) {
		if v, ok := loadKlineOfDate(instId, bar, dt); ok {
			if !apit0.IsZero() && !apit1.IsZero() {
				// 加载在线数据/保存到结果/保存到缓存
				kus := getKlineFromApi(instId, apit0, apit1, bar, fnprg)
				result = append(result, kus...)
				saveKlines(instId, bar, kus)
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
			apit1 = dt.AddDate(0, 0, 1)
		}
	}

	if !apit0.IsZero() && !apit1.IsZero() {
		// 加载在线数据/保存到结果/保存到缓存
		kus := getKlineFromApi(instId, apit0, apit1, bar, fnprg)
		result = append(result, kus...)
		saveKlines(instId, bar, kus)
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
		bar = "1H"
	case 3600 * 2:
		bar = "2H"
	case 3600 * 4:
		bar = "4H"
	case 86400:
		bar = "1D"
	default:
		logger.LogPanic(logPrefix, "invalid kline intervalsec: %d", intervalSec)
	}
	return bar, len(bar) > 0
}

func klineCachePath(instId, bar string, dt time.Time) string {
	appDataPath := os.Getenv("APPDATA")
	return fmt.Sprintf("%s/dagger/okx/klines/%s/%s/%s.kline", appDataPath, instId, bar, dt.Format(time.DateOnly))
}

// 加载某一日的k线数据
func loadKlineOfDate(instId, bar string, dt time.Time) ([]okexv5api.KLineUnit, bool) {
	if DisableCached {
		return nil, false
	}

	path := klineCachePath(instId, bar, dt)
	result := make([]okexv5api.KLineUnit, 0)
	ok := util.FileDeserializeToObjects(
		path,
		func() *okexv5api.KLineUnit { return &okexv5api.KLineUnit{} },
		func(ku *okexv5api.KLineUnit) bool { result = append(result, *ku); return true })
	return result, ok
}

// 保证kl按时间排序
func saveKlines(instId, bar string, kl []okexv5api.KLineUnit) {
	// 当日数据不要保存，因为还不全
	today := util.DateOfTime(time.Now())
	dataByPath := make(map[string][]okexv5api.KLineUnit)
	for _, ku := range kl {
		if ku.Time.UnixMilli() < today.UnixMilli() {
			path := klineCachePath(instId, bar, ku.Time)
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

// 在线数据加载
func getKlineFromApi(instId string, t0, t1 time.Time, bar string, fnprg fnprg) []okexv5api.KLineUnit {
	tEnd := t1
	kus := make([]okexv5api.KLineUnit, 0)
	errCount := 0
	for errCount < 10 {
		resp, err := okexv5api.GetKlineBefore(instId, tEnd, bar, 0)
		if err != nil {
			logger.LogImportant(logPrefix, err.Error())
			time.Sleep(time.Second)
			errCount++
		} else if resp.Code != "0" {
			logger.LogImportant(logPrefix, resp.Msg)
			time.Sleep(time.Second)
			errCount++
		} else {
			// 取不到数据就认为结束了
			if len(resp.Data) == 0 {
				break
			}

			for _, ku := range resp.Data {
				tEnd = ku.Time
				if tEnd.Before(t0) {
					break
				}
				kus = append(kus, ku)
			}

			if fnprg != nil {
				fnprg(tEnd)
			}

			// 取到足够多的数据也认为结束了
			if tEnd.Before(t0) {
				break
			}
		}
	}

	slices.Reverse(kus)
	return kus
}
