/*
 * @Author: aztec
 * @Date: 2023-02-24 17:32:09
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binanceapi

const (
	OrderStatus_New             = "NEW"
	OrderStatus_Rejected        = "REJECTED"
	OrderStatus_Canceled        = "CANCELED"
	OrderStatus_PartiallyFilled = "PARTIALLY_FILLED"
	OrderStatus_Filled          = "FILLED"
)

// 外部通过设置这个回调来处理关键错误
var ErrorCallback func(e error)
