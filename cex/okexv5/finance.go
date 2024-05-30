/*
 * @Author: aztec
 * @Date: 2024-03-28 11:48:30
 * @Description: 实现common.finance接口
 *
 * Copyright (c) 2024 by aztec, All Rights Reserved.
 */
package okexv5

import (
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

type Finance struct {
	sync.Mutex
	apyOfCcy map[string]decimal.Decimal
	balOfCcy map[string]decimal.Decimal
}

func (f *Finance) init() {
	f.apyOfCcy = make(map[string]decimal.Decimal)
	f.balOfCcy = make(map[string]decimal.Decimal)

	// 首次刷新
	f.refreshApy()
	f.refreshBalance()

	// 持续刷新
	go func() {
		for {
			time.Sleep(time.Minute)
			f.refreshApy()
			f.refreshBalance()
		}
	}()
}

func (f *Finance) refreshApy() {
	if resp, err := okexv5api.GetMarketLendingRateSummary(""); err == nil {
		if resp.Code == "0" {
			f.Lock()
			for _, d := range resp.Data {
				f.apyOfCcy[strings.ToLower(d.Ccy)] = d.EstRate
			}
			defer f.Unlock()
		} else {
			logger.LogImportant(logPrefix, "refresh apy failed: %s", resp.Msg)
		}
	} else {
		logger.LogImportant(logPrefix, "refresh apy failed: %s", err.Error())
	}
}

func (f *Finance) refreshBalance() {
	if resp, err := okexv5api.GetFinanceSavingBalance(""); err == nil {
		if resp.Code == "0" {
			f.Lock()
			for _, d := range resp.Data {
				f.balOfCcy[strings.ToLower(d.Ccy)] = d.Amount
			}
			defer f.Unlock()
		} else {
			logger.LogImportant(logPrefix, "refresh balance failed: %s", resp.Msg)
		}
	} else {
		logger.LogImportant(logPrefix, "refresh balance failed: %s", err.Error())
	}
}

func (f *Finance) GetSavingApy(ccy string) decimal.Decimal {
	f.Lock()
	defer f.Unlock()
	if v, ok := f.apyOfCcy[ccy]; ok {
		return v
	} else {
		return decimal.Zero
	}
}

func (f *Finance) GetSavedBalance(ccy string) decimal.Decimal {
	f.Lock()
	defer f.Unlock()
	if v, ok := f.balOfCcy[ccy]; ok {
		return v
	} else {
		return decimal.Zero
	}
}

func (f *Finance) Save(ccy string, amount decimal.Decimal) bool {
	ccy = strings.ToUpper(ccy)
	defer f.refreshBalance()

	// 先把资金划转到资产账户
	if resp, err := okexv5api.Transfer(ccy, amount, true); err != nil {
		logger.LogImportant(logPrefix, "transfer %v %s to assert failed: %s", amount, ccy, err.Error())
		return false
	} else if resp.Code != "0" {
		logger.LogImportant(logPrefix, "transfer %v %s to assert failed: %s", amount, ccy, resp.Msg)
		return false
	}

	// 质押
	success := true
	for i := 0; i < 10; i++ {
		if resp, err := okexv5api.FinanceSavingPurchaseRedempt(ccy, amount, true); err != nil {
			logger.LogImportant(logPrefix, "puchase %v %s failed: %s", amount, ccy, err.Error())
			success = false
		} else if resp.Code != "0" {
			logger.LogImportant(logPrefix, "puchase %v %s failed: %s", amount, ccy, resp.Msg)
			success = false
		} else {
			logger.LogImportant(logPrefix, "puchase %v %s success", amount, ccy)
			success = true
			break
		}

		time.Sleep(time.Second)
	}

	return success
}

func (f *Finance) Draw(ccy string, amount decimal.Decimal) bool {
	ccy = strings.ToUpper(ccy)
	defer f.refreshBalance()

	// 赎回
	if resp, err := okexv5api.FinanceSavingPurchaseRedempt(ccy, amount, false); err != nil {
		logger.LogImportant(logPrefix, "redempt %v %s failed: %s", amount, ccy, err.Error())
		return false
	} else if resp.Code != "0" {
		logger.LogImportant(logPrefix, "redempt %v %s failed: %s", amount, ccy, resp.Msg)
		return false
	}

	success := false
	for i := 0; i < 10; i++ {
		// 转账
		if resp, err := okexv5api.Transfer(ccy, amount, false); err != nil {
			logger.LogImportant(logPrefix, "transfer %v %s back from assert failed: %s", amount, ccy, err.Error())
			success = false
		} else if resp.Code != "0" {
			logger.LogImportant(logPrefix, "transfer %v %s back from assert failed: %s", amount, ccy, resp.Msg)
			success = false
		} else {
			logger.LogImportant(logPrefix, "transfer %v %s back from assert success", amount, ccy)
			success = true
			break
		}

		time.Sleep(time.Second)
	}

	return success
}
