/*
- @Author: aztec
- @Date: 2024-03-08 10:16:25
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package ibkrtws

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// 日志接口（将来应该封装起来）
type fnLog func(msg string)

var logInfoFn fnLog
var logDebugFn fnLog
var logErrorFn fnLog

type tolerateRecord struct {
	startTime time.Time
	count     int
}

var toleratedErrorRecordsMu sync.Mutex
var toleratedErrorRecords map[string]*tolerateRecord

func logDebug(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logDebugFn(msg)
}

func logInfo(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logInfoFn(msg)
}

func logError(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logErrorFn(msg)
}

func logErrorWithTolerate(id string, periodSec, maxCount int, logPrefix string, format string, params ...interface{}) {
	if toleratedErrorRecords == nil {
		toleratedErrorRecords = map[string]*tolerateRecord{}
	}

	var tr *tolerateRecord

	toleratedErrorRecordsMu.Lock()
	if _, ok := toleratedErrorRecords[id]; !ok {
		toleratedErrorRecords[id] = &tolerateRecord{}
	}
	tr = toleratedErrorRecords[id]
	toleratedErrorRecordsMu.Unlock()

	if time.Since(tr.startTime).Seconds() > float64(periodSec) {
		tr.count = 1
		tr.startTime = time.Now()
	} else {
		tr.count++
		if tr.count > maxCount {
			logError(logPrefix, format, params...)
		}
	}
}

// 交易所配置
type ExchangeConfig struct {
	Addr      string           `json:"addr"`
	Port      int              `json:"port"`
	Contracts []ContractConfig `json:"contracts"`
	Symbols   []string         `json:"-"`
	Currencys []string         `json:"-"`
}

func (e *ExchangeConfig) parse() {
	for _, ct := range e.Contracts {
		if !slices.Contains(e.Symbols, ct.Symbol) {
			e.Symbols = append(e.Symbols, ct.Symbol)
		}

		if !slices.Contains(e.Currencys, ct.Currency) {
			e.Currencys = append(e.Currencys, ct.Currency)
		}
	}
}

func (e *ExchangeConfig) findContractSeed(baseCcy, quoteCcy string) (ContractConfig, bool) {
	baseCcy = strings.ToUpper(baseCcy)
	quoteCcy = strings.ToUpper(quoteCcy)

	for _, c := range e.Contracts {
		if c.Symbol == baseCcy && c.Currency == quoteCcy {
			return c, true
		}
	}

	return ContractConfig{}, false
}

// 交易品种索引。用于查询完整的Contract细节，以及一些本地配置
type ContractConfig struct {
	Symbol                       string          `json:"symbol"`
	Currency                     string          `json:"currency"`
	SecType                      string          `json:"sectype"`
	Exchange                     string          `json:"exchange"`
	Tif                          string          `json:"tif"`
	MaxPriceDistValueToBestPrice decimal.Decimal `json:"maxPriceDist"`
	MaxPriceDistRatioToBestPrice decimal.Decimal `json:"maxPriceDistRatio"`
}

func (c ContractConfig) toTwsContract() twsmodel.Contract {
	return twsmodel.Contract{
		Symbol:   c.Symbol,
		Currency: c.Currency,
		SecType:  c.SecType,
		Exchange: c.Exchange,
	}
}

func tradingTimesFromString(t *common.TradingTimes, str string) {
	sDate := strings.Split(str, ";")
	for _, strDate := range sDate {
		if strDate == "" {
			continue
		}

		// strDate有两种情况：
		// 20240325:0930-20240325:1600表示正常的开盘、收盘时间
		// 20240324:CLOSED表示今日不开盘
		// 时区为美国东部时区
		strDateTime := strings.Split(strDate, "-")
		tts := common.TradingTimeSeg{}
		if len(strDateTime) == 2 {
			if t0, err := time.ParseInLocation("20060102:1504", strDateTime[0][:13], util.AmericaNYZone()); err == nil {
				if t1, err := time.ParseInLocation("20060102:1504", strDateTime[1][:13], util.AmericaNYZone()); err == nil {
					tts.OpenTime = t0
					tts.CloseTime = t1
				} else {
					logError(logPrefix, "parse trading time failed: %s", strDateTime[1])
				}
			} else {
				logError(logPrefix, "parse trading time failed: %s", strDateTime[0])
			}
		}
		(*t) = append(*t, tts)
	}
}
