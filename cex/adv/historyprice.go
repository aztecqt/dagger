/*
- @Author: aztec
- @Date: 2023-12-06 11:11:00
- @Description: 获取某一品种的历史价格
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package adv

import (
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/binance"
	"github.com/aztecqt/dagger/cex/okexv5"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

func GetOkxSpotHistoryPrice(baseCcy, quoteCcy string, t time.Time) decimal.Decimal {
	instId := okexv5.SpotTypeToInstId(baseCcy, quoteCcy)
	t = t.Add(time.Minute)
	bestPrice := decimal.Zero
	if resp, err := okexv5api.GetKlineBefore(instId, t, "1m", 3); err == nil {
		minDelta := int64(99999)
		for _, ku := range resp.Data {
			delta := util.AbsInt64(ku.Time.UnixMilli() - t.UnixMilli())
			if delta < minDelta {
				minDelta = delta
				bestPrice = ku.Open
			}
		}
	}
	return bestPrice
}

func GetOkxContractHistoryPrice(symbol, contractType string, t time.Time) decimal.Decimal {
	instId := okexv5.CCyCttypeToInstId(symbol, contractType)
	t = t.Add(time.Minute)
	bestPrice := decimal.Zero
	if resp, err := okexv5api.GetKlineBefore(instId, t, "1m", 3); err == nil {
		minDelta := int64(99999)
		for _, ku := range resp.Data {
			delta := util.AbsInt64(ku.Time.UnixMilli() - t.UnixMilli())
			if delta < minDelta {
				minDelta = delta
				bestPrice = ku.Open
			}
		}
	}
	return bestPrice
}

func GetBinanceSpotHistoryPrice(baseCcy, quoteCcy string, t time.Time) decimal.Decimal {
	bestPrice := decimal.Zero
	resp := binance.GetSpotKline(baseCcy, quoteCcy, t.Add(-time.Minute), t.Add(time.Minute), 60)
	minDelta := int64(99999)
	for _, ku := range resp {
		delta := util.AbsInt64(ku.Time.UnixMilli() - t.UnixMilli())
		if delta < minDelta {
			minDelta = delta
			bestPrice = ku.OpenPrice
		}
	}
	return bestPrice
}

func GetBinanceContractHistoryPrice(symbol, contractType string, t time.Time) decimal.Decimal {
	bestPrice := decimal.Zero
	resp := binance.GetFutureKline(symbol, contractType, t.Add(-time.Minute), t.Add(time.Minute), 60)
	minDelta := int64(99999)
	for _, ku := range resp {
		delta := util.AbsInt64(ku.Time.UnixMilli() - t.UnixMilli())
		if delta < minDelta {
			minDelta = delta
			bestPrice = ku.OpenPrice
		}
	}
	return bestPrice
}
