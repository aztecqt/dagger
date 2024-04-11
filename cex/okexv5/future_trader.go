/*
 * @Author: aztec
 * @Date: 2022-04-01 14:14:14
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-04-09 15:53:13
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\okexv5\future_trader.go
 * @Description:合约交易器okexv5版本。实现common.FutureTrader接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5

import (
	"bytes"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

type FutureTrader struct {
	market    *FutureMarket
	exchange  *Exchange
	logPrefix string
	orderTag  string

	// 仓位
	pos *common.PositionImpl

	balance *common.BalanceImpl // 保证金权益
	lever   int                 // 杠杆倍率

	orders   map[string]*ContractOrder // clientId-order
	muOrders sync.RWMutex

	errorlock bool // 出现异常时，锁定订单创建等关键操作
	finished  bool // 结束标志，用来退出某些循环
}

func (t *FutureTrader) Init(ex *Exchange, orderTag string, m *FutureMarket, lever int) {
	t.market = m
	t.exchange = ex
	t.orderTag = orderTag
	t.orders = make(map[string]*ContractOrder)
	t.logPrefix = fmt.Sprintf("%s-Trader-%s", logPrefix, m.instId)
	t.finished = false

	// 设置杠杆倍率
	for {
		// 组合保证金模式+合约全仓模式，不需要、也无法设置杠杆倍率
		// 此时将杠杆率视作1来处理
		if t.exchange.excfg.AccLevel == okexv5api.AccLevel_Portfolio && t.exchange.excfg.ContractTradeMode == okexv5api.TradeMode_Cross {
			logger.LogImportant(t.logPrefix, "lever set to 1(in portfolio mode)")
			t.lever = 1
			break
		}

		if resp, err := okexv5api.GetLeverage(m.instId); err == nil && resp.Code == "0" {
			if resp.Data[0].Lever == lever {
				t.lever = lever
				logger.LogImportant(t.logPrefix, "lever is already %d", lever)
				break
			}
		}

		resp, err := okexv5api.SetLeverage(m.instId, lever)
		if err == nil && resp.Code == "0" {
			t.lever = resp.Data[0].Lever
			logger.LogImportant(t.logPrefix, "lever set to %d", resp.Data[0].Lever)
			break
		} else {
			if err != nil {
				logger.LogImportant(t.logPrefix, "set leverage failed: %s", err.Error())
			} else {
				logger.LogImportant(t.logPrefix, "set leverage failed: %s", resp.Msg)
			}
		}

		time.Sleep(time.Second)
	}

	// 获取balance指针
	t.balance = ex.balanceMgr.FindBalance(m.SettlementCurrency())

	// 获取positoin指针
	t.pos = ex.findPosition(m.instId)

	// 订阅order信息
	ex.RegOrderSnapshot(m.instId, func(os orderSnapshot) {
		var o *ContractOrder = nil
		var ok bool = false

		if len(os.tag) > 0 && os.tag != orderTag {
			t.errorlock = true
			logger.LogPanic(t.logPrefix, "found order from other stratergy(%s)!", os.tag)
		}

		t.muOrders.RLock()
		o, ok = t.orders[os.clientId]
		t.muOrders.RUnlock()

		if ok {
			o.onSnapshot(os)
		}
	})

	// 清理finished orders
	go func() {
		for !t.finished {
			t.muOrders.Lock()
			for cid, o := range t.orders {
				if o.IsFinished() {
					delete(t.orders, cid)
				}
			}
			t.muOrders.Unlock()
			time.Sleep(time.Second)
		}
	}()

	logger.LogImportant(logPrefix, "future trader(%s) inited", m.instId)
}

func (t *FutureTrader) Uninit() {
	t.finished = true
	t.exchange.UnregOrderSnapshot(t.market.instId)
	t.market.Uninit()
	logger.LogImportant(logPrefix, "future trader(%s) uninited", t.market.instId)
}

// 实现common.OrderObserver
func (t *FutureTrader) OnDeal(deal common.Deal) {
	// 记录因为成交而带来的仓位变化
	o := deal.O.(*CommonOrder)
	if o.posSide == "long" {
		if o.Dir == common.OrderDir_Buy {
			// 开多
			t.pos.RecordTempLong(deal.Amount, deal.UTime)
		} else if o.Dir == common.OrderDir_Sell {
			// 平多
			t.pos.RecordTempLong(deal.Amount.Neg(), deal.UTime)
		}
	} else if o.posSide == "short" {
		if o.Dir == common.OrderDir_Buy {
			// 平空
			t.pos.RecordTempShort(deal.Amount.Neg(), deal.UTime)
		} else if o.Dir == common.OrderDir_Sell {
			// 开空
			t.pos.RecordTempShort(deal.Amount, deal.UTime)
		}
	}
}

// #region 实现common.FutureTrader
func (t *FutureTrader) Market() common.CommonMarket {
	return t.market
}

func (t *FutureTrader) FutureMarket() common.FutureMarket {
	return t.market
}

func (t *FutureTrader) String() string {
	bb := bytes.Buffer{}
	bb.WriteString(t.market.String())
	bb.WriteString(fmt.Sprintf("\nfuture trader:%s\n", t.market.instId))
	bb.WriteString(fmt.Sprintf("balance of deposit(%s): %s\n", t.market.SettlementCurrency(), t.balance.Rights().String()))
	bb.WriteString(fmt.Sprintf("position: long=%s, short=%s\n", t.pos.Long().String(), t.pos.Short().String()))

	t.muOrders.RLock()
	bb.WriteString(fmt.Sprintf("%d alive orders:\n", len(t.orders)))
	for _, o := range t.orders {
		bb.WriteString(o.String())
	}
	t.muOrders.RUnlock()

	return bb.String()
}

func (t *FutureTrader) Ready() bool {
	return t.market.Ready() && t.pos.Ready() && t.balance.Ready() && exchangeReady && !t.errorlock
}

func (t *FutureTrader) UnreadyReason() string {
	if !t.market.Ready() {
		return t.market.UnreadyReason()
	} else if !t.pos.Ready() {
		return "postion not ready"
	} else if t.balance.Ready() {
		return "balance not ready"
	} else if !exchangeReady {
		return "exchange not ready"
	} else if t.errorlock {
		return "locked by error"
	} else {
		return ""
	}
}

func (t *FutureTrader) BuyPriceRange() (min, max decimal.Decimal) {
	return decimal.Zero, decimal.NewFromInt(math.MaxInt32)
}

func (t *FutureTrader) SellPriceRange() (min, max decimal.Decimal) {
	return decimal.Zero, decimal.NewFromInt(math.MaxInt32)
}

func (t *FutureTrader) MakeOrder(
	price,
	amount decimal.Decimal,
	dir common.OrderDir,
	makeOnly, reduceOnly bool,
	purpose string,
	obs common.OrderObserver) common.Order {
	if t.Ready() {
		o := new(ContractOrder)
		if o.Init(t, price, amount, dir, makeOnly, reduceOnly, purpose) {
			t.muOrders.Lock()
			t.orders[o.CltOrderId.(string)] = o
			t.muOrders.Unlock()
			o.AddObserver(t)   // 先内部处理
			o.AddObserver(obs) // 再外部处理
			o.Go()
			return o
		} else {
			return nil
		}
	} else {
		logger.LogInfo(t.logPrefix, "trader not ready, can't Makeorder. reason=%s", t.UnreadyReason())
		time.Sleep(time.Second)
		return nil
	}
}

func (t *FutureTrader) Orders() []common.Order {
	orders := make([]common.Order, 0, len(t.orders))
	for _, o := range t.orders {
		orders = append(orders, o)
	}
	return orders
}

func (t *FutureTrader) FeeTaker() decimal.Decimal {
	return decimal.Zero
}

func (t *FutureTrader) FeeMaker() decimal.Decimal {
	return decimal.Zero
}

func (t *FutureTrader) AvailableAmount(dir common.OrderDir, price decimal.Decimal) decimal.Decimal {
	// 开仓时，可用数量以保证金计算
	// 反向合约（USD合约）为coin x price x lever / AmountValue
	// 正向合约（USDT合约）为usdt / price x lever / AmountValue
	// 平仓时，可用数量以剩余仓位计算（目前不考虑对向开仓，这样比较保守和简单）
	availableMargin := t.balance.Available().InexactFloat64()
	if !t.exchange.isSingleMarginMode() {
		// 注意，单币种保证金模式和跨币种/组合保证金模式的保证金计算方式不一样。
		// 前者为balance，后者则需要从api获取。特殊的：组合保证金模式的杠杆率以1计算（因为没有杠杆率的概念）
		if v, ok := t.exchange.getMaxAvailable(t.market.instId); ok {
			availableMargin = util.ValueIf(dir == common.OrderDir_Buy, v.AvailableBuy.InexactFloat64(), v.AvailableSell.InexactFloat64())
		} else {
			logger.LogImportant(t.logPrefix, "get max available from exchange failed")
			return decimal.Zero
		}
	}

	valueAmnt := t.FutureMarket().ValueAmount().InexactFloat64()

	if price.IsZero() {
		price = util.ValueIf(dir == common.OrderDir_Buy, t.market.orderBook.Sell1Price(), t.market.orderBook.Buy1Price())
	}
	px := price.InexactFloat64()

	available := decimal.Zero
	if t.market.IsUsdtContract() {
		available = decimal.NewFromFloat(availableMargin / px * float64(t.lever) / valueAmnt * 0.95) // 按保守估计
	} else {
		available = decimal.NewFromFloat(availableMargin * px * float64(t.lever) / valueAmnt * 0.95) // 按保守估计
	}

	available = t.exchange.instrumentMgr.AlignSize(t.market.instId, available)
	if dir == common.OrderDir_Buy {
		if t.pos.Short().IsPositive() {
			return t.pos.Short() // 平空
		} else {
			return available // 开多
		}
	} else if dir == common.OrderDir_Sell {
		if t.pos.Long().IsPositive() {
			return t.pos.Long() // 平多
		} else {
			return available // 开空
		}
	} else {
		return decimal.Zero
	}
}

func (t *FutureTrader) Lever() int {
	return t.lever
}

func (t *FutureTrader) Balance() common.Balance {
	return t.balance
}

func (t *FutureTrader) AssetId() int {
	return 0 // okex是统一账户
}

func (t *FutureTrader) Position() common.Position {
	return t.pos
}

// #endregion 实现common.FutureTrader
