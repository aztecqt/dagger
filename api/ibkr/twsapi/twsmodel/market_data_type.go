/*
- @Author: aztec
- @Date: 2024-03-01 17:13:27
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "fmt"

type MarketDataType int

const (
	MarketDataType_Live          MarketDataType = 1
	MarketDataType_Frozen        MarketDataType = 2
	MarketDataType_Delayed       MarketDataType = 3
	MarketDataType_DelayedFrozen MarketDataType = 4
)

func MarketDataType2String(t MarketDataType) string {
	switch t {
	case MarketDataType_Live:
		return "live"
	case MarketDataType_Frozen:
		return "frozen"
	case MarketDataType_Delayed:
		return "delayed"
	case MarketDataType_DelayedFrozen:
		return "delayed_frozen"
	default:
		return fmt.Sprintf("unknown: %d", t)
	}
}
