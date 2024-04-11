/*
 * @Author: aztec
 * @Date: 2022-04-20 09:36:11
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-03-12 16:43:48
 * @FilePath: \dagger\cex\okexv5\spot_order.go
 * @Description:okexv5的现货订单
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5

import (
	"github.com/aztecqt/dagger/cex/common"

	"github.com/shopspring/decimal"
)

type SpotOrder struct {
	CommonOrder
	trader *SpotTrader
}

func (o *SpotOrder) Init(
	trader *SpotTrader,
	price, amount decimal.Decimal,
	dir common.OrderDir,
	makeOnly bool,
	purpose string) bool {
	o.trader = trader
	o.CltOrderId = NewClientOrderId(o.Purpose)
	if o.CommonOrder.Init(trader, trader.ex.instrumentMgr, trader.market.instId, price, amount, dir, makeOnly, false, purpose) {
		o.CommonOrder.getPosSide = o.getPosSide
		o.CommonOrder.tradeMode = o.tradeMode
		return true
	} else {
		return false
	}
}

// #region 提供给CommonOrder
func (o *SpotOrder) getPosSide() string {
	return ""
}

func (o *SpotOrder) tradeMode() string {
	return string(o.trader.ex.excfg.SpotTradeMode)
}

// #endregion 提供给CommonOrder
