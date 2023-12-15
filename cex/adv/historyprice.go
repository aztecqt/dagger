/*
- @Author: aztec
- @Date: 2023-12-06 11:11:00
- @Description: 获取某一品种的历史价格
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package adv

import (
	"strings"
	"time"

	"github.com/aztecqt/dagger/api/binanceapi/binancefutureapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancespotapi"
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
			delta := util.AbsInt64(ku.TS - t.UnixMilli())
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
			delta := util.AbsInt64(ku.TS - t.UnixMilli())
			if delta < minDelta {
				minDelta = delta
				bestPrice = ku.Open
			}
		}
	}
	return bestPrice
}

func GetBinanceSpotHistoryPrice(baseCcy, quoteCcy string, t time.Time) decimal.Decimal {
	instId := binance.SpotTypeToInstId(baseCcy, quoteCcy)
	bestPrice := decimal.Zero
	resp := binance.GetKline(instId, t.Add(-time.Minute), t.Add(time.Minute), 60, binancespotapi.GetKline)
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
	instId := binance.CCyCttypeToInstId(symbol, contractType)
	bestPrice := decimal.Zero
	fn := binancefutureapi.GetKline_Usd
	if strings.Contains(contractType, "usdt") {
		fn = binancefutureapi.GetKline_Usdt
	}

	resp := binance.GetKline(instId, t.Add(-time.Minute), t.Add(time.Minute), 60, fn)
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
