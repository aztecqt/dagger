/*
- @Author: aztec
- @Date: 2024-03-11 10:14:08
- @Description:
- @ 目前只支持现货订单
- @ ibkr的订单不支持单独查询订单，因此没有类似其他交易所的“单独主动刷新订单”的逻辑
- @ 主动刷新订单的任务，交给ex统一、周期性完成
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package ibkrtws

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/ibkr/twsapi"
	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type SpotOrder struct {
	common.OrderImpl
	c            *twsapi.Client
	contract     *twsmodel.Contract
	twsOrder     twsmodel.Order
	baseCcy      string
	quoteCcy     string
	ex           *Exchange // 订单需要在exchange对象中注册，以便得到更新推送
	orderTif     string    // 优先GTC(good till cancel)，不行的话，按照交易所规定来
	canceling    bool      // 是否正在取消(调试用)
	modifying    bool      // 是否正在修改(调试用)
	refreshCount int       // 刷新次数
}

func (o *SpotOrder) init(
	trader *SpotTrader,
	price, amount decimal.Decimal,
	dir common.OrderDir,
	tif string,
	purpose string) bool {
	o.c = trader.ex.c
	o.contract = trader.market.contract
	o.ex = trader.ex
	o.orderTif = tif
	o.baseCcy = trader.market.baseCcy
	o.quoteCcy = trader.market.quoteCcy

	o.CltOrderId = o.c.NextOrderId()
	return o.OrderImpl.Init(trader, trader.ex.instrumentMgr, trader.market.inst.Id, price, amount, dir, false, false, purpose)
}

func (o *SpotOrder) Go() {
	go o.create()
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
	if !o.IsFinished() {
		go o.modify(newPrice, newSize)
	}
}

func (o *SpotOrder) Cancel() {
	if !o.IsFinished() {
		go o.cancel()
	}
}

// #endregion

// #region 自身逻辑
func (o *SpotOrder) create() {
	defer util.DefaultRecover()

	// 已经创建的订单不会再次被创建
	if o.OrderId > 0 {
		return
	}

	logInfo(o.LogPrefix, "creating [%s]", o.String())

	// 调用api
	// 目前仅支持限价单，不支持其他模式，包括我们自己的MakeOnly也不支持（限价单）
	o.ex.registerOrderStatusHandler(o.CltOrderId.(int), o.onOrderStatus)
	o.twsOrder = twsmodel.NewOrder()

	o.twsOrder.OrderId = o.CltOrderId.(int)
	o.twsOrder.Action = util.ValueIf(o.Dir == common.OrderDir_Buy, "BUY", "SELL")
	o.twsOrder.LmtPrice = o.Price
	o.twsOrder.OrderType = "LMT"
	o.twsOrder.Tif = o.orderTif
	o.twsOrder.TotalQuantity = o.Size

	// 返回不会为nil
	resp := *o.c.PlaceOrder(*o.contract, o.twsOrder)
	if resp.RespCode == twsapi.RespCode_Ok {
		if resp.OrderStatus != nil {
			// tws返回正常下单结果
			logInfo(o.LogPrefix, "create order success")
		} else if resp.Err != nil {
			// tws返回失败
			o.ErrMsg = fmt.Sprintf("place order error, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
			o.FatalError = true
			logError(o.LogPrefix, o.ErrMsg)
		} else {
			// 不可能
			o.ErrMsg = fmt.Sprintf("place order failed, invalid response: %v", resp)
			o.FatalError = true
			logError(o.LogPrefix, o.ErrMsg)
		}
	} else {
		// 下单失败
		if resp.RespCode == twsapi.RespCode_TimeOut {
			o.ErrMsg = fmt.Sprintf("place order time-out")
			o.FatalError = true
			logErrorWithTolerate("SpotOrder.create.timeout", 300, 5, o.LogPrefix, o.ErrMsg)
		} else {
			o.ErrMsg = fmt.Sprintf("place order inner error, respCode=%d", resp.RespCode)
			o.FatalError = true
			logInfo(o.LogPrefix, o.ErrMsg, "")
		}
	}
}

// 由trade调用
func (o *SpotOrder) uninit() {
	o.ex.unregisterOrderStatusHandler(o.CltOrderId.(int))
	o.ex.clearFrozenBalance(o.twsOrder.OrderId)
}

// 取消订单
func (o *SpotOrder) cancel() {
	if !o.canceling {
		o.canceling = true
		defer func() {
			o.canceling = false
		}()

		logInfo(o.LogPrefix, "canceling [%s]", o.String())
		resp := o.c.CancelOrder(o.CltOrderId.(int), "")
		if resp.RespCode == twsapi.RespCode_Ok {
			if resp.OrderStatus != nil {
				// 正常返回结果
				if resp.OrderStatus.Status == twsmodel.OrderStatus_Cancelled {
					// 撤单成功
					logInfo(o.LogPrefix, "cancel success")
				} else {
					// 实际不可能
					logError(o.LogPrefix, "cancel order responsed, but status is not Cancelled")
				}
			} else if resp.Err != nil {
				// tws返回失败
				o.ErrMsg = fmt.Sprintf("cancel order error, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
				logInfo(o.LogPrefix, o.ErrMsg)
				time.Sleep(time.Second)
			} else {
				// 不可能
				o.ErrMsg = fmt.Sprintf("cancel order failed, invalid response: %v", resp)
				logError(o.LogPrefix, o.ErrMsg)
				time.Sleep(time.Second)
			}
		} else {
			// 撤单调用失败
			o.ErrMsg = fmt.Sprintf("cancel order inner error, respCode=%d", resp.RespCode)
			o.FatalError = true
			logInfo(o.LogPrefix, o.ErrMsg)
			time.Sleep(time.Second)
		}
	}
}

// 修改订单（似乎问题比较多，比如部分成交不可以修改等，暂时不要使用）
func (o *SpotOrder) modify(newPrice, newSize decimal.Decimal) {
	if !o.modifying {
		o.modifying = true
		defer func() {
			o.modifying = false
		}()
	}

	if newPrice.IsPositive() {
		newPrice = o.InstrumentMgr.AlignPrice(
			o.InstId,
			newPrice,
			o.Dir,
			false,
			o.Trader.Market().OrderBook().Buy1Price(),
			o.Trader.Market().OrderBook().Sell1Price(),
		)
		o.twsOrder.LmtPrice = newPrice
		logInfo(o.LogPrefix, "modifying [%s], new price=%v", o.String(), newPrice)
	}

	if newSize.IsPositive() {
		newSize = o.InstrumentMgr.AlignSize(o.InstId, newSize)
		minSize := o.InstrumentMgr.MinSize(o.InstId, o.Price)
		if newSize.LessThan(minSize) {
			o.Cancel()
			return
		}
		logInfo(o.LogPrefix, "modifying [%s], new size=%v", o.String(), newSize)
		o.twsOrder.TotalQuantity = newSize
	}

	// 返回不会为nil
	resp := *o.c.PlaceOrder(*o.contract, o.twsOrder)
	if resp.RespCode == twsapi.RespCode_Ok {
		if resp.OrderStatus != nil {
			// tws返回正常下单结果
			logInfo(o.LogPrefix, "modify order success")
		} else if resp.Err != nil {
			// tws返回失败
			o.ErrMsg = fmt.Sprintf("motify order error, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
			logError(o.LogPrefix, o.ErrMsg)
		} else {
			// 不可能
			o.ErrMsg = fmt.Sprintf("modify order failed, invalid response: %v", resp)
			logError(o.LogPrefix, o.ErrMsg)
		}
	} else {
		// 下单失败
		if resp.RespCode == twsapi.RespCode_TimeOut {
			o.ErrMsg = fmt.Sprintf("modify order time out")
			logErrorWithTolerate("SpotOrder.modify.timeout", 300, 5, o.LogPrefix, o.ErrMsg)
		} else {
			o.ErrMsg = fmt.Sprintf("modify order inner error, respCode=%d", resp.RespCode)
			logInfo(o.LogPrefix, o.ErrMsg)
		}
	}
}

func (o *SpotOrder) onOrderStatus(os *twsapi.OrderStatusMsg, oo *twsapi.OpenOrdersMsg) {
	if os != nil {
		// permId才是真正的orderId
		if o.OrderId == 0 {
			o.OrderId = int64(os.PermId)
		} else if o.OrderId != int64(os.PermId) {
			logError(o.LogPrefix, "order id not match! o=%s, new id=%d", o.String(), os.OrderId)
		}

		// os.OrderId实际上是ClientOrderId
		if o.CltOrderId != os.OrderId {
			logError(o.LogPrefix, "order client id not match, o=%s, new id=%d", o.String(), os.ClientId)
		}

		// 刷新数据
		logInfo(o.LogPrefix, "recv order snapshot:%s", os.String())
		if os.Filled.GreaterThanOrEqual(o.Filled) {
			filledOld := o.Filled
			avgPriceOld := o.AvgPrice
			filledNew := os.Filled
			avgPricNew := os.AvgFillPrice

			o.Filled = filledNew
			o.AvgPrice = avgPricNew
			o.UpdateTime = time.Now()
			o.Status = os.Status

			deal := common.Deal{}
			price, amount := common.CalculateOrderDeal(filledOld, avgPriceOld, filledNew, avgPricNew)
			if price.IsPositive() && amount.IsPositive() {
				logInfo(o.LogPrefix, "order dealing, dir=%s, price=%v, amount=%v", common.OrderDir2Str(o.Dir), price, amount)
				deal = common.Deal{O: o, Price: price, Amount: amount, LocalTime: time.Now(), UTime: time.Now()}
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
			finished := o.Status == twsmodel.OrderStatus_Cancelled || o.Status == twsmodel.OrderStatus_Filled || o.Status == twsmodel.OrderStatus_Inactive
			if !o.Finished && finished {
				o.Finished = finished
				logInfo(o.LogPrefix, "order finished")
			} else if o.Finished && !finished {
				logError(o.LogPrefix, "order already finished but try set to unfinished? impossible!")
			}

			o.refreshCount++
		}

		o.UpdateTime = time.Now()
	}

	if oo != nil {
		// 仅用来刷新价格、数量（当modify order时）
		o.Price = oo.Order.LmtPrice
		o.Size = oo.Order.TotalQuantity
	}

	if o.Dir == common.OrderDir_Buy {
		// 买单冻结quoteCurrency
		o.ex.setFrozenBalance(o.twsOrder.OrderId, o.quoteCcy, o.Price.Mul(o.Size))
	} else {
		// 卖单冻结baseCurrency
		o.ex.setFrozenBalance(o.twsOrder.OrderId, o.baseCcy, o.Size)
	}
}

// #endregion
