/*
- @Author: aztec
- @Date: 2024-03-05 15:55:04
- @Description: 订单的“腿”
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "github.com/shopspring/decimal"

type OrderComboLeg struct {
	Price decimal.Decimal
}