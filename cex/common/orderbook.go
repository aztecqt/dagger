/*
 * @Author: aztec
 * @Date: 2022-03-30 09:44:46
 * @LastEditors: aztec
 * @LastEditTime: 2023-01-03 11:45:10
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\common\orderbook.go
 * @Description: 订单簿
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package common

import (
	"bytes"
	"fmt"
	"sync"

	"aztecqt/dagger/util"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/shopspring/decimal"
)

type Orderbook struct {
	Asks  *treemap.Map
	Bids  *treemap.Map
	mu    sync.Mutex
	buy1  decimal.Decimal
	sell1 decimal.Decimal
}

func (ob *Orderbook) Empty() bool {
	return ob.Asks.Empty() || ob.Bids.Empty()
}

func (ob *Orderbook) Lock() {
	ob.mu.Lock()
}

func (ob *Orderbook) Unlock() {
	ob.mu.Unlock()
}

func (ob *Orderbook) Init() {
	ob.Lock()
	defer ob.Unlock()
	ob.Asks = util.NewDecimalTreeMap()         // 由小到大排列，卖1放在第1个
	ob.Bids = util.NewDecimalTreeMapInverted() // 由大到小排列，买1放在第1个
}

// 最高买价
func (ob *Orderbook) Buy1() decimal.Decimal {
	return ob.buy1
}

// 最低卖价
func (ob *Orderbook) Sell1() decimal.Decimal {
	return ob.sell1
}

// 中间价
func (ob *Orderbook) MiddlePrice() decimal.Decimal {
	return ob.buy1.Add(ob.sell1).Div(decimal.NewFromInt(2))
}

// 根据吃单买入数量，计算吃单均价
func (ob *Orderbook) GetBuyPriceByAmount(amount decimal.Decimal) decimal.Decimal {
	ob.Lock()
	defer ob.Unlock()

	amountAcc := decimal.Zero
	amountPriceAcc := decimal.Zero
	it := ob.Asks.Iterator()
	for it.Next() {
		p := it.Key().(decimal.Decimal)
		a := it.Value().(decimal.Decimal)
		if amountAcc.Add(a).LessThan(amount) {
			amountAcc = amountAcc.Add(a)
			amountPriceAcc = amountPriceAcc.Add(a.Mul(p))
		} else {
			da := amount.Sub(amountAcc)
			amountAcc = amountAcc.Add(da)
			amountPriceAcc = amountPriceAcc.Add(da.Mul(p))
			break
		}
	}

	if amountAcc.IsPositive() {
		price := amountPriceAcc.Div(amountAcc)
		return price
	} else {
		return ob.sell1
	}
}

// 根据吃单卖出数量，计算吃单均价
func (ob *Orderbook) GetSellPriceByAmount(amount decimal.Decimal) decimal.Decimal {
	ob.Lock()
	defer ob.Unlock()

	amountAcc := decimal.Zero
	amountPriceAcc := decimal.Zero
	it := ob.Bids.Iterator()
	for it.Next() {
		p := it.Key().(decimal.Decimal)
		a := it.Value().(decimal.Decimal)
		if amountAcc.Add(a).LessThan(amount) {
			amountAcc = amountAcc.Add(a)
			amountPriceAcc = amountPriceAcc.Add(a.Mul(p))
		} else {
			da := amount.Sub(amountAcc)
			amountAcc = amountAcc.Add(da)
			amountPriceAcc = amountPriceAcc.Add(da.Mul(p))
			break
		}
	}

	if amountAcc.IsPositive() {
		price := amountPriceAcc.Div(amountAcc)
		return price
	} else {
		return ob.buy1
	}
}

// 根据最大滑点，计算最大吃单买入数量
func (ob *Orderbook) MaxBuyAmountBySlipPoint(maxSp decimal.Decimal) decimal.Decimal {
	ob.Lock()
	defer ob.Unlock()

	startPx := ob.sell1
	it := ob.Asks.Iterator()
	amount := decimal.Zero
	for it.Next() {
		p := it.Key().(decimal.Decimal)
		a := it.Value().(decimal.Decimal)
		sp := p.Sub(startPx).Div(startPx).Sub(decimal.NewFromInt(1))
		if sp.LessThanOrEqual(maxSp) {
			amount = amount.Add(a)
		} else {
			if amount.IsZero() {
				amount = amount.Add(a) // 至少给一档
			}
			break
		}
	}
	return amount
}

// 根据最大滑点，计算最大吃单卖出数量
func (ob *Orderbook) MaxSellAmountBySlipPoint(maxSp decimal.Decimal) decimal.Decimal {
	ob.Lock()
	defer ob.Unlock()

	startPx := ob.buy1
	it := ob.Bids.Iterator()
	amount := decimal.Zero
	for it.Next() {
		p := it.Key().(decimal.Decimal)
		a := it.Value().(decimal.Decimal)
		sp := startPx.Sub(p).Div(startPx).Sub(decimal.NewFromInt(1))
		if sp.LessThanOrEqual(maxSp) {
			amount = amount.Add(a)
		} else {
			break
		}
	}
	return amount
}

// 计算平均挂单密度，也就是在一定价格跨度内有多少量
func (ob *Orderbook) Density(r float64) float64 {
	ob.Lock()
	defer ob.Unlock()

	sd0, _ := ob.Asks.Min()
	sd1, _ := ob.Asks.Max()
	bd0, _ := ob.Bids.Max()
	bd1, _ := ob.Bids.Min()

	if sd0 != nil && sd1 != nil && bd0 != nil && bd1 != nil {
		s0 := sd0.(decimal.Decimal).InexactFloat64()
		s1 := sd1.(decimal.Decimal).InexactFloat64()
		b0 := bd0.(decimal.Decimal).InexactFloat64()
		b1 := bd1.(decimal.Decimal).InexactFloat64()

		priceRangeRatio := 0.0
		priceRangeRatio += (s1 - s0) / s0
		priceRangeRatio += (b1 - b0) / b0

		amountTotal := 0.0
		it := ob.Asks.Iterator()
		for it.Next() {
			amountTotal += it.Value().(decimal.Decimal).InexactFloat64()
		}
		it = ob.Bids.Iterator()
		for it.Next() {
			amountTotal += it.Value().(decimal.Decimal).InexactFloat64()
		}

		if priceRangeRatio > 0 {
			return amountTotal / priceRangeRatio * r
		}
	}

	return 0
}

// 清空数据
func (ob *Orderbook) Clear() {
	ob.Lock()
	defer ob.Unlock()
	ob.Asks.Clear()
	ob.Bids.Clear()
}

// 更新数据
func (ob *Orderbook) UpdateAsk(price, amount decimal.Decimal) {
	ob.Lock()
	defer ob.Unlock()

	if amount.IsZero() {
		ob.Asks.Remove(price)
	} else {
		ob.Asks.Put(price, amount)
	}

	k, _ := ob.Asks.Min()
	if k == nil {
		ob.sell1 = decimal.Zero
	} else {
		ob.sell1 = k.(decimal.Decimal)
	}
}

func (ob *Orderbook) UpdateBids(price, amount decimal.Decimal) {
	ob.Lock()
	defer ob.Unlock()

	if amount.IsZero() {
		ob.Bids.Remove(price)
	} else {
		ob.Bids.Put(price, amount)
	}

	k, _ := ob.Bids.Min() // 注意，bids是个反向map
	if k == nil {
		ob.buy1 = decimal.Zero
	} else {
		ob.buy1 = k.(decimal.Decimal)
	}
}

// asks：px,sz,px,sz
func (ob *Orderbook) Rebuild(asks, bids []decimal.Decimal) {
	ob.Lock()
	ob.Unlock()

	ob.Asks.Clear()
	ob.Bids.Clear()
	for i := 0; i < len(asks)-1; i += 2 {
		px := asks[i]
		sz := asks[i+1]
		ob.Asks.Put(px, sz)
	}

	for i := 0; i < len(bids)-1; i += 2 {
		px := bids[i]
		sz := bids[i+1]
		ob.Bids.Put(px, sz)
	}

	k, _ := ob.Asks.Min()
	if k == nil {
		ob.sell1 = decimal.Zero
	} else {
		ob.sell1 = k.(decimal.Decimal)
	}

	k, _ = ob.Bids.Min() // 注意，bids是个反向map
	if k == nil {
		ob.buy1 = decimal.Zero
	} else {
		ob.buy1 = k.(decimal.Decimal)
	}
}

// 转换为字符串
func (ob *Orderbook) String(length int) string {
	ob.Lock()
	askPrices := ob.Asks.Keys()
	askAmounts := ob.Asks.Values()
	bidPrices := ob.Bids.Keys()
	bidAmounts := ob.Bids.Values()
	ob.Unlock()

	if length > len(askPrices) {
		length = len(askPrices)
	}

	if length > len(bidPrices) {
		length = len(bidPrices)
	}

	bb := bytes.Buffer{}

	bb.WriteString("sell:")
	for i := 0; i < length; i++ {
		index := i
		askPrice := askPrices[index].(decimal.Decimal)
		askAmount := askAmounts[index].(decimal.Decimal)

		if askAmount.IsPositive() {
			bb.WriteString(fmt.Sprintf("[%s %s]", askPrice.String(), askAmount.String()))
		}
	}

	bb.WriteString("\nbuy:")

	for i := 0; i < length; i++ {
		index := i
		bidPrice := bidPrices[index].(decimal.Decimal)
		bidAmount := bidAmounts[index].(decimal.Decimal)

		if bidAmount.IsPositive() {
			bb.WriteString(fmt.Sprintf("[%s %s]", bidPrice.String(), bidAmount.String()))
		}
	}

	bb.WriteString("\n")

	return bb.String()
}
