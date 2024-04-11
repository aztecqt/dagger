/*
- @Author: aztec
- @Date: 2024-03-05 10:14:33
- @Description: 价格最低增量
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "github.com/shopspring/decimal"

type PriceIncrement struct {
	LowEdge   decimal.Decimal
	Increment decimal.Decimal
}
