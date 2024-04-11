/*
 * @Author: aztec
 * @Date: 2022-04-01 15:56:40
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-03-12 16:49:12
 * @FilePath: \dagger\cex\okexv5\helper.go
 * @Description: okexv5的帮助函数
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

// btc usdt_swap -> BTC-USDT-SWAP
// btc usd_swap -> BTC-USD-SWAP
func CCyCttypeToInstId(symbol, contractType string) string {
	switch contractType {
	case "usd_swap":
		instId := fmt.Sprintf("%s-USD-SWAP", strings.ToUpper(symbol))
		return instId
	case "usdt_swap":
		instId := fmt.Sprintf("%s-USDT-SWAP", strings.ToUpper(symbol))
		return instId
	default:
		logger.LogPanic(logPrefix, "unknown contractType: %s", contractType)
		return ""
	}
}

func ContractType2OkxInstType(contractType string) string {
	switch contractType {
	case "usd_swap":
	case "usdt_swap":
		return "SWAP"
	case "this_week":
	case "next_week":
	case "this_quarter":
	case "next_quarter":
		return "FUTURE"
	default:
		logger.LogPanic(logPrefix, "unknown contractType:%s", contractType)
	}

	return ""
}

// BTC-USDT-SWAP -> btc
func InstId2Symbol(instId string) string {
	return strings.ToLower(strings.Split(instId, "-")[0])
}

func InstId2ContractType(instId string) string {
	if strings.Contains(instId, "USDT-SWAP") {
		return "usdt_swap"
	} else if strings.Contains(instId, "USD-SWAP") {
		return "usd_swap"
	} else {
		logger.LogPanic(logPrefix, "unknown contractType of instId: %s", instId)
		return ""
	}
}

// btc,usdt -> BTC-USDT
func SpotTypeToInstId(baseCcy, quoteCcy string) string {
	return fmt.Sprintf("%s-%s", strings.ToUpper(baseCcy), strings.ToUpper(quoteCcy))
}

// BTC-USDT -> btc, usdt
func InstIdToSpotType(instId string) (baseCcy, quoteCcy string) {
	ss := strings.Split(instId, "-")
	return strings.ToLower(ss[0]), strings.ToLower(ss[1])
}

// BTC-USDT-SWAP/BTC-USD-SWAP -> BTC-USDT
func FutureInstId2SpotInstId(instId string) string {
	ss := strings.Split(instId, "-")
	if len(ss) >= 2 {
		return ss[0] + "-" + ss[1]
	}

	return ""
}

// BTC-USDT-SWAP -> btc
func InstIdToCcy(instId string) string {
	ss := strings.Split(instId, "-")
	return strings.ToLower(ss[0])
}

var accClientOrderId int32

func NewClientOrderId(purpose string) string {
	newId := atomic.AddInt32(&accClientOrderId, 1)
	return util.ToLetterNumberOnly(fmt.Sprintf("%05d%s", newId, purpose), 32)
}

var accAmendId int32

func NewAmendId() string {
	newId := atomic.AddInt32(&accAmendId, 1)
	return fmt.Sprintf("%05d", newId)
}
