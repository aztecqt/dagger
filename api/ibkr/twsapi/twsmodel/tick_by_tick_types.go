/*
- @Author: aztec
- @Date: 2024-03-01 19:02:12
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "fmt"

type TickByTickType int

const (
	TickByTickType_None     TickByTickType = 0
	TickByTickType_Last     TickByTickType = 1
	TickByTickType_AllLast  TickByTickType = 2
	TickByTickType_BidAsk   TickByTickType = 3
	TickByTickType_MidPoint TickByTickType = 4
)

func TickByTickType2String(t TickByTickType) string {
	switch t {
	case TickByTickType_None:
		return "None"
	case TickByTickType_Last:
		return "Last"
	case TickByTickType_AllLast:
		return "AllLast"
	case TickByTickType_BidAsk:
		return "BidAsk"
	case TickByTickType_MidPoint:
		return "MidPoint"
	default:
		return fmt.Sprintf("unknown: %d", t)
	}
}
