/*
- @Author: aztec
- @Date: 2024-03-12 09:19:22
- @Description: ibkr现货交易器。实现common.SpotTrader接口
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package ibkrtws

import (
	"bytes"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type SpotTrader struct {
	market    *SpotMarket
	ex        *Exchange
	logPrefix string

	// 余额
	baseBalance  *common.BalanceImpl
	quoteBalance *common.BalanceImpl

	// 订单
	tif      string                     // 订单的time in force
	orders   map[interface{}]*SpotOrder // clientId-order
	muOrders sync.RWMutex

	finished bool // 结束标志，用来退出某些循环
}

func (t *SpotTrader) Init(ex *Exchange, m *SpotMarket, tif string) {
	t.market = m
	t.ex = ex
	t.tif = tif
	t.orders = make(map[interface{}]*SpotOrder)
	t.logPrefix = fmt.Sprintf("%s-Trader-%s", logPrefix, m.inst.Id)
	t.finished = false

	// 获取balance指针
	t.baseBalance = ex.balanceMgr.FindBalance(t.market.BaseCurrency())
	t.quoteBalance = ex.balanceMgr.FindBalance(t.market.QuoteCurrency())

	// 清理finished orders
	go func() {
		for !t.finished {
			t.muOrders.Lock()
			for cid, o := range t.orders {
				if o.Finished {
					o.uninit()
					delete(t.orders, cid)
				}
			}
			t.muOrders.Unlock()
			time.Sleep(time.Second)
		}
	}()

	logInfo(logPrefix, "spot trader(%s) inited", m.inst.Id)
}
func (t *SpotTrader) Uninit() {
	t.finished = true
	t.market.Uninit()
	logInfo(logPrefix, "spot trader(%s) uninited", t.market.inst.Id)
}

// 实现common.OrderObserver
func (t *SpotTrader) OnDeal(deal common.Deal) {
	// 订单成交时，记录订单成交造成的权益临时变化
	if deal.O.GetDir() == common.OrderDir_Buy {
		t.baseBalance.RecordTempRights(deal.Amount, deal.UTime)
		t.quoteBalance.RecordTempRights(deal.Amount.Mul(deal.Price).Neg(), deal.UTime)
	} else if deal.O.GetDir() == common.OrderDir_Sell {
		t.baseBalance.RecordTempRights(deal.Amount.Neg(), deal.UTime)
		t.quoteBalance.RecordTempRights(deal.Amount.Mul(deal.Price), deal.UTime)
	}
}

// #region 实现 common.SpotTrader
func (t *SpotTrader) Market() common.CommonMarket {
	return t.market
}

func (t *SpotTrader) SpotMarket() common.SpotMarket {
	return t.market
}

func (t *SpotTrader) String() string {
	bb := bytes.Buffer{}
	bb.WriteString(t.market.String())
	bb.WriteString(fmt.Sprintf("\nspot trader:%s\n", t.market.inst.Id))
	bb.WriteString(fmt.Sprintf("base currency(%s): %v/%v\n", t.market.baseCcy, t.baseBalance.Available(), t.baseBalance.Rights()))
	bb.WriteString(fmt.Sprintf("quote currency(%s): %v/%v\n", t.market.quoteCcy, t.quoteBalance.Available(), t.quoteBalance.Rights()))

	t.muOrders.RLock()
	bb.WriteString(fmt.Sprintf("%d alive orders:\n", len(t.orders)))
	for _, o := range t.orders {
		bb.WriteString(o.String())
	}
	t.muOrders.RUnlock()
	return bb.String()
}

func (t *SpotTrader) Ready() bool {
	baseBalOk, _ := t.baseBalance.Ready()
	quoteBalOk, _ := t.quoteBalance.Ready()
	return t.market.Ready() && baseBalOk && quoteBalOk && t.ex.ready()
}

func (t *SpotTrader) UnreadyReason() string {
	if !t.market.Ready() {
		return t.market.UnreadyReason()
	}

	if ok, reason := t.baseBalance.Ready(); !ok {
		return fmt.Sprintf("base balance(%s) not ready: %s", t.baseBalance.Ccy(), reason)
	}

	if ok, reason := t.quoteBalance.Ready(); !ok {
		return fmt.Sprintf("quote balance(%s) not ready: %s", t.quoteBalance.Ccy(), reason)
	}

	if !t.ex.ready() {
		return t.ex.unreadyReason()
	}

	return ""
}

func (t *SpotTrader) BuyPriceRange() (min, max decimal.Decimal) {
	// 买入价格有最低限制
	buy1, _ := t.market.orderBook.Buy1()
	cc := t.market.contractConfig
	return decimal.Min(buy1.Sub(cc.MaxPriceDistValueToBestPrice), buy1.Mul(util.DecimalOne.Sub(cc.MaxPriceDistRatioToBestPrice))), decimal.NewFromInt(math.MaxInt32)
}

func (t *SpotTrader) SellPriceRange() (min, max decimal.Decimal) {
	// 卖出价格有最高限制
	sell1, _ := t.market.orderBook.Sell1()
	cc := t.market.contractConfig
	return decimal.Zero, decimal.Max(sell1.Add(cc.MaxPriceDistValueToBestPrice), sell1.Mul(util.DecimalOne.Add(cc.MaxPriceDistRatioToBestPrice)))
}

func (t *SpotTrader) MakeOrder(
	price,
	amount decimal.Decimal,
	dir common.OrderDir,
	makeOnly, reduceOnly bool,
	purpose string,
	obs common.OrderObserver) common.Order {
	if t.Ready() {
		o := new(SpotOrder)
		if o.init(t, price, amount, dir, t.tif, purpose) {
			t.muOrders.Lock()
			t.orders[o.CltOrderId] = o
			t.muOrders.Unlock()
			o.AddObserver(t)   // 先内部处理
			o.AddObserver(obs) // 再外部处理
			o.Go()
			return o
		} else {
			return nil
		}
	} else {
		logInfo(t.logPrefix, "trader not ready, can't Makeorder. reason=%s", t.UnreadyReason())
		time.Sleep(time.Second)
		return nil
	}
}

func (t *SpotTrader) Orders() []common.Order {
	orders := make([]common.Order, 0, len(t.orders))

	t.muOrders.Lock()
	for _, o := range t.orders {
		orders = append(orders, o)
	}
	t.muOrders.Unlock()

	return orders
}

func (t *SpotTrader) FeeTaker() decimal.Decimal {
	return decimal.Zero
}

func (t *SpotTrader) FeeMaker() decimal.Decimal {
	return decimal.Zero
}

func (t *SpotTrader) AvailableAmount(dir common.OrderDir, price decimal.Decimal) decimal.Decimal {
	if dir == common.OrderDir_Sell {
		rights := t.baseBalance.Rights()
		frozen := t.ex.getFrozenBalance(t.market.baseCcy)
		return rights.Sub(frozen)
	} else {
		rights := t.quoteBalance.Rights()
		frozen := t.ex.getFrozenBalance(t.market.quoteCcy)
		return rights.Sub(frozen)
	}
}

func (t *SpotTrader) BaseBalance() common.Balance {
	return t.baseBalance
}

func (t *SpotTrader) QuoteBalance() common.Balance {
	return t.quoteBalance
}

func (t *SpotTrader) AssetId() int {
	return 0
}
