/*
 * @Author: aztec
 * @Date: 2022-04-02 10:03:20
 * @LastEditors: aztec
 * @LastEditTime: 2023-02-20 15:22:53
 * @FilePath: \dagger\cex\okexv5\defines.go
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"fmt"
	"time"

	"aztecqt/dagger/api/okexv5api"
	"aztecqt/dagger/util"

	"github.com/shopspring/decimal"
)

// 订单快照，用于订单刷新
type OrderSnapshot struct {
	localTime  time.Time
	id         int64
	clientId   string
	tag        string
	price      decimal.Decimal
	size       decimal.Decimal
	filled     decimal.Decimal
	avgPrice   decimal.Decimal
	status     string
	updateTime time.Time
	source     string
}

func (os *OrderSnapshot) Parse(resp okexv5api.OrderResp, source string) {
	os.source = source
	os.id = util.String2Int64Panic(resp.OrderId)
	os.clientId = resp.ClientOrderId
	os.tag = resp.Tag
	os.price = util.String2DecimalPanicUnless(resp.Price, "")
	os.size = util.String2DecimalPanic(resp.Size)
	os.filled = util.String2DecimalPanic(resp.AccFillSize)
	os.avgPrice = util.String2DecimalPanicUnless(resp.AvgPrice, "")
	os.status = resp.Status
	os.updateTime = util.ConvetUnix13StrToTimePanic(resp.UTime)
}

func (os *OrderSnapshot) String() string {
	return fmt.Sprintf(
		"(from %s)[id:%d clientId:%s price:%v size:%v filled:%v avgPrice:%v status:%s uTime:%v]",
		os.source,
		os.id,
		os.clientId,
		os.price,
		os.size,
		os.filled,
		os.avgPrice,
		os.status,
		os.updateTime.UnixMilli())
}
