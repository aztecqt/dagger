/*
 * @Author: aztec
 * @Date: 2022-04-02 10:03:20
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-10-03 11:04:04
 * @FilePath: \dagger\cex\okexv5\defines.go
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"

	"github.com/shopspring/decimal"
)

// 交易所配置
type ExchangeConfig struct {
	// 是否从ticker来生成Depth数据。true则不订阅depth，而是ticker
	DepthFromTicker bool `json:"depth_from_ticker"`

	// 是否通过rest拉取ticker。是的话，由exchange统一拉取所有ticker，否则各个交易对自行订阅
	TickerFromRest bool `json:"ticker_from_rest"`
}

func (c *ExchangeConfig) init() {
	c.DepthFromTicker = true
}

// 订单快照，用于订单刷新
type orderSnapshot struct {
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

func (os *orderSnapshot) Parse(resp okexv5api.OrderResp, source string) {
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

func (os *orderSnapshot) String() string {
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
