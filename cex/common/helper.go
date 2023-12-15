/*
 * @Author: aztec
 * @Date: 2022-04-01 17:13:43
 * @FilePath: \stratergyc:\work\svn\quant\go\src\dagger\cex\common\helper.go
 * @Description: 通用帮助函数
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package common

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// 生成ClientOrderId
var accClientOrderId int32

func NewClientOrderId(postfix string) string {
	newId := atomic.AddInt32(&accClientOrderId, 1)
	return util.ToLetterNumberOnly(fmt.Sprintf("%05d%s", newId, postfix), 32)
}

// 计算订单的成交明细
func CalculateOrderDeal(filled, avgPrice, filledNew, avgPriceNew decimal.Decimal) (price decimal.Decimal, amount decimal.Decimal) {
	if filled.IsZero() && avgPrice.IsZero() {
		price = avgPriceNew
		amount = filledNew
	} else if !filled.IsZero() && !avgPrice.IsZero() {
		m0 := filled.Mul(avgPrice)
		m1 := filledNew.Mul(avgPriceNew)
		if m1.LessThan(m0) {
			logger.LogPanic("CalculateOrderDeal", `"filledNew" can't less than "filled"`)
			price = decimal.Zero
			amount = decimal.Zero
		} else {
			amount = filledNew.Sub(filled)
			if amount.IsZero() {
				price = decimal.Zero
			} else {
				price = m1.Sub(m0).Div(amount)
			}
		}
	} else {
		logger.LogPanic("CalculateOrderDeal", `"filled" and "avgPrice" can not one zero one not zero`)
		price = decimal.Zero
		amount = decimal.Zero
	}
	return
}

// U数量转换为合约张数
func USDT2ContractAmount(usdt decimal.Decimal, m FutureMarket) decimal.Decimal {
	if strings.Contains(m.ValueCurrency(), "usd") {
		// usd合约
		return usdt.Div(m.ValueAmount()).RoundDown(0)
	} else {
		// usdt合约
		if m.MarkPrice().IsPositive() {
			valueAmountUsdt := m.ValueAmount().Mul(m.MarkPrice())
			return usdt.Div(valueAmountUsdt).RoundDown(0)
		} else {
			return decimal.Zero
		}
	}
}

func USDT2ContractAmountFloatUnRounded(usdt decimal.Decimal, m FutureMarket) decimal.Decimal {
	if strings.Contains(m.ValueCurrency(), "usd") {
		// usd合约
		return usdt.Div(m.ValueAmount())
	} else {
		// usdt合约
		if m.MarkPrice().IsPositive() {
			valueAmountUsdt := m.ValueAmount().Mul(m.MarkPrice())
			return usdt.Div(valueAmountUsdt)
		} else {
			return decimal.Zero
		}
	}
}

func USDT2ContractAmountAtLeast1(usdt decimal.Decimal, m FutureMarket) decimal.Decimal {
	isUSDTContract := !strings.Contains(m.ValueCurrency(), "usd")
	if isUSDTContract {
		// usdt合约
		if m.MarkPrice().IsPositive() {
			valueAmountUsdt := m.ValueAmount().Mul(m.MarkPrice())
			ct := usdt.Div(valueAmountUsdt)
			if ct.GreaterThan(util.DecimalOne) {
				return ct.RoundDown(0)
			} else if ct.IsPositive() {
				return util.DecimalOne
			} else {
				return decimal.Zero
			}
		} else {
			return decimal.Zero
		}
	} else {
		// usd合约
		return usdt.Div(m.ValueAmount()).RoundDown(0)
	}
}

// 合约张数转为usd价值
func ContractAmount2USD(amount decimal.Decimal, m FutureMarket) decimal.Decimal {
	if strings.Contains(m.ValueCurrency(), "usd") {
		// usd合约，合约面值以USD计价
		return amount.Mul(m.ValueAmount())
	} else {
		// usdt合约，合约面值以币计价
		mkPrice := m.MarkPrice()
		if mkPrice.IsPositive() {
			return amount.Mul(m.ValueAmount()).Mul(mkPrice)
		} else {
			return decimal.Zero
		}
	}
}

// 求以usd为单位的合约盘口密度
func ContractDensityUSD(r float64, ob *Orderbook, valueCcy string, valueAmount float64) float64 {
	valAmountUSD := 0.0
	if strings.Contains(strings.ToLower(valueCcy), "usd") {
		valAmountUSD = valueAmount
	} else {
		valAmountUSD = valueAmount * ob.MiddlePrice().InexactFloat64()
	}
	densityUSD := ob.Density(r) * valAmountUSD
	return densityUSD
}

// 计算利润
func CalProfit(sz, px0, px1 decimal.Decimal, dir1 OrderDir) float64 {
	if dir1 == OrderDir_Buy {
		return px0.Sub(px1).Div(px0).Mul(sz).InexactFloat64()
	} else if dir1 == OrderDir_Sell {
		return px1.Sub(px0).Div(px0).Mul(sz).InexactFloat64()
	} else {
		return 0
	}
}

// 等待trader就绪
func WaitTraderReady(trader CommonTrader, logPrefix string) {
	logger.LogInfo(logPrefix, "using trader %s", trader.Market().Type())
	for {
		time.Sleep(time.Millisecond * 100)
		if trader.Ready() {
			logger.LogInfo(logPrefix, "trader %s is ready", trader.Market().Type())
			break
		} else {
			logger.LogInfo(logPrefix, trader.ReadyStr())
		}
	}
}

// 等待market就绪
func WaitMarketReady(market CommonMarket) {
	for !market.Ready() {
		time.Sleep(time.Millisecond * 100)
	}
}
