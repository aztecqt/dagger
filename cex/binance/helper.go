/*
 * @Author: aztec
 * @Date: 2022-10-21 10:35:06
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binance

import (
	"fmt"
	"strings"

	"github.com/aztecqt/dagger/util/logger"
)

// btc_usdt_swap -> BTCUSDT, true
// btc_usd_swap -> BTCUSDT, false
func FutureInstId2Symbol(instid string) (symbol string, isusdt bool, ok bool) {
	ss := strings.Split(instid, "_")
	if len(ss) == 3 {
		ccy := ss[0]
		ctype1 := ss[1]
		ctype2 := ss[2]
		if ctype2 == "swap" {
			if ctype1 == "usd" {
				symbol = fmt.Sprintf("%sUSD_PERP", strings.ToUpper(ccy))
				isusdt = false
				ok = true
			} else if ctype1 == "usdt" {
				symbol = fmt.Sprintf("%sUSDT", strings.ToUpper(ccy))
				isusdt = true
				ok = true
			} else {
				// 不是usdt或者usd
				logger.LogImportant(logPrefix, "can't convert instid to bn future symbol: %s", instid)
				ok = false
			}
		} else {
			// 暂不支持swap之外的合约类型
			logger.LogImportant(logPrefix, "can't convert instid to bn future symbol: %s", instid)
			ok = false
		}
	} else {
		// 就离谱
		logger.LogImportant(logPrefix, "can't convert instid to bn future symbol: %s", instid)
		ok = false
	}

	return
}

// btc_usdt -> BTCUSDT
func SpotInstId2Symbol(instid string) (symbol string, ok bool) {
	ss := strings.Split(instid, "_")
	if len(ss) == 2 {
		baseccy := ss[0]
		quoteccy := ss[1]
		symbol = fmt.Sprintf("%s%s", strings.ToUpper(baseccy), strings.ToUpper(quoteccy))
		ok = true
	} else {
		// 就离谱
		logger.LogImportant(logPrefix, "can't convert instid to bn spot symbol: %s", instid)
		ok = false
	}

	return
}

// btc, usdt->BTCUSDT
func SpotTypeToInstId(baseccy, quoteccy string) string {
	return fmt.Sprintf("%s%s", strings.ToUpper(baseccy), strings.ToUpper(quoteccy))
}
