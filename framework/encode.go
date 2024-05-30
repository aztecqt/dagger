/*
 * @Author: aztec
 * @Date: 2024-05-05 09:19:28
 * @Description:
 *
 * Copyright (c) 2024 by aztec, All Rights Reserved.
 */
package framework

import (
	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

// 一组用于传输的dataline
type EncodedDataLines struct {
	Names      []string    `json:"names"`  // dataline名称
	TimeStamps []int64     `json:"ts"`     // 时间序列
	Values     [][]float64 `json:"values"` // 第一个维度对应时间，第二个维度对应names
}

// 一组用于传输的成交记录
// 格式：
// type：timestamp、dir、price、amount、timestamp、dir、price、amount
// 数据的len应该是4的整数倍
type EncodedDealRecords struct {
	Data map[string][]interface{} `json:"data"`
}

// 一组用于传输的订单
// 格式：
// dir、price、size、filled
// 数据的len应该是4的整数倍
type EncodedOrders struct {
	Data []interface{} `json:"data"`
}

// 所有dl的后半段数据应该是完全对齐的，前半段不做要求
func EncodeDatalines(dls []*DataLine, fromTs int64, limit int) (EncodedDataLines, bool) {
	en := EncodedDataLines{}

	maxLen := 0
	for _, dl := range dls {
		if dl.Length() > maxLen {
			maxLen = dl.Length()
		}
	}

	// 不允许不设限制
	if limit == 0 {
		limit = 1
	}

	if limit > maxLen {
		limit = maxLen
	}

	// 确保dataline末端是对齐的
	if len(dls) > 0 {
		dl := dls[0]

		tlast, _ := dl.LastMS()
		for i := 1; i < len(dls); i++ {
			tlast_i, _ := dls[i].LastMS()
			if tlast_i != tlast {
				return en, false
			}
		}

		// 确定数据条数
		count := limit
		limit2 := (tlast-fromTs)/dl.IntervalMS() + 1
		if count > int(limit2) {
			count = int(limit2)
		}

		// 填充名字
		for _, dl := range dls {
			en.Names = append(en.Names, dl.Name())
		}

		// 填充数据
		for i := count; i > 0; i-- {
			ms, _ := dl.GetTime(-i)
			if ms > fromTs {
				en.TimeStamps = append(en.TimeStamps, ms)

				vals := []float64{}
				for _, dl := range dls {
					v, _ := dl.GetValue(-i)
					vals = append(vals, v)
				}
				en.Values = append(en.Values, vals)
			}
		}
	}

	return en, true
}

// 编码交易时间
func EncodeTradingTime(t common.TradingTimes) []int64 {
	rst := []int64{}
	for _, v := range t {
		if !v.OpenTime.IsZero() && !v.CloseTime.IsZero() {
			rst = append(rst, v.OpenTime.UnixMilli(), v.CloseTime.UnixMilli())
		}
	}
	return rst
}

// 编码历史成交
func EncodeDealRecords(deals []DealRecord, fromTs int64, limit int) EncodedDealRecords {
	edrs := EncodedDealRecords{Data: map[string][]interface{}{}}

	// 不允许不设限制
	if limit == 0 {
		limit = 1
	}

	// 计算起始索引
	startIndex := len(deals)

	if fromTs == 0 {
		startIndex = len(deals) - limit
		if startIndex < 0 {
			startIndex = 0
		}
	} else {
		// 反向遍历
		for i := 0; i < limit; i++ {
			index := len(deals) - 1 - i
			if index < 0 {
				break
			}
			if deals[index].TimeStamp > fromTs {
				startIndex = index
			} else {
				break
			}
		}
	}

	for i := startIndex; i < len(deals); i++ {
		d := deals[i]
		arry := edrs.Data[d.Type]
		arry = append(arry, d.TimeStamp, d.Dir, d.Price, d.Amount)
		edrs.Data[d.Type] = arry
	}

	return edrs
}

// 编码一个订单
func EncodeOrder(eords *EncodedOrders, price, size, filled decimal.Decimal, dir common.OrderDir) {
	eords.Data = append(eords.Data, dir, price, size, filled)
}
