/*
 * @Author: aztec
 * @Date: 2024-01-28 09:34:37
 * @Description: 带本地缓存的数据访问接口
 *
 * Copyright (c) 2024 by aztec, All Rights Reserved.
 */
package cachedok

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

// 获取一段时间内的账单
func GetBills(acc string, t0, t1 time.Time, fn func(prg time.Time)) ([]okexv5api.Bill, bool) {
	dt0 := util.DateOfTime(t0)
	dt1 := util.DateOfTime(t1).AddDate(0, 0, 1)
	apit0 := time.Time{}
	apit1 := time.Time{}
	result := make([]okexv5api.Bill, 0)
	for dt := dt0; dt.Before(dt1); dt = dt.AddDate(0, 0, 1) {
		if v, ok := loadBillsOfDate(acc, dt); ok {
			if !apit0.IsZero() && !apit1.IsZero() {
				// 加载在线数据/保存到结果/保存到缓存
				bills := getBillsFromApi(apit0, apit1, fn)
				result = append(result, bills...)
				saveBills(acc, bills)
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
		bills := getBillsFromApi(apit0, apit1, fn)
		result = append(result, bills...)
		saveBills(acc, bills)
	}

	// 对result进行截断
	for i, ph := range result {
		if ph.Time.UnixMilli() >= t0.UnixMilli() {
			result = result[i:]
			break
		}
	}

	for i, ph := range result {
		if ph.Time.UnixMilli() >= t1.UnixMilli() {
			result = result[:i]
			break
		}
	}

	return result, true
}

// 从api中获取一段时间内的账单[t0, t1)
func getBillsFromApi(t0, t1 time.Time, fn func(prg time.Time)) []okexv5api.Bill {
	var bills []okexv5api.Bill
	finished := false
	fromTradeId := ""
	for !finished {
		if resp, err := okexv5api.GetBills(fromTradeId, t0, 100); err == nil && resp.Code == "0" {
			if len(resp.Data) == 0 {
				finished = true
				break
			} else {
				prg := time.Time{}
				for i := len(resp.Data) - 1; i >= 0; i-- {
					bill := resp.Data[i]
					prg = bill.Time
					fromTradeId = bill.BillId
					if prg.Before(t1) {
						bills = append(bills, bill)
					} else {
						finished = true
						break
					}
				}

				if fn != nil {
					fn(prg)
				}
			}
		} else {
			if err != nil {
				logger.LogImportant(logPrefix, "load bills from okex failed: %s\n", err.Error())
			} else {
				logger.LogImportant(logPrefix, "load bills from okex failed: %s\n", resp.Msg)
			}

			time.Sleep(time.Millisecond * 500)
		}
		time.Sleep(time.Millisecond * 500)
	}
	return bills
}

// bills缓存路径
func billsCachePath(acc string, dt time.Time) string {
	return fmt.Sprintf("%s/dagger/okx/bills/%s/%s.bills", util.SystemCachePath(), acc, dt.Format(time.DateOnly))
}

// 加载某一日的bills
func loadBillsOfDate(acc string, dt time.Time) ([]okexv5api.Bill, bool) {
	path := billsCachePath(acc, dt)
	var bills []okexv5api.Bill
	if util.ObjectFromFile(path, &bills) {
		return bills, true
	} else {
		return nil, false
	}
}

// 保存bills缓存
func saveBills(acc string, bills []okexv5api.Bill) {
	// 当日数据不要保存，因为还不全
	today := util.DateOfTime(time.Now())
	dataByPath := make(map[string][]okexv5api.Bill)
	for _, bill := range bills {
		if bill.Time.UnixMilli() < today.UnixMilli() {
			path := billsCachePath(acc, bill.Time)
			billOfDate := dataByPath[path]
			billOfDate = append(billOfDate, bill)
			dataByPath[path] = billOfDate
		}
	}

	// 执行保存
	for path, v := range dataByPath {
		util.ObjectToFile(path, v)
	}
}
