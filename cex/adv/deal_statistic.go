/*
 * @Author: aztec
 * @Date: 2023-03-03 11:54:54
 * @Description: 成交分析器。分析一段时间内的成交量、利润等信息。
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package adv

import (
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

type DealStatisticResult struct {
	VolumeCcy string
	Volume    float64
	VolumeUsd float64
	Profit    float64
	Profitusd float64
}

type DealStatistic struct {
	cex common.CEx
}

func (d *DealStatistic) Init(cex common.CEx) {
	d.cex = cex
}

func (d *DealStatistic) StatisticFuture(ccy, contractType string, t0, t1 time.Time) DealStatisticResult {
	rst := DealStatisticResult{}
	deals := d.cex.GetFutureDealHistory(ccy, contractType, t0, t1)
	if deals == nil {
		return rst
	}

	market := d.cex.UseFutureMarket(ccy, contractType, false)
	market.ValueAmount()
	contractValueinUsd := market.ValueCurrency() == "usd"

	// 交易利润计算
	stack := make([]common.DealHistory, 0)
	stackDir := common.OrderDir_None

	for _, dh := range deals {
		if stackDir == common.OrderDir_None || stackDir == dh.Dir {
			// 同交易方向入栈
			stack = append(stack, dh)
			stackDir = dh.Dir
		} else {
			// 不同交易方向，出栈并计算利润
			profit := 0.0
			profitusd := 0.0
			for {
				if len(stack) > 0 {
					dh0 := stack[len(stack)-1]
					if dh0.Amount.GreaterThanOrEqual(dh.Amount) {
						sz := dh.Amount
						px0 := dh0.Price
						px1 := dh.Price

						// 计算利润
						p := common.CalProfit(sz, px0, px1, dh.Dir)
						if contractValueinUsd {
							profit += p * market.ValueAmount().InexactFloat64()
							profitusd += p * market.ValueAmount().InexactFloat64()
						} else {
							profit += p * market.ValueAmount().InexactFloat64()
							profitusd += p * market.ValueAmount().InexactFloat64() * dh.Price.InexactFloat64()
						}

						// 扣除平仓量
						dh0.Amount = dh0.Amount.Sub(dh.Amount)
						dh.Amount = decimal.Zero
					} else {
						sz := dh0.Amount
						px0 := dh0.Price
						px1 := dh.Price

						// 计算利润
						p := common.CalProfit(sz, px0, px1, dh.Dir)
						if contractValueinUsd {
							profit += p * market.ValueAmount().InexactFloat64()
							profitusd += p * market.ValueAmount().InexactFloat64()
						} else {
							profit += p * market.ValueAmount().InexactFloat64()
							profitusd += p * market.ValueAmount().InexactFloat64() * dh.Price.InexactFloat64()
						}

						// 扣除平仓量
						dh.Amount = dh.Amount.Sub(dh0.Amount)
						dh0.Amount = decimal.Zero
					}

					if dh0.Amount.IsZero() {
						// 末位已全部计入，删除它
						stack = stack[:len(stack)-1]
					} else {
						// 否则覆盖
						stack[len(stack)-1] = dh0
					}

					if dh.Amount.IsZero() {
						// 新成交已完全抵消，继续下一个
						break
					}
				} else {
					// 全部抵消，反向入栈
					stack = append(stack, dh)
					stackDir = dh.Dir
				}
			}

			rst.Profit += profit
			rst.Profitusd += profitusd
		}

		// 计算成交量
		rst.VolumeCcy = market.ValueCurrency()
		rst.Volume += dh.Amount.InexactFloat64()
		if contractValueinUsd {
			rst.VolumeUsd += dh.Amount.InexactFloat64() * market.ValueAmount().InexactFloat64()
		} else {
			rst.VolumeUsd += dh.Amount.InexactFloat64() * market.ValueAmount().InexactFloat64() * dh.Price.InexactFloat64()
		}
	}

	return rst
}
