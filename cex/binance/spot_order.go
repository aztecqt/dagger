/*
 * @Author: aztec
 * @Date: 2023-02-22 17:17:17
 * @Description: 币安的现货和合约相差较大，先尝试分开
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binance

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancespotapi"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type SpotOrder struct {
	common.OrderImpl

	canceling             bool // 是否正在取消(调试用)
	modifying             bool // 是否正在修改(调试用)
	refreshCount          int  // 刷新次数
	restRefreshErrorCount int  // rest调用错误次数

	// 刷新
	muRefresh        sync.Mutex
	tkRefreshTimeout *time.Ticker
	chRefreshImm     chan int
}

// 初始化
func (o *SpotOrder) Init(
	trader *SpotTrader,
	price, amount decimal.Decimal,
	dir common.OrderDir,
	makeOnly bool,
	purpose string) bool {
	return o.OrderImpl.Init(trader, trader.exchange.instrumentMgr, trader.Market().Type(), price, amount, dir, makeOnly, false, purpose)
}

func (o *SpotOrder) Go() {
	o.tkRefreshTimeout = time.NewTicker(time.Second * 10)
	go o.update()
}

// #region 实现common.Order
func (o *SpotOrder) GetExchangeName() string {
	return exchangeName
}

func (o *SpotOrder) String() string {
	return fmt.Sprintf("%s[frame:%d modifying:%v canceling:%v]", o.OrderImpl.String(), o.refreshCount, o.modifying, o.canceling)
}

func (o *SpotOrder) IsSupportModify() bool {
	return false
}

func (o *SpotOrder) Modify(newPrice, newSize decimal.Decimal) {
	logger.LogPanic(o.LogPrefix, "modify not supported")
}

func (o *SpotOrder) Cancel() {
	if !o.IsFinished() {
		go o.cancel()
	}
}

// #endregion

// #region 自身逻辑
// 创建订单
func (o *SpotOrder) create() {
	defer util.DefaultRecover()

	// 已经创建的订单不会再次被创建
	if o.OrderId > 0 {
		return
	}

	side := "BUY"
	if o.Dir == common.OrderDir_Sell {
		side = "SELL"
	}

	logger.LogInfo(o.LogPrefix, "creating [%s]", o.String())
	resp, err := binancespotapi.MakeOrder(o.InstId, side, "LIMIT", o.CltOrderId, o.Price, o.Size)
	if err == nil {
		if resp.Code == 0 && len(resp.Message) == 0 {
			if resp.OrderID > 0 {
				// 创建成功
				o.OrderId = resp.OrderID
				logger.LogInfo(o.LogPrefix, "create success, order id = %v", o.OrderId)
			} else {
				// 订单id缺失，应该是不会出现这种情况
				o.ErrMsg = "create success but missing order id"
				o.FatalError = true
				logger.LogImportant(o.LogPrefix, "create order error, missing order id ")
			}
		} else {
			// 订单创建失败
			o.ErrMsg = fmt.Sprintf("create failed, code=%d, msg=%s", resp.Code, resp.Message)
			o.FatalError = true
			logger.LogImportant(o.LogPrefix, "create order error: %s", o.ErrMsg)
		}
	} else {
		// 网络错误不代表订单未创建成功
		// 应该查询时返回“订单不存在”作为订单错误的触发条件
		logger.LogImportant(o.LogPrefix, "create order with rest error: %s", err.Error())
	}
}

// 取消订单
// 无论成功与否，都直接返回。逻辑层如果觉得仍有必要取消，再次调用即可
func (o *SpotOrder) cancel() {
	if !o.canceling {
		o.canceling = true
		defer util.DefaultRecover()
		defer func() {
			o.canceling = false
		}()

		logger.LogInfo(o.LogPrefix, "canceling [%s]", o.String())
		resp, err := binancespotapi.CancelOrder(o.InstId, 0, o.CltOrderId)
		if err == nil {
			if resp.Code != 0 || len(resp.Message) > 0 {
				o.ErrMsg = fmt.Sprintf("code:%d, msg:%s", resp.Code, resp.Message)
				logger.LogImportant(o.LogPrefix, "cancel order error: %s", o.ErrMsg)
				time.Sleep(time.Second)
			} else {
				logger.LogInfo(o.LogPrefix, "cancel responsed")
			}
		} else {
			logger.LogImportant(o.LogPrefix, "cancel order with rest error: %s", err.Error())
			time.Sleep(time.Second)
		}
	}
}

// 刷新订单
func (o *SpotOrder) onSnapshot(os OrderSnapshot) {
	o.tkRefreshTimeout.Reset(time.Second * 10)
	defer util.DefaultRecover()

	deal := common.Deal{}
	func() {
		// rest和ws都可能会调用这个函数，所以此处需要锁
		o.muRefresh.Lock()
		defer o.muRefresh.Unlock()

		if o.OrderId == 0 {
			o.OrderId = os.OrderID
		} else if o.OrderId > 0 && o.OrderId != os.OrderID {
			logger.LogPanic(o.LogPrefix, "order id not match! o=%s, new id=%d", o.String(), os.OrderID)
		}

		if o.CltOrderId != os.ClientOrderID {
			logger.LogPanic(o.LogPrefix, "order client-id not match! o=%s, new id=%s", o.String(), os.ClientOrderID)
		}

		// 刷新数据
		logger.LogInfo(o.LogPrefix, "recv order snapshot:%s", os.String())
		if os.UpdateTime.UnixMilli() >= o.UpdateTime.UnixMilli() && os.FilledSize.GreaterThanOrEqual(o.Filled) {

			deal = common.Deal{O: o, LocalTime: os.LocalTime, UTime: os.UpdateTime}
			if os.FillingPrice.IsPositive() && os.FillingSize.IsPositive() {
				deal.Price = os.FillingPrice
				deal.Amount = os.FillingSize
			} else {
				// 没有Filling数据时，是Rest得到的数据，采用预估值
				deal.Price = os.Price
				deal.Amount = os.FilledSize.Sub(o.Filled)
			}

			o.AvgPrice = o.AvgPrice.Mul(o.Filled).Add(deal.Price.Mul(deal.Amount))
			o.Price = os.Price
			o.Size = os.Size
			o.UpdateTime = os.UpdateTime
			o.Status = os.Status
			o.Filled = os.FilledSize

			if deal.Price.IsPositive() && deal.Amount.IsPositive() {
				logger.LogInfo(
					o.LogPrefix,
					"order dealing, dir=%s, price=%v, amount=%v, time=%v",
					common.OrderDir2Str(o.Dir), deal.Price, deal.Amount, deal.UTime)

				// 回调外部
				for _, obs := range o.Observers {
					if obs != nil {
						obs.OnDeal(deal)
					}
				}
			}

			// 注意一定要等外部回调结束后，再置订单完成状态
			finished := o.Status == binanceapi.OrderStatus_Canceled || o.Status == binanceapi.OrderStatus_Filled
			if !o.Finished && finished {
				o.Finished = finished
				logger.LogInfo(o.LogPrefix, "order finished")
			} else if o.Finished && !finished {
				logger.LogImportant(o.LogPrefix, "order already finished but try set to unfinished? impossible!")
			}

			o.refreshCount++
		}
	}()
}

// 立即刷新订单
func (o *SpotOrder) refreshImm() {
	o.chRefreshImm <- 0
}

func (o *SpotOrder) doRestRefresh() {
	logger.LogInfo(o.LogPrefix, "geting order info from rest...")
	resp, err := binancespotapi.GetOrder(o.InstId, 0, o.CltOrderId)
	b, _ := json.Marshal(resp)
	logger.LogInfo(o.LogPrefix, "getted order info from rest, resp=%s", string(b))
	if err == nil {
		if resp.Code == 0 && len(resp.Message) == 0 {
			os := NewOrderSnapShotFromRestResponse(*resp)
			o.onSnapshot(os)
		} else {
			// 其他错误连续出现3次则认为订单异常，强制结束
			o.restRefreshErrorCount++
			if o.restRefreshErrorCount >= 3 {
				o.ErrMsg = fmt.Sprintf("code:%d, msg:%s", resp.Code, resp.Message)
				o.FatalError = true
			}
		}
	}
}

func (o *SpotOrder) update() {
	defer logger.LogInfo(o.LogPrefix, "update exit")

	// go o.create()
	o.create()

	tkRepeat := time.NewTicker(time.Second)
	for {
		if o.IsFinished() {
			break
		}

		select {
		case <-o.chRefreshImm:
			o.doRestRefresh()
		case <-o.tkRefreshTimeout.C:
			o.doRestRefresh()
		case <-tkRepeat.C:
		}
	}
}

// #endregion
