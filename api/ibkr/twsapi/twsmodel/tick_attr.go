/*
- @Author: aztec
- @Date: 2024-03-01 15:58:26
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

type TickAttribute struct {
	CanAutoExecute bool
	PastLimit      bool
	PreOpen        bool
	Unreported     bool
	BidPastLow     bool
	AskPastHigh    bool
}
