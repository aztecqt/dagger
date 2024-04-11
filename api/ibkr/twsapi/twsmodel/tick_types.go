/*
- @Author: aztec
- @Date: 2024-03-01 12:30:59
- @Description: 只搬了会用到的几个定义，其他定义按需搬运
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "fmt"

type TickType int

const (
	TickType_Bid               TickType = 1
	TickType_Ask               TickType = 2
	TickType_Last              TickType = 4
	TickType_BidSize           TickType = 0
	TickType_AskSize           TickType = 3
	TickType_LastSize          TickType = 5
	TickType_VOLUME            TickType = 8
	TickType_CLOSE             TickType = 9
	TickType_LAST_TIMESTAMP    TickType = 45
	TickType_HALTED            TickType = 49
	TickType_DELAYED_BID       TickType = 66
	TickType_DELAYED_ASK       TickType = 67
	TickType_DELAYED_BID_SIZE  TickType = 69
	TickType_DELAYED_ASK_SIZE  TickType = 70
	TickType_DELAYED_LAST      TickType = 68
	TickType_DELAYED_LAST_SIZE TickType = 71
)

func TickType2Str(t TickType) string {
	switch t {
	case TickType_Bid:
		return "bid"
	case TickType_Ask:
		return "ask"
	case TickType_Last:
		return "last"
	case TickType_BidSize:
		return "bidSize"
	case TickType_AskSize:
		return "askSize"
	case TickType_LastSize:
		return "lastSize"
	case TickType_VOLUME:
		return "volume"
	case TickType_CLOSE:
		return "close"
	case TickType_LAST_TIMESTAMP:
		return "last_ts"
	case TickType_HALTED:
		return "halted"
	case TickType_DELAYED_BID:
		return "bidDelayed"
	case TickType_DELAYED_ASK:
		return "askDelayed"
	case TickType_DELAYED_LAST:
		return "lastDelayed"
	case TickType_DELAYED_BID_SIZE:
		return "bidSizeDelayed"
	case TickType_DELAYED_ASK_SIZE:
		return "askSizeDelayed"
	case TickType_DELAYED_LAST_SIZE:
		return "lastSizeDelayed"
	default:
		return fmt.Sprintf("unknown:%d", t)
	}
}
