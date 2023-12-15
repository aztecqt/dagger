/*
 * @Author: aztec
 * @Date: 2022-05-07
 * @LastEditors: aztec
 * @LastEditTime: 2023-02-24 11:30:03
 * @FilePath: \dagger\cex\adv\taker.go
 * @Description: 对冲吃单任务
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package adv

import (
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

type TakerDeal struct {
	T        *Taker
	Deal     common.Deal
	Finished bool // 单独记录一个结束状态，否则外部调用T.Finish()函数，可能会因为Channel处理延迟，导致多次返回true的情况
	UserData interface{}
}

type OnTakerDeal func(tkDeal TakerDeal)
type OnTakerFinish func()

var TakerTaskIndex int

type Taker struct {
	mu         sync.Mutex
	index      int
	logPrefix  string
	purpose    string
	trader     common.CommonTrader
	amount     decimal.Decimal
	dir        common.OrderDir
	reduceOnly bool
	userdata   interface{}

	dealed         decimal.Decimal
	dealedMulPrice decimal.Decimal
	O              common.Order
	startTime      time.Time

	// 订单错误次数
	orderErrorCount int

	// 成交回调
	fnDeal OnTakerDeal

	// finish回调
	fnFinish OnTakerFinish

	chStop chan int
}

func (t *Taker) Init(
	trader common.CommonTrader,
	amount decimal.Decimal,
	dir common.OrderDir,
	reduceOnly bool,
	purpose string,
	userdata interface{},
) {
	t.index = TakerTaskIndex
	TakerTaskIndex++
	t.logPrefix = fmt.Sprintf("TakerTask-%s-%d-%s", trader.Market().Type(), t.index, purpose)
	t.trader = trader
	t.amount = amount
	t.dir = dir
	t.reduceOnly = reduceOnly
	t.purpose = purpose
	t.userdata = userdata
	t.startTime = time.Now()
	t.chStop = make(chan int)
	t.fnDeal = nil
	t.fnFinish = nil

	logger.LogInfo(t.logPrefix, "task begin: %s", t.String())
}

func (t *Taker) Go() {
	t.updateOrder() // 先更新一次，以保证立即下单
	go t.update()
}

func (t *Taker) Stop() {
	t.chStop <- 0
}

func (t *Taker) String() string {
	return fmt.Sprintf("[amount:%v, dir:%s, reduceOnly:%v, purpose:%s, startTime:%s]",
		t.amount,
		common.OrderDir2Str(t.dir),
		t.reduceOnly,
		t.purpose,
		t.startTime.String())
}

func (t *Taker) DealPrice() decimal.Decimal {
	if t.dealedMulPrice.IsZero() {
		return decimal.Zero
	} else {
		return t.dealedMulPrice.Div(t.dealed)
	}
}

func (t *Taker) Dealed() decimal.Decimal {
	return t.dealed
}

// 订单错误次数过多，或者成交量足够时，认为结束
func (t *Taker) Finished() bool {
	if t.orderErrorCount >= 3 {
		return true
	}

	undealed := t.amount.Sub(t.dealed)
	return undealed.LessThan(t.trader.Market().MinSize())
}

func (t *Taker) SetDealFn(fn OnTakerDeal) {
	t.fnDeal = fn
}

func (t *Taker) SetFinishFn(fn OnTakerFinish) {
	t.fnFinish = fn
}

// 实现common.OrderObserver
func (t *Taker) OnDeal(deal common.Deal) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 订单成交
	id, cid := deal.O.GetID()
	logger.LogInfo(t.logPrefix, "order dealing, cid=%s/%s, dir=%s, price=%v, amount=%v", id, cid, common.OrderDir2Str(deal.O.GetDir()), deal.Price, deal.Amount)

	// 计算成交价格
	t.dealed = t.dealed.Add(deal.Amount)
	t.dealedMulPrice = t.dealedMulPrice.Add(deal.Amount.Mul(deal.Price))

	if t.Finished() {
		ms := float64(time.Now().UnixMicro()-t.startTime.UnixMicro()) / 1000.0
		logger.LogInfo(t.logPrefix, "task finished, time cost:%.1f ms", ms)
	}

	// 回调外部
	if t.fnDeal != nil {
		t.fnDeal(TakerDeal{T: t, Deal: deal, Finished: t.Finished(), UserData: t.userdata})
	}
}

func (t *Taker) updateOrder() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.O != nil && t.O.IsFinished() {
		if t.O.HasFatalError() {
			t.orderErrorCount++
		}
		t.O = nil
	}

	if !t.Finished() {
		if t.O == nil {
			// 创建订单
			price := t.trader.Market().OrderBook().Buy1().Mul(decimal.NewFromFloat(0.99))
			if t.dir == common.OrderDir_Buy {
				price = t.trader.Market().OrderBook().Sell1().Mul(decimal.NewFromFloat(1.01))
			}
			size := t.amount.Sub(t.dealed)
			alignedPrice := t.trader.Market().AlignPrice(price, t.dir, false)
			alignedSize := t.trader.Market().AlignSize(size)
			t.O = t.trader.MakeOrder(alignedPrice, alignedSize, t.dir, false, t.reduceOnly, t.purpose, t)
			if t.O == nil {
				t.orderErrorCount++ // 订单创建失败
			}
		} else {
			needCancel := false
			if t.dir == common.OrderDir_Buy && t.O.GetPrice().LessThanOrEqual(t.trader.Market().OrderBook().Sell1()) {
				needCancel = true
			}

			if t.dir == common.OrderDir_Sell && t.O.GetPrice().GreaterThanOrEqual(t.trader.Market().OrderBook().Buy1()) {
				needCancel = true
			}

			if time.Now().Unix()-t.O.GetBornTime().Unix() > 10 {
				needCancel = true
			}

			if needCancel {
				t.O.Cancel()
			}
		}
	} else {
		if t.fnFinish != nil {
			t.fnFinish()
		}
	}
}

func (t *Taker) update() {
	tm := time.NewTicker(time.Millisecond * 10)

	for {
		select {
		case <-tm.C:
			t.updateOrder()
		case <-t.chStop:
			return
		}
	}
}
