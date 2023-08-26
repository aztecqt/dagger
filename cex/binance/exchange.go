/*
 * @Author: aztec
 * @Date: 2023-02-16 18:23:08
 * @Description: binance的总入口，实现common.CEx接口
 * 由于binance目前还没有统一账户，所以现货和合约是两套东西
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binance

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancespotapi"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/emirpasic/gods/sets/hashset"
)

const logPrefix = "Binance"
const exchangeName = "Binance"

var exchangeReady = false

type OnOrderSnapshotFn func(OrderSnapshot)

type Exchange struct {
	// 区分订单所属策略
	stratergyId int

	// 现货部分
	wsSpot           *binancespotapi.WsClient
	spotMarkets      map[string]*SpotMarket
	spotTraders      map[string]*SpotTrader
	spotMarketsSlice []common.SpotMarket
	spotTradersSlice []common.SpotTrader

	// 交易品种
	instrumentMgr *common.InstrumentMgr

	// 现货权益
	spotBalanceMgr *common.BalanceMgr

	// 现货订单更新的分发
	spotOrderSnapshotFns map[string] /*spot-symbol*/ OnOrderSnapshotFn
	muSpotOSFn           sync.Mutex
}

func (e *Exchange) Init(key, secret string, ecb func(e error)) {
	logger.LogImportant(logPrefix, "exchange starting...")

	e.spotMarkets = make(map[string]*SpotMarket)
	e.spotTraders = make(map[string]*SpotTrader)
	e.spotMarketsSlice = make([]common.SpotMarket, 0)
	e.spotTradersSlice = make([]common.SpotTrader, 0)

	e.stratergyId = int(time.Now().Unix())
	e.spotBalanceMgr = common.NewBalanceMgr()
	e.instrumentMgr = common.NewInstrumentMgr(logPrefix)
	e.spotOrderSnapshotFns = make(map[string]OnOrderSnapshotFn)

	// 初始化api
	logger.LogImportant(logPrefix, "init api...")
	binanceapi.Init(key, secret, binancespotapi.ServerTs)
	binanceapi.ErrorCallback = ecb

	// 获取所有交易对列表
	logger.LogImportant(logPrefix, "fetching spot instruments...")
	e.initSpotInstruments("")

	// 关闭所有订单
	logger.LogImportant(logPrefix, "close all spot orders...")
	e.CloseAllOrders()

	// 初始化现货账户权益
	logger.LogImportant(logPrefix, "initializing spot account info...")
	e.initSpotAccountInfo()

	// 启动ws，订阅各种数据
	logger.LogImportant(logPrefix, "starting spot websocket...")
	e.wsSpot = new(binancespotapi.WsClient)
	e.wsSpot.Start()
	e.wsSpot.SubscribeUserData(e.onWsAccountUpdate, e.onWsOrderUpdate)

	exchangeReady = true
}

// 初始化现货交易对信息
func (e *Exchange) initSpotInstruments(instId string) {
	resp, err := binancespotapi.GetExchangeInfo_Symbols(instId)
	if err == nil {
		for _, symbol := range resp.Symbols {
			ins := new(common.Instruments)
			ins.Id = symbol.Symbol
			ins.BaseCcy = strings.ToLower(symbol.BaseCcy)
			ins.QuoteCcy = strings.ToLower(symbol.QuoteCcy)

			if filter := symbol.FindFilterByType("PRICE_FILTER"); filter != nil {
				if v, ok := filter["tickSize"]; ok {
					ins.TickSize = util.String2DecimalPanic(v.(string))
				}
			}

			if filter := symbol.FindFilterByType("LOT_SIZE"); filter != nil {
				if v, ok := filter["minQty"]; ok {
					ins.MinSize = util.String2DecimalPanic(v.(string))
				}

				if v, ok := filter["stepSize"]; ok {
					ins.LotSize = util.String2DecimalPanic(v.(string))
				}
			}

			if filter := symbol.FindFilterByType("MIN_NOTIONAL"); filter != nil {
				if v, ok := filter["minNotional"]; ok {
					ins.MinValue = util.String2DecimalPanic(v.(string))
				}
			}

			if ins.TickSize.IsZero() || ins.LotSize.IsZero() || ins.MinSize.IsZero() {
				logger.LogPanic(logPrefix, "invalid instruments: %v", symbol)
			}

			e.instrumentMgr.Set(symbol.Symbol, ins)
		}
	} else {
		logger.LogPanic(logPrefix, "get spot symbols error: %s", err.Error())
	}
}

func (e *Exchange) findOrGetSpotInstrument(instId string) *common.Instruments {
	inst := e.instrumentMgr.Get(instId)
	if inst != nil {
		return inst
	} else {
		e.initSpotInstruments(instId)
		inst := e.instrumentMgr.Get(instId)
		return inst
	}
}

// 初始化现货账户权益
func (e *Exchange) initSpotAccountInfo() {
	accountInfo, err := binancespotapi.GetAccountInfo()
	if err == nil {
		ts := time.UnixMilli(accountInfo.Timestamp)
		for _, v := range accountInfo.Balances {
			if v.Free.IsPositive() || v.Frozen.IsPositive() {
				ccy := strings.ToLower(v.Asset)
				fmt.Printf("%s: free=%v, frozen=%v\n", ccy, v.Free, v.Frozen)
				e.spotBalanceMgr.RefreshBalance(ccy, v.Free, v.Frozen, ts)
			}
		}
	} else {
		logger.LogPanic(logPrefix, "get account info failed! err=%s", err.Error())
	}
}

// 刷新现货账户权益
func (e *Exchange) onWsAccountUpdate(msg interface{}) {
	au := msg.(binanceapi.WSPayload_AccountUpdate)
	ts := time.UnixMilli(au.AccountUpdateTimeStamp)
	for _, detail := range au.Detail {
		ccy := strings.ToLower(detail.AssetName)
		e.spotBalanceMgr.RefreshBalance(ccy, detail.Free, detail.Frozen, ts)
		b := e.spotBalanceMgr.FindBalance(ccy)
		fmt.Printf("%s: rights=%v, frozen=%v\n", ccy, b.Rights(), b.Frozen())
	}
}

// 订阅订单推送
func (e *Exchange) RegSpotOrderSnapshot(symbol string, fn OnOrderSnapshotFn) {
	e.muSpotOSFn.Lock()
	defer e.muSpotOSFn.Unlock()
	if _, ok := e.spotOrderSnapshotFns[symbol]; ok {
		logger.LogPanic(logPrefix, "order can only regist once. instID=%s", symbol)
	}
	e.spotOrderSnapshotFns[symbol] = fn
}

func (e *Exchange) UnregSpotOrderSnapshot(instID string) {
	e.muSpotOSFn.Lock()
	defer e.muSpotOSFn.Unlock()
	if _, ok := e.spotOrderSnapshotFns[instID]; ok {
		delete(e.spotOrderSnapshotFns, instID)
	}
}

// 订单推送处理
func (e *Exchange) onWsOrderUpdate(msg interface{}) {
	e.muSpotOSFn.Lock()
	defer e.muSpotOSFn.Unlock()
	ou := msg.(binanceapi.WSPayload_OrderUpdate)
	if fn, ok := e.spotOrderSnapshotFns[ou.Symblo]; ok {
		os := NewOrderSnapshotFromWsResponse(ou)
		fn(os)
	}
}

// 撤销所有订单
// 因为查询订单成本太高，这里就不用循环确认的方式了，而是采用一过性撤销，不检查结果
func (e *Exchange) CloseAllOrders() {
	logger.LogImportant(logPrefix, "closing open orders...")

	// 查询当前所有挂单
	symbolset := hashset.New()
	r0, emsg0, e0 := binancespotapi.GetOpenOrders("")
	if e0 != nil {
		logger.LogPanic(logPrefix, "GetOpenOrders failed: %s", e0.Error())
	} else if emsg0 != nil {
		logger.LogPanic(logPrefix, "GetOpenOrders failed, code=%d, msg=%s", emsg0.Code, emsg0.Message)
	}

	for _, os := range *r0 {
		symbolset.Add(os.Symbol)
	}

	symbols := symbolset.Values()
	for _, v := range symbols {
		symbol := v.(string)
		logger.LogImportant(logPrefix, "closing %s...", symbol)
		_, emsg1, e1 := binancespotapi.CancelOpenOrders(symbol)
		if e1 != nil {
			logger.LogPanic(logPrefix, "CancelOpenOrders failed: %s", e1.Error())
		} else if emsg1 != nil {
			logger.LogPanic(logPrefix, "CancelOpenOrders failed, code:%d, msg:%s", emsg1.Code, emsg1.Message)
		}
	}

	logger.LogImportant(logPrefix, "all open orders closed")
}

// #region 实现common.CEx接口
func (e *Exchange) Name() string {
	return exchangeName
}

func (e *Exchange) Instruments() []*common.Instruments {
	return e.instrumentMgr.GetAll()
}

func (e *Exchange) UseFutureMarket(ccy string, contractType string, withDepth bool) common.FutureMarket {
	return nil
}

func (e *Exchange) UseFutureTrader(ccy string, contractType string, lever int) common.FutureTrader {
	return nil
}

func (e *Exchange) UseSpotMarket(baseCcy string, quoteCcy string, withDepth bool) common.SpotMarket {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)

	m, ok := e.spotMarkets[instId]
	if ok {
		return m
	} else {
		inst := e.findOrGetSpotInstrument(instId)
		if inst == nil {
			logger.LogImportant(logPrefix, "unknown instId:%s", instId)
			return nil
		} else {
			m := new(SpotMarket)
			m.Init(e, instId, withDepth)
			e.spotMarkets[instId] = m
			e.spotMarketsSlice = append(e.spotMarketsSlice, m)
			return m
		}
	}
}

func (e *Exchange) UseSpotTrader(baseCcy string, quoteCcy string) common.SpotTrader {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)
	t, ok := e.spotTraders[instId]
	if ok {
		return t
	} else {
		t := new(SpotTrader)
		mi := e.UseSpotMarket(baseCcy, quoteCcy, true)
		if mi == nil {
			return nil
		} else {
			m := mi.(*SpotMarket)
			t.Init(e, e.stratergyId, m)
			e.spotTraders[instId] = t
			e.spotTradersSlice = append(e.spotTradersSlice, t)
			return t
		}
	}
}

func (e *Exchange) UseFundingFeeInfoObserver(maxLength int) common.FundingFeeObserver {
	return nil
}

func (e *Exchange) FundingFeeInfoObserver() common.FundingFeeObserver {
	return nil
}

func (e *Exchange) UseContractObserver(contractType string) common.ContractObserver {
	return nil
}

func (e *Exchange) FutureMarkets() []common.FutureMarket {
	return nil
}

func (e *Exchange) FutureTraders() []common.FutureTrader {
	return nil
}

func (e *Exchange) SpotMarkets() []common.SpotMarket {
	return e.spotMarketsSlice
}

func (e *Exchange) SpotTraders() []common.SpotTrader {
	return e.spotTradersSlice
}

func (e *Exchange) GetFutureDealHistory(ccy, contractType string, t0, t1 time.Time) []common.DealHistory {
	return nil

}

func (e *Exchange) GetSpotDealHistory(baseCcy, quoteCcy string, t0, t1 time.Time) []common.DealHistory {
	return nil
}

func (e *Exchange) Exit() {
	// 这样会停止一切下单行为
	exchangeReady = false

	// 撤销所有订单
	e.CloseAllOrders()
}

// #endregion
