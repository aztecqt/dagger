/*
 * @Author: aztec
 * @Date: 2022-04-03 19:39:28
 * @LastEditors: aztec
 * @LastEditTime: 2023-02-24 11:39:28
 * @FilePath: \dagger\cex\okexv5\contract_order.go
 * @Description: okexv5的合约订单
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"github.com/aztecqt/dagger/cex/common"

	"github.com/shopspring/decimal"
)

type ContractOrder struct {
	CommonOrder
	trader *FutureTrader
}

func (o *ContractOrder) Init(
	trader *FutureTrader,
	price, amount decimal.Decimal,
	dir common.OrderDir,
	makeOnly, reduceOnly bool,
	purpose string) bool {
	o.trader = trader
	if o.CommonOrder.Init(trader, trader.exchange.instrumentMgr, trader.market.instId, price, amount, dir, makeOnly, reduceOnly, purpose) {
		o.CommonOrder.getPosSide = o.getPosSide
		o.CommonOrder.tradeMode = o.tradeMode
		return true
	} else {
		return false
	}
}

// #region 覆盖CommonOrder
func (o *ContractOrder) getPosSide() string {
	posSide := "long"
	if o.Dir == common.OrderDir_Buy {
		if o.trader.pos.Short().GreaterThanOrEqual(o.Size) || o.ReduceOnly {
			posSide = "short" // 买操作，空仓足够，平空/只允许平仓
		} else {
			posSide = "long" // 否则开多
		}
	} else {
		if o.trader.pos.Long().GreaterThanOrEqual(o.Size) || o.ReduceOnly {
			posSide = "long" // 卖操作，多仓足够，平多/只允许平仓
		} else {
			posSide = "short" // 否则开空
		}
	}
	return posSide
}

func (o *ContractOrder) tradeMode() string {
	return "cross"
}

// #endregion 覆盖CommonOrder
