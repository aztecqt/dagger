/*
 * @Author: aztec
 * @Date: 2022-04-03 19:39:28
 * @LastEditors: aztec
 * @LastEditTime: 2023-03-02 11:58:15
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\okexv5\common_order.go
 * @Description: okexv5订单。合约订单和现货订单分别“继承”自这个struct
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// okx现货订单/合约订单的共同基类
type CommonOrder struct {
	common.OrderImpl

	posSide               string // 操作哪一侧的仓位（仅合约）
	canceling             bool   // 是否正在取消(调试用)
	modifying             bool   // 是否正在修改(调试用)
	restRefreshErrorCount int    // rest调用错误次数
	refreshCount          int    // 刷新次数

	// 子类提供
	getPosSide func() string
	tradeMode  func() string

	// 刷新
	muRefresh        sync.Mutex
	tkRefreshTimeout *time.Ticker
	chRefreshImm     chan int
}

func (o *CommonOrder) Go() {
	o.tkRefreshTimeout = time.NewTicker(time.Second * 10)
	go o.update()
}

// #region 实现common.Order
func (o *CommonOrder) GetExchangeName() string {
	return exchangeName
}

func (o *CommonOrder) String() string {
	return fmt.Sprintf("%s[frame:%d modifying:%v canceling:%v]", o.OrderImpl.String(), o.refreshCount, o.modifying, o.canceling)
}

func (o *CommonOrder) IsSupportModify() bool {
	return true
}

func (o *CommonOrder) Modify(newPrice, newSize decimal.Decimal) {
	if !o.IsFinished() {
		go o.modify(newPrice, newSize)
	}
}

func (o *CommonOrder) Cancel() {
	if !o.IsFinished() {
		go o.cancel()
	}
}

// #endregion

// #region 自身逻辑
func (o *CommonOrder) create() {
	defer util.DefaultRecover()

	// 已经创建的订单不会再次被创建
	if o.OrderId > 0 {
		return
	}

	side := "buy"
	if o.Dir == common.OrderDir_Sell {
		side = "sell"
	}

	o.posSide = o.getPosSide()

	orderType := "limit"
	if o.MakeOnly {
		orderType = "post_only"
	}

	// 调用api
	logger.LogInfo(o.LogPrefix, "creating [%s]", o.String())
	resp, err := okexv5api.MakeOrder(
		o.InstId,
		o.CltOrderId,
		orderTag(),
		side,
		o.posSide,
		orderType,
		o.tradeMode(),
		o.ReduceOnly,
		o.Price,
		o.Size)
	if err == nil {
		if len(resp.Data) > 0 {
			if resp.Data[0].SCode != "0" {
				o.ErrMsg = fmt.Sprintf("code=%s, msg=%s", resp.Data[0].SCode, resp.Data[0].SMsg)
				o.FatalError = true // 只有这种情况可以明确的认为订单已经失败了
				logger.LogImportant(o.LogPrefix, "create order error: %s", o.ErrMsg)
			} else if resp.Data[0].OrderId != "0" {
				o.OrderId = util.String2Int64Panic(resp.Data[0].OrderId)
				logger.LogInfo(o.LogPrefix, "create success, order id = %v", o.OrderId)
			} else {
				o.ErrMsg = "create success but missing order id"
				logger.LogPanic(o.LogPrefix, "create order error, invalid order id")
			}
		} else {
			o.ErrMsg = "response error, no data"
			o.FatalError = true // 这种情况应该是服务器还没准备好，订单可以尝试重新创建
			logger.LogInfo(o.LogPrefix, "create order error, no data")
		}
	} else {
		// 网络错误不代表订单未创建成功
		// 应该查询时返回“订单不存在”作为订单错误的触发条件
		logger.LogImportant(o.LogPrefix, "create order with rest error: %s", err.Error())
	}
}

// 取消订单
// 无论成功与否，都直接返回。逻辑层如果觉得仍有必要取消，再次调用即可
func (o *CommonOrder) cancel() {
	if !o.canceling {
		o.canceling = true
		defer util.DefaultRecover()
		defer func() {
			o.canceling = false
		}()

		logger.LogInfo(o.LogPrefix, "canceling [%s]", o.String())
		resp, err := okexv5api.CancelOrder(o.InstId, o.CltOrderId, 0)
		if err == nil {
			if resp.Data[0].SCode != "0" {
				o.ErrMsg = fmt.Sprintf("code:%s, msg:%s", resp.Data[0].SCode, resp.Data[0].SMsg)
				code := util.String2IntPanic(resp.Data[0].SCode)
				if code == 51400 /*不存在*/ || code == 51401 /*已撤销*/ || code != 51402 /*已完成*/ {
					o.refreshImm()
				} else if code != 51410 /*撤销中*/ && code != 51405 /*没有未成交的订单*/ && code != 51404 /*不可撤单*/ {
					logger.LogImportant(o.LogPrefix, "cancel order error: %s", o.ErrMsg)
				}
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

// 修改订单
// 无论修改成功与否，都直接返回。逻辑层如果觉得仍有必要修改，再次调用即可
func (o *CommonOrder) modify(newPrice, newSize decimal.Decimal) {
	if !o.modifying {
		o.modifying = true
		defer util.DefaultRecover()
		defer func() {
			o.modifying = false
		}()

		if newPrice.IsPositive() {
			newPrice = o.InstrumentMgr.AlignPrice(
				o.InstId,
				newPrice,
				o.Dir,
				o.MakeOnly,
				o.Trader.Market().OrderBook().Buy1(),
				o.Trader.Market().OrderBook().Sell1())
		}

		if newSize.IsPositive() {
			newSize = o.InstrumentMgr.AlignSize(o.InstId, newSize)
			minSize := o.InstrumentMgr.MinSize(o.InstId, o.Price)
			if newSize.LessThan(minSize) {
				o.Cancel()
				return
			}
		}

		if newSize.IsPositive() || newPrice.IsPositive() {
			logger.LogInfo(o.LogPrefix, "modifying [%s], newPrice=%v, newSize=%v", o.String(), newPrice, newSize)
			resp, err := okexv5api.AmendOrder(o.InstId, o.CltOrderId, NewAmendId(), 0, newPrice, newSize)
			if err == nil {
				if resp.Data[0].SCode != "0" {
					o.ErrMsg = fmt.Sprintf("code:%s, msg:%s", resp.Data[0].SCode, resp.Data[0].SMsg)
					code := util.String2IntPanic(resp.Data[0].SCode)
					if code == 51509 /*已撤销*/ || code == 51510 /*已完成*/ || code == 51503 /*订单不存在*/ {
						o.refreshImm()
					} else {
						logger.LogImportant(o.LogPrefix, "modify order error: %s", o.ErrMsg)
					}
					time.Sleep(time.Second)
				} else {
					logger.LogInfo(o.LogPrefix, "modify responsed")
				}
			} else {
				logger.LogImportant(o.LogPrefix, "modify order with rest error: %s", err.Error())
				time.Sleep(time.Second)
			}
		}
	}
}

func (o *CommonOrder) onSnapshot(os OrderSnapshot) {
	o.tkRefreshTimeout.Reset(time.Second * 10)
	defer util.DefaultRecover()

	deal := common.Deal{}
	func() {
		// rest和ws都可能会调用这个函数，所以此处需要锁
		o.muRefresh.Lock()
		defer o.muRefresh.Unlock()

		if o.OrderId == 0 {
			o.OrderId = os.id
		} else if o.OrderId > 0 && o.OrderId != os.id {
			logger.LogPanic(o.LogPrefix, "order id not match! o=%s, new id=%d", o.String(), os.id)
		}

		if o.CltOrderId != os.clientId {
			logger.LogPanic(o.LogPrefix, "order client-id not match! o=%s, new id=%s", o.String(), os.clientId)
		}

		// 刷新数据
		logger.LogInfo(o.LogPrefix, "recv order snapshot:%s", os.String())
		if os.updateTime.UnixMilli() >= o.UpdateTime.UnixMilli() && os.filled.GreaterThanOrEqual(o.Filled) {
			filledOld := o.Filled
			avgPriceOld := o.AvgPrice
			filledNew := os.filled
			avgPricNew := os.avgPrice

			o.Price = os.price
			o.Size = os.size
			o.Filled = filledNew
			o.AvgPrice = avgPricNew
			o.UpdateTime = os.updateTime
			o.Status = os.status

			price, amount := common.CalculateOrderDeal(filledOld, avgPriceOld, filledNew, avgPricNew)
			if price.IsPositive() && amount.IsPositive() {
				logger.LogInfo(o.LogPrefix, "order dealing, dir=%s, price=%v, amount=%v, time=%v", common.OrderDir2Str(o.Dir), price, amount, os.updateTime)
				deal = common.Deal{O: o, Price: price, Amount: amount, LocalTime: os.localTime, UTime: os.updateTime}
			}

			// 回调外部
			if deal.Price.IsPositive() && deal.Amount.IsPositive() && len(o.Observers) > 0 {
				for _, obs := range o.Observers {
					if obs != nil {
						obs.OnDeal(deal)
					}
				}
			}

			// 注意一定要等外部回调结束后，再置订单完成状态
			finished := o.Status == okexv5api.OrderStatus_Canceled || o.Status == okexv5api.OrderStatus_Filled
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
func (o *CommonOrder) refreshImm() {
	o.chRefreshImm <- 0
}

func (o *CommonOrder) doRestRefresh() {
	logger.LogInfo(o.LogPrefix, "geting order info from rest...")
	resp, err := okexv5api.GetOrderInfo(o.InstId, 0, o.CltOrderId)
	b, _ := json.Marshal(resp)
	logger.LogInfo(o.LogPrefix, "getted order info from rest, resp=%s", string(b))

	if err == nil {
		if resp.Code == "0" {
			os := OrderSnapshot{}
			os.localTime = resp.LocalTime
			os.Parse(resp.Data[0], "rest")
			o.onSnapshot(os)
		} else if resp.Code == "51603" { // 订单不存在
			o.ErrMsg = fmt.Sprintf("code:%s, msg:%s", resp.Code, resp.Msg)
			o.FatalError = true // 此时订单生命周期可以结束了
		} else {
			// 其他错误连续出现3次则认为订单异常，强制结束
			o.restRefreshErrorCount++
			if o.restRefreshErrorCount >= 3 {
				o.ErrMsg = fmt.Sprintf("code:%s, msg:%s", resp.Code, resp.Msg)
				o.FatalError = true
			}
		}
	}
}

func (o *CommonOrder) update() {
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
