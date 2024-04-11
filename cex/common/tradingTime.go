/*
- @Author: aztec
- @Date: 2024-03-29 09:27:46
- @Description: 开盘/收盘时间
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package common

import "time"

// 交易时间片段
type TradingTimeSeg struct {
	OpenTime  time.Time
	CloseTime time.Time
}

type TradingTimes []TradingTimeSeg

func (t *TradingTimes) Contains(tm time.Time) bool {
	for _, v := range *t {
		if v.OpenTime.Before(tm) && v.CloseTime.After(tm) {
			return true
		}
	}
	return false
}
