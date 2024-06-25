/*
- @Author: aztec
- @Date: 2023-12-18 09:48:58
- @Description: 带本地缓存的数据访问接口
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedok

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

// 获取一段时间内的历史仓位（基于缓存）
func GetPositionHistory(acc, instType string, t0, t1 time.Time, fn func(prg time.Time)) ([]okexv5api.PositionHistory, bool) {
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1).AddDate(0, 0, 1)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]okexv5api.PositionHistory, 0)
	for dt := dt0; dt.Before(dt1); dt = dt.AddDate(0, 0, 1) {
		if v, ok := loadPositionHistoryOfDate(acc, instType, dt); ok {
			if !apit0.IsZero() && !apit1.IsZero() {
				// 加载在线数据/保存到结果/保存到缓存
				phs := getPositionHistoryFromApi(instType, apit0, apit1, fn)
				result = append(result, phs...)
				savePositionHistorys(acc, instType, phs)
				apit0 = time.Time{}
				apit1 = time.Time{}
			}
			if fn != nil {
				fn(dt)
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
		phs := getPositionHistoryFromApi(instType, apit0, apit1, fn)
		result = append(result, phs...)
		savePositionHistorys(acc, instType, phs)
	}

	// 对result进行截断
	for i, ph := range result {
		if ph.UpdateTime.UnixMilli() >= t0.UnixMilli() {
			result = result[i:]
			break
		}
	}

	for i, ph := range result {
		if ph.UpdateTime.UnixMilli() >= t1.UnixMilli() {
			result = result[:i]
			break
		}
	}

	return result, true
}

// 从api中获取一段时间内的历史记录[t0, t1)
func getPositionHistoryFromApi(instType string, t0, t1 time.Time, fn func(prg time.Time)) []okexv5api.PositionHistory {
	var phs []okexv5api.PositionHistory
	finished := false
	for t := t0.Add(-time.Millisecond); !finished; {
		if resp, err := okexv5api.GetPositionHistory(instType, "", okexv5api.PositionCloseType_None, t); err == nil && resp.Code == "0" {
			if len(resp.Data) == 0 {
				finished = true
				break
			} else {
				for _, ph := range resp.Data {
					t = ph.UpdateTime
					if t.Before(t1) {
						phs = append(phs, ph)
					} else {
						finished = true
						break
					}
				}

				if fn != nil {
					fn(t)
				}
			}
		} else {
			if err != nil {
				logger.LogImportant(logPrefix, "load pos-history from okex failed: %s\n", err.Error())
			} else {
				logger.LogImportant(logPrefix, "load pos-history from okex failed: %s\n", resp.Msg)
			}

			time.Sleep(time.Second * 2)
		}
		time.Sleep(time.Second * 2)
	}
	return phs
}

func positionHistoryCachePath(acc, instType string, dt time.Time) string {
	return fmt.Sprintf("%s/dagger/okx/position_history/%s/%s/%s.poshis", util.SystemCachePath(), acc, instType, dt.Format(time.DateOnly))
}

func loadPositionHistoryOfDate(acc, instType string, dt time.Time) ([]okexv5api.PositionHistory, bool) {
	path := positionHistoryCachePath(acc, instType, dt)
	var phs []okexv5api.PositionHistory
	if util.ObjectFromFile(path, &phs) {
		return phs, true
	} else {
		return nil, false
	}
}

func savePositionHistorys(acc, instType string, phs []okexv5api.PositionHistory) {
	// 有两种情况不能保存：
	// 1、当日数据不要保存（因为数据还不全）
	// 2、存在“部分平仓”仓位的当日数据
	today := util.DateOfTime(time.Now())
	dataByPath := make(map[string][]okexv5api.PositionHistory)

	// 仅添加非当日数据
	for _, ph := range phs {
		if ph.UpdateTime.UnixMilli() < today.UnixMilli() {
			path := positionHistoryCachePath(acc, instType, ph.UpdateTime)
			phs := dataByPath[path]
			phs = append(phs, ph)
			dataByPath[path] = phs
		}
	}

	// 去除存在“部分平仓”的那些文件
	for path, v := range dataByPath {
		for _, ph := range v {
			if ph.Type == okexv5api.PositionCloseType_PartlyClosed {
				delete(dataByPath, path)
				break
			}
		}
	}

	for path, v := range dataByPath {
		util.ObjectToFile(path, v)
	}
}
