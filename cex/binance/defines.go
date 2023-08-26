/*
 * @Author: aztec
 * @Date: 2022-10-21 10:41:39
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binance

import (
	"fmt"
	"time"

	"aztecqt/dagger/api/binanceapi"
	"github.com/shopspring/decimal"
)

// 资产账户ID
const (
	AssetId_Fund = iota
	AssetId_Spot
	AssetId_Margin
	AssetId_Contract
)

// 订单快照
// 这个对象用于更新订单
// rest的订单反馈，也可以生成这个对象
type OrderSnapshot struct {
	Source        string
	OrderID       int64
	ClientOrderID string
	StratergyId   int
	Status        string
	UpdateTime    time.Time
	LocalTime     time.Time
	Price         decimal.Decimal
	Size          decimal.Decimal
	FilledSize    decimal.Decimal
	FillingSize   decimal.Decimal
	FillingPrice  decimal.Decimal
}

func (o *OrderSnapshot) String() string {
	return fmt.Sprintf("from:%s id: %d, cid:%s, status:%s, ts: %d, px: %v, sz: %v, filled: %v, fillingPx: %v, fillingSz: %v",
		o.Source,
		o.OrderID,
		o.ClientOrderID,
		o.Status,
		o.UpdateTime.UnixMilli(),
		o.Price,
		o.Size,
		o.FilledSize,
		o.FillingSize,
		o.FillingPrice)
}

func NewOrderSnapShotFromRestResponse(resp binanceapi.GetOrderResponse) OrderSnapshot {
	os := OrderSnapshot{}
	os.Source = "rest"
	os.OrderID = resp.OrderId
	os.ClientOrderID = resp.ClientOrderID
	os.Status = resp.Status
	os.UpdateTime = time.UnixMilli(resp.RefreshTimestamp)
	os.LocalTime = resp.LocalTime
	os.Price = resp.Price
	os.Size = resp.Size
	os.FilledSize = resp.FilledSize
	os.FillingSize = decimal.Zero
	os.FillingPrice = decimal.Zero
	return os
}

func NewOrderSnapshotFromWsResponse(resp binanceapi.WSPayload_OrderUpdate) OrderSnapshot {
	os := OrderSnapshot{}
	os.Source = "ws"
	os.OrderID = resp.OrderID
	if len(resp.OrignClientOrderId) > 0 {
		os.ClientOrderID = resp.OrignClientOrderId
	} else {
		os.ClientOrderID = resp.ClientOrderID
	}
	os.StratergyId = resp.StratergyId
	os.Status = resp.Status
	os.UpdateTime = time.UnixMilli(resp.TimeStamp)
	os.LocalTime = resp.LocalTime
	os.Price = resp.Price
	os.Size = resp.Size
	os.FilledSize = resp.FilledSize
	os.FillingSize = resp.FillingSize
	os.FillingPrice = resp.FillingPrice
	return os
}

func NewOrderSnapshot(
	id int64,
	timestamp int64,
	localTime time.Time,
	clientId, status string,
	px, sz, filled, fillingSz, fillingPx decimal.Decimal) OrderSnapshot {
	os := OrderSnapshot{}
	os.OrderID = id
	os.ClientOrderID = clientId
	os.Status = status
	os.UpdateTime = time.UnixMilli(timestamp)
	os.LocalTime = localTime
	os.Price = px
	os.Size = sz
	os.FilledSize = filled
	os.FillingSize = fillingSz
	os.FillingPrice = fillingPx
	return os
}
