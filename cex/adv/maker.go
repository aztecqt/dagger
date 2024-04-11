/*
 * @Author: aztec
 * @Date: 2022-04-07 09:44:45
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-04-09 16:22:57
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\adv\maker.go
 * @Description: 自动化的动态（make）订单，目的是固化一些常用操作，简化策略代码
 * 外部可以随时指定其价格和数量
 * 内部实现订单的量价调整：
 * 当订单价格与指定价格相差超过一个设定的阈值后，撤单重新挂单或者调整订单价格
 * 当订单剩余数量不足指定数量的一半时，撤单重新挂单
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package adv

import (
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type MakerOrderDeal struct {
	Deal     common.Deal
	UserData interface{}
}

type OnMakerOrderDeal func(deal MakerOrderDeal)

type Maker struct {
	logPrefix        string
	mu               sync.Mutex
	autoUpdateTicker time.Ticker // 自更新

	O                 common.Order
	trader            common.CommonTrader
	price             decimal.Decimal
	size              decimal.Decimal
	dir               common.OrderDir
	Usderdata         interface{}
	maxPriceDeviation float64 // 价格偏差度上限
	maxSizeDeviation  float64 // 未成交数量偏差上限
	makeOnly          bool    // 只挂单
	reduceOnly        bool    // 只减仓
	enableModify      bool    // 是否允许订单修改（在交易所支持的前提下）
	autoReborn        bool    // 订单结束后，是否自动创建新订单
	running           bool
	purpose           string

	chStop chan bool
	fnDeal OnMakerOrderDeal // 成交回调
}

func (d *Maker) String() string {
	return fmt.Sprintf("maker: [purpose:%v price=%v, size=%v, dir=%v, O=%v]", d.purpose, d.price, d.size, d.dir, d.O)
}

func (d *Maker) Init(
	trader common.CommonTrader,
	makeonly bool,
	autoReborn bool,
	enableModify bool,
	maxPriceDeviation float64,
	maxSizeDeviation float64,
	purpose string) {
	d.logPrefix = fmt.Sprintf("maker-%s-%v", trader.Market().Type(), purpose)
	d.trader = trader
	d.makeOnly = makeonly
	d.autoReborn = autoReborn
	d.enableModify = enableModify
	d.purpose = purpose
	d.maxPriceDeviation = maxPriceDeviation
	d.maxSizeDeviation = maxSizeDeviation
	d.chStop = make(chan bool, 1)
	d.fnDeal = nil
	d.autoUpdateTicker = *time.NewTicker(time.Millisecond * 100)
}

func (d *Maker) Modify(price, size decimal.Decimal, dir common.OrderDir, reduceOnly bool) {
	d.price = price
	d.size = size
	d.dir = dir
	d.reduceOnly = reduceOnly
	d.updateOrder(true)
}

func (d *Maker) ModifyWithoutOrderModify(price, size decimal.Decimal, dir common.OrderDir, reduceOnly bool) {
	d.price = price
	d.size = size
	d.dir = dir
	d.reduceOnly = reduceOnly

	temp := d.enableModify
	d.enableModify = false
	d.updateOrder(true)
	d.enableModify = temp
}

func (d *Maker) Cancel() {
	d.Modify(decimal.Zero, decimal.Zero, common.OrderDir_None, false)
}

func (d *Maker) Go() {
	d.running = true
	go d.update()
}

func (d *Maker) Stop() {
	if d.running {
		d.chStop <- true
	}
}

func (d *Maker) SetDealFn(fn OnMakerOrderDeal) {
	d.fnDeal = fn
}

func (d *Maker) update() {
	d.running = true
	for {
		select {
		case <-d.autoUpdateTicker.C:
			d.updateOrder(d.autoReborn)
		case <-d.chStop:
			d.running = false
		}

		if !d.running {
			break
		}
	}
}

// 实现common.OrderObserver
func (d *Maker) OnDeal(deal common.Deal) {
	id, cid := deal.O.GetID()
	logger.LogInfo(d.logPrefix, "order dealing, cid=%s/%s, dir=%s, price=%v, amount=%v", id, cid, common.OrderDir2Str(deal.O.GetDir()), deal.Price, deal.Amount)

	// 成交回调
	if d.fnDeal != nil {
		d.fnDeal(MakerOrderDeal{Deal: deal, UserData: d.Usderdata})
	}
}

// 暂停、量价为0等情况，关闭订单
// 否则执行订单修改逻辑
func (d *Maker) updateOrder(reborn bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	defer d.autoUpdateTicker.Reset(time.Millisecond * 10)

	if d.O != nil && d.O.IsFinished() {
		if d.O.HasFatalError() {
			time.Sleep(time.Second)
		}

		d.O = nil
	}

	px := d.trader.Market().AlignPrice(d.price, d.dir, d.makeOnly)
	sz := d.trader.Market().AlignSize(d.size) // 注意！这里的size仅指未成交数量，不包括已成交数量
	if !px.IsPositive() || !sz.IsPositive() || d.dir == common.OrderDir_None || !common.PriceInRange(px, d.dir, d.trader) {
		if d.O != nil {
			d.O.Cancel()
		}
	} else if reborn {
		if d.O == nil {
			// 创建订单
			d.O = d.trader.MakeOrder(px, sz, d.dir, d.makeOnly, d.reduceOnly, d.purpose, d)
			if d.O == nil {
				time.Sleep(time.Second) // 订单创建未通过本地验证
			}
		} else {
			priceDv := util.DecimalDeviationAbs(d.O.GetPrice(), px).InexactFloat64()
			sizeDv := util.DecimalDeviationAbs(d.O.GetUnfilled(), sz).InexactFloat64()
			if d.dir != d.O.GetDir() {
				d.O.Cancel()
			} else if priceDv > d.maxPriceDeviation || sizeDv > d.maxSizeDeviation {
				/*logger.LogInfo(
				d.logPrefix,
				"do_price:%v, price:%v, do_size:%v, size:%v, need cancel/modify",
				px,
				d.O.GetPrice(),
				sz,
				d.O.GetUnfilled())*/

				if d.enableModify && d.O.IsSupportModify() {
					priceNeedModify := priceDv > d.maxPriceDeviation
					sizeNeedModify := sizeDv > d.maxSizeDeviation
					if priceNeedModify && !sizeNeedModify {
						d.O.Modify(px, decimal.Zero)
					} else if !priceNeedModify && sizeNeedModify {
						d.O.Modify(decimal.Zero, sz.Add(d.O.GetFilled()))
					} else if priceNeedModify && sizeNeedModify {
						d.O.Modify(px, sz.Add(d.O.GetFilled()))
					}
				} else {
					d.O.Cancel()
				}
			}
		}
	}
}
