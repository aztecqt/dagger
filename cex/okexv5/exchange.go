/*
 * @Author: aztec
 * @Date: 2022-03-30 09:33:49
 * @Description: okex交易所总入口。实现common.CEx接口
 * Exchange的一个重要职责，是负责初步解析ws推送数据，并分发给trader或者market
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package okexv5

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

const logPrefix = "OKEx"
const exchangeName = "OKEx"

var exchangeReady = false
var StratergyName string = ""
var _orderTag string = ""

func orderTag() string {
	if len(_orderTag) == 0 && len(StratergyName) > 0 {
		_orderTag = util.ToLetterNumberOnly(StratergyName, 16)
		logger.LogInfo(logPrefix, "order tag set to [%s]", _orderTag)
	}
	return _orderTag
}

type OnOrderSnapshotFn func(orderSnapshot) // 订单刷新回调

type Exchange struct {
	ws *okexv5api.WsClient

	// 配置
	excfg ExchangeConfig

	// 所有行情和交易器
	futureMarkets      map[string] /*instId*/ *FutureMarket
	futureTraders      map[string] /*instId*/ *FutureTrader
	futureMarketsSlice []common.FutureMarket
	futureTradersSlice []common.FutureTrader

	spotMarkets      map[string]*SpotMarket
	spotTraders      map[string]*SpotTrader
	spotMarketsSlice []common.SpotMarket
	spotTradersSlice []common.SpotTrader

	fundingFeeObserver *FundingFeeObserver

	// contractType->observer
	contractObservers map[string]*ContractObserver

	// 交易品种
	instrumentMgr *common.InstrumentMgr

	// 用户权益
	balanceMgr *common.BalanceMgr

	// 仓位
	ctPositions map[string] /*instId*/ *common.PositionImpl
	muPosition  sync.RWMutex

	// 订单。订单保存在trader中，ex不需要持有
	// ex负责分发现货订单更新
	orderSnapshotFns map[string] /*instId*/ OnOrderSnapshotFn
	muOSFn           sync.RWMutex

	// 所有交易对的行情信息的拉取和通知。根据配置决定是否启用
	// 行情推送给注册过的回调函数
	tickerCallbacks    map[string][]func(t okexv5api.TickerResp)
	muTickerCallback   sync.Mutex
	tickerRestInstType map[string]int
}

func (e *Exchange) Init(key, secret, pass string, excfg *ExchangeConfig, ecb func(e error)) {
	logger.LogImportant(logPrefix, "exchange starting...")
	e.excfg.init()
	if excfg != nil {
		e.excfg = *excfg
	}

	e.futureMarkets = make(map[string]*FutureMarket)
	e.futureTraders = make(map[string]*FutureTrader)
	e.futureMarketsSlice = make([]common.FutureMarket, 0)
	e.futureTradersSlice = make([]common.FutureTrader, 0)

	e.spotMarkets = make(map[string]*SpotMarket)
	e.spotTraders = make(map[string]*SpotTrader)
	e.spotMarketsSlice = make([]common.SpotMarket, 0)
	e.spotTradersSlice = make([]common.SpotTrader, 0)

	e.balanceMgr = common.NewBalanceMgr()
	e.instrumentMgr = common.NewInstrumentMgr(logPrefix)
	e.ctPositions = make(map[string]*common.PositionImpl)
	e.orderSnapshotFns = make(map[string]OnOrderSnapshotFn)
	e.contractObservers = make(map[string]*ContractObserver)
	e.tickerCallbacks = make(map[string][]func(t okexv5api.TickerResp))
	e.tickerRestInstType = make(map[string]int)

	// 初始化api
	logger.LogImportant(logPrefix, "init api...")
	okexv5api.Init(key, secret, pass)
	okexv5api.ErrorCallback = ecb

	// 获取所有交易对列表
	logger.LogImportant(logPrefix, "fetching instruments...")
	e.initInstruments()

	if okexv5api.HasKey() {
		// 撤销所有订单
		logger.LogImportant(logPrefix, "closing pending orders...")
		e.CloseAllOrders()

		// 检查账户配置
		logger.LogImportant(logPrefix, "checking account config...")
		e.checkAccountConfig()
	}

	// 启动ws
	logger.LogImportant(logPrefix, "starting websocket...")
	e.ws = new(okexv5api.WsClient)
	e.ws.Start()

	// 启动rest拉取ticker
	if e.excfg.TickerFromRest {
		logger.LogImportant(logPrefix, "start ticker-rest thread")
		go e.updateTickersByRest()
	}

	if okexv5api.HasKey() {
		// 登录
		e.ws.Login()

		wg := sync.WaitGroup{}
		wg.Add(2)

		// 订阅account
		// account数据由exchange统一订阅，然后各个FutureTrader通过二次订阅获取到自己想要的数据
		go e.updateAccount(&wg)

		// 订阅position，处理逻辑跟account一样
		go e.updatePosition(&wg)

		// 订阅订单，处理逻辑类似。区别是instId放在每个order数据单元里，而不是消息头部
		go e.updateOrders()

		wg.Wait()
	}

	exchangeReady = true
	logger.LogImportant(logPrefix, "exchange started")
}

// #region 实现common.CEx接口
func (e *Exchange) Name() string {
	return exchangeName
}

func (e *Exchange) Instruments() []*common.Instruments {
	return e.instrumentMgr.GetAll()
}

func (e *Exchange) GetInstrument(id string) *common.Instruments {
	return e.instrumentMgr.Get(id)
}

func (e *Exchange) UseFutureMarket(symbol string, contractType string) common.FutureMarket {
	instId := CCyCttypeToInstId(symbol, contractType)
	instType := util.ValueIf(strings.Contains(contractType, "swap"), "SWAP", "FUTURES")

	if e.excfg.TickerFromRest {
		e.tickerRestInstType[instType] = 1
	}

	m, ok := e.futureMarkets[instId]
	if ok {
		return m
	} else {
		var inst = e.findOrGetInstrument(instType, instId)
		if inst == nil {
			logger.LogImportant(logPrefix, "unknown instId:%s", instId)
			return nil
		} else {
			m := new(FutureMarket)
			m.Init(e, instId, e.excfg.DepthFromTicker, e.excfg.TickerFromRest)
			e.futureMarkets[instId] = m
			e.futureMarketsSlice = append(e.futureMarketsSlice, m)
			return m
		}
	}
}

func (e *Exchange) UseFutureTrader(symbol string, contractType string, lever int) common.FutureTrader {
	if lever == 0 {
		lever = 10
		if contractType == "usdt_swap" || contractType == "usd_swap" {
			if symbol == "btc" || symbol == "eth" {
				lever = 50
			}
		}
	}

	instId := CCyCttypeToInstId(symbol, contractType)
	t, ok := e.futureTraders[instId]
	if ok {
		return t
	} else {
		t := new(FutureTrader)
		mi := e.UseFutureMarket(symbol, contractType)
		if mi == nil {
			return nil
		} else {
			m := mi.(*FutureMarket)
			t.Init(e, orderTag(), m, lever)
			e.futureTraders[instId] = t
			e.futureTradersSlice = append(e.futureTradersSlice, t)
			return t
		}
	}
}

func (e *Exchange) UseSpotMarket(baseCcy string, quoteCcy string) common.SpotMarket {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)

	m, ok := e.spotMarkets[instId]
	if ok {
		return m
	} else {
		inst := e.findOrGetInstrument("SPOT", instId)
		if inst == nil {
			logger.LogImportant(logPrefix, "unknown instId:%s", instId)
			return nil
		} else {
			m := new(SpotMarket)
			m.Init(e, instId, inst.BaseCcy, inst.QuoteCcy, e.excfg.DepthFromTicker, e.excfg.TickerFromRest)
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
		mi := e.UseSpotMarket(baseCcy, quoteCcy)
		if mi == nil {
			return nil
		} else {
			m := mi.(*SpotMarket)
			t.Init(e, orderTag(), m)
			e.spotTraders[instId] = t
			e.spotTradersSlice = append(e.spotTradersSlice, t)
			return t
		}
	}
}

func (e *Exchange) UseFundingFeeInfoObserver(maxLength int) common.FundingFeeObserver {
	if e.fundingFeeObserver == nil {
		e.fundingFeeObserver = new(FundingFeeObserver)
		e.fundingFeeObserver.init(e)

		// 从所有swap合约中，挑选出日成交量前xx名
		resp, err := okexv5api.GetTickers("SWAP")
		if err == nil && resp.Code == "0" {
			vol2InstId := util.NewFloatTreeMapInverted()
			for _, ticker := range resp.Data {
				if !strings.Contains(ticker.InstId, "USDT") {
					price := util.String2DecimalPanic(ticker.Last).InexactFloat64()
					vol24Ccy := util.String2DecimalPanic(ticker.VolCcy24h).InexactFloat64()
					vol24Usd := vol24Ccy * price
					vol2InstId.Put(vol24Usd, ticker.InstId)
				}
			}

			it := vol2InstId.Iterator()
			count := 0
			for it.Next() {
				e.fundingFeeObserver.AddType(it.Value().(string))
				count++
				if count >= maxLength {
					break
				}
			}
		} else {
			logger.LogInfo(logPrefix, "get all tickers failed, err=%s", err.Error())
		}
	}

	return e.fundingFeeObserver
}

func (e *Exchange) FundingFeeInfoObserver() common.FundingFeeObserver {
	return e.fundingFeeObserver
}

func (e *Exchange) UseContractObserver(contractType string) common.ContractObserver {
	if _, ok := e.contractObservers[contractType]; !ok {
		os := new(ContractObserver)
		os.Init(contractType)
		e.contractObservers[contractType] = os
	}

	os, _ := e.contractObservers[contractType]
	return os
}

func (e *Exchange) FutureMarkets() []common.FutureMarket {
	return e.futureMarketsSlice
}

func (e *Exchange) FutureTraders() []common.FutureTrader {
	return e.futureTradersSlice
}

func (e *Exchange) SpotMarkets() []common.SpotMarket {
	return e.spotMarketsSlice
}

func (e *Exchange) SpotTraders() []common.SpotTrader {
	return e.spotTradersSlice
}

func (e *Exchange) GetSpotDealHistory(baseCcy, quoteCcy string, t0, t1 time.Time) []common.DealHistory {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)
	return e.GetDealHistory(instId, t0, t1)
}

func (e *Exchange) GetFutureDealHistory(symbol, contractType string, t0, t1 time.Time) []common.DealHistory {
	instId := CCyCttypeToInstId(symbol, contractType)
	return e.GetDealHistory(instId, t0, t1)
}

func (e *Exchange) GetDealHistory(instId string, t0, t1 time.Time) []common.DealHistory {
	totalSec := float64(t1.Unix() - t0.Unix())
	deals := make([]common.DealHistory, 0)
	for {
		resp, err := okexv5api.GetFillsHistory(instId, t0, t1)
		if err != nil {
			logger.LogImportant(logPrefix, "get fills failed: %s", err.Error())
			return nil
		} else {
			c := 0
			for i := 0; i < len(resp.Data); i++ {
				f := resp.Data[i]
				deal := common.DealHistory{}
				deal.Time = f.FillTime
				if f.Side == "buy" {
					deal.Dir = common.OrderDir_Buy
				} else if f.Side == "sell" {
					deal.Dir = common.OrderDir_Sell
				}
				deal.Price = f.Price
				deal.Amount = f.Size
				if len(deals) == 0 || deals[len(deals)-1].Time.After(deal.Time) {
					c++
					deals = append(deals, deal)
					t1 = deal.Time
				}
			}

			leftSec := float64(t1.Unix() - t0.Unix())
			fmt.Printf("GetDealHistory progress: %.2f%%\n", (leftSec/totalSec)*100)

			if c == 0 {
				break
			} else {
				time.Sleep(time.Millisecond * time.Duration(200))
			}
		}
	}

	rdeals := make([]common.DealHistory, 0, len(deals))
	for i := 0; i < len(deals); i++ {
		rdeals = append(rdeals, deals[len(deals)-i-1])
	}
	return rdeals
}

func (e *Exchange) GetSpotKline(baseCcy, quoteCcy string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)
	return GetKline(instId, t0, t1, intervalSec)
}

func (e *Exchange) GetFutureKline(symbol, contractType string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	instId := CCyCttypeToInstId(symbol, contractType)
	return GetKline(instId, t0, t1, intervalSec)
}

func GetKline(instId string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	bar := "1m"
	switch intervalSec {
	case 60:
		bar = "1m"
	case 60 * 3:
		bar = "3m"
	case 60 * 5:
		bar = "5m"
	case 60 * 15:
		bar = "15m"
	case 60 * 30:
		bar = "30m"
	case 3600:
		bar = "1H"
	case 3600 * 2:
		bar = "2H"
	case 3600 * 4:
		bar = "4H"
	case 86400:
		bar = "1D"
	case 86400 * 7:
		bar = "1W"
	default:
		logger.LogPanic(logPrefix, "invalid kline intervalsec for okx: %d", intervalSec)
	}

	for {
		tEnd := t1
		temp := make([]common.KUnit, 0)
		valid := true
		for {
			resp, err := okexv5api.GetKlineBefore(instId, tEnd, bar, 0)
			if err != nil {
				logger.LogImportant(logPrefix, err.Error())
				valid = false
				break
			} else if resp.Code != "0" {
				logger.LogImportant(logPrefix, resp.Msg)
				return []common.KUnit{}
			} else {
				if len(resp.Data) == 0 {
					break
				}

				for _, ku := range resp.Data {
					ku2 := common.KUnit{
						Time:         time.UnixMilli(ku.TS),
						OpenPrice:    ku.Open,
						ClosePrice:   ku.Close,
						HighestPrice: ku.High,
						LowestPrice:  ku.Low,
						VolumeUSD:    ku.VolumeUSD,
					}

					temp = append(temp, ku2)
					tEnd = ku2.Time
					if tEnd.Before(t0) {
						break
					}
				}

				if tEnd.Before(t0) {
					break
				}
			}
		}

		if valid && len(temp) > 0 {
			rst := make([]common.KUnit, 0, len(temp))
			for i := len(temp) - 1; i >= 0; i-- {
				rst = append(rst, temp[i])
			}

			return rst
		} else {
			logger.LogImportant(logPrefix, "get kline failed, instId=%s, t0=%s, t1=%s", instId, t0.Format(time.DateTime), t1.Format(time.DateTime))
			time.Sleep(time.Second * 3)
		}
	}
}

func (e *Exchange) Exit() {
	// 这样会停止一切下单行为
	exchangeReady = false

	// 撤销所有订单
	e.CloseAllOrders()
}

// #endregion 实现common.CEx接口

// #region account
func (e *Exchange) updateAccount(wg *sync.WaitGroup) {
	// 订阅Account，20秒收不到数据则超时重连
	timeout := time.NewTicker(time.Second * 20)
	accOk := false
	s := e.ws.SubscribeAccountBalance(func(resp interface{}) {
		r := resp.(okexv5api.AccountBalanceWsResp)
		arr := r.Data[0].Details

		// 解析并推送
		func() {
			e.balanceMgr.Lock()
			defer e.balanceMgr.Unlock()
			defer timeout.Reset(time.Second * 20)

			for i := 0; i < len(arr); i++ {
				logger.LogDebug(logPrefix, "recv account: %v", arr[i])
				ccy := strings.ToLower(arr[i].Currency)
				b := e.balanceMgr.FindBalanceUnsafe(ccy)
				rights := util.String2DecimalPanic(arr[i].Eq)
				frozen := util.String2DecimalPanic(arr[i].Frozen)
				updateTime := util.ConvetUnix13StrToTimePanic(arr[i].UTime)
				b.RefreshRights(rights, frozen, updateTime)
			}
		}()

		if !accOk {
			accOk = true
			wg.Done()
		}
	})

	for {
		<-timeout.C
		logger.LogInfo(logPrefix, "account time out, re-subscribe it")
		s.Reset()
	}
}

// #endregion account

// #region position
func (e *Exchange) findPosition(instId string) *common.PositionImpl {
	e.muPosition.Lock()
	defer e.muPosition.Unlock()

	var p *common.PositionImpl

	if _, ok := e.ctPositions[instId]; !ok {
		e.ctPositions[instId] = common.NewPositionImpl(instId)
	}
	p = e.ctPositions[instId]

	return p
}

func (e *Exchange) updatePosition(wg *sync.WaitGroup) {
	index := 0
	timeout := time.NewTicker(time.Second * 20)
	posOk := false

	s := e.ws.SubscribePosition(func(resp interface{}) {
		r := resp.(okexv5api.PositionWsResp)
		arr := r.Data

		// 解析并推送
		func() {
			defer timeout.Reset(time.Second * 20)

			// 如果是首次推送，初始化所有注册过的仓位
			index++
			if index == 1 {
				e.muPosition.RLock()
				for _, p := range e.ctPositions {
					p.RefreshLong(decimal.Zero, decimal.Zero, time.Now())
					p.RefreshShort(decimal.Zero, decimal.Zero, time.Now())
				}
				e.muPosition.RUnlock()
			}

			for i := 0; i < len(arr); i++ {
				d := arr[i]
				if d.MgnMode == "cross" && (d.InstType == "SWAP" || d.InstType == "FUTURES") {
					position := e.findPosition(d.InstId)
					size := util.String2DecimalPanic(d.Pos)
					avgPx := util.String2DecimalPanicUnless(d.AvgPx, "")
					time := util.ConvetUnix13StrToTimePanic(d.UTime)
					if d.PosSide == "net" {
						// 仅支持双向模式
					} else if d.PosSide == "long" {
						position.RefreshLong(size, avgPx, time)
					} else if d.PosSide == "short" {
						position.RefreshShort(size, avgPx, time)
					}
				}
			}
		}()

		if !posOk {
			posOk = true
			wg.Done()
		}
	})

	for {
		<-timeout.C
		logger.LogInfo(logPrefix, "position time out, re-subscribe it")
		s.Reset()
	}
}

// #endregion position

// #region orders
func (e *Exchange) RegOrderSnapshot(instID string, fn OnOrderSnapshotFn) {
	e.muOSFn.Lock()
	defer e.muOSFn.Unlock()
	if _, ok := e.orderSnapshotFns[instID]; ok {
		logger.LogPanic(logPrefix, "order can only regist once. instID=%s", instID)
	}
	e.orderSnapshotFns[instID] = fn
}

func (e *Exchange) UnregOrderSnapshot(instID string) {
	e.muOSFn.Lock()
	defer e.muOSFn.Unlock()
	if _, ok := e.orderSnapshotFns[instID]; ok {
		delete(e.orderSnapshotFns, instID)
	}
}

func (e *Exchange) updateOrders() {
	timeout := time.NewTicker(time.Second * 20)
	s := e.ws.SubscribeOrders(func(resp interface{}) {
		r := resp.(okexv5api.OrderWsResp)
		arr := r.Data

		// 解析并推送
		func() {
			e.muOSFn.RLock()
			defer e.muOSFn.RUnlock()
			defer timeout.Reset(time.Second * 20)

			// 根据instId进行推送
			for i := 0; i < len(arr); i++ {
				d := arr[i]
				if fn, ok := e.orderSnapshotFns[d.InstId]; ok {
					os := orderSnapshot{}
					os.localTime = r.LocalTime
					os.Parse(d, "ws")
					fn(os)
				}
			}
		}()
	})

	for {
		<-timeout.C
		s.Reset()
	}
}

// #endregion

// #region 其他
func (e *Exchange) initInstruments() {
	logger.LogImportant(logPrefix, "fetching instruments...SPOT")
	e.processInstruments("SPOT")
	logger.LogImportant(logPrefix, "fetching instruments...SWAP")
	e.processInstruments("SWAP")
	logger.LogImportant(logPrefix, "fetching instruments...FUTURES")
	e.processInstruments("FUTURES")
}

func (e *Exchange) processInstruments(instType string) {
	resp, err := okexv5api.GetInstruments(instType)
	if err == nil {
		for _, data := range resp.Data {
			ins := new(common.Instruments)
			instId := data.InstID
			ins.Id = instId
			ins.BaseCcy = strings.ToLower(data.BaseCcy)
			ins.QuoteCcy = strings.ToLower(data.QuoteCcy)
			ins.CtSettleCcy = strings.ToLower(data.SettleCcy)
			ins.CtValCcy = strings.ToLower(data.CtValCcy)
			ins.CtVal, _ = util.String2Decimal(data.CtVal)
			ins.ExpTime, _ = util.ConvetUnix13StrToTime(data.ExpTime)
			ins.Lever, _ = strconv.Atoi(data.Lever)
			ins.TickSize = util.String2DecimalPanic(data.TickSize)
			ins.LotSize = util.String2DecimalPanic(data.LotSz)
			ins.MinSize = util.String2DecimalPanic(data.MinSz)

			if strings.Contains(instId, "USD-SWAP") {
				// usd合约
				ins.CtSymbol = ins.CtSettleCcy
				ins.CtType = "usd_swap"
			} else if strings.Contains(instId, "USDT-SWAP") {
				// usdt合约
				ins.CtSymbol = ins.CtValCcy
				ins.CtType = "usdt_swap"
			} else {
				// 交割合约暂不支持
			}

			// 记录
			e.instrumentMgr.Set(instId, ins)
		}
	} else {
		logger.LogPanic(logPrefix, "can't get instruments of type [%s]", instType)
	}
}

func (e *Exchange) findOrGetInstrument(instType, instId string) *common.Instruments {
	inst := e.instrumentMgr.Get(instId)
	if inst != nil {
		return inst
	} else {
		e.processInstruments(instType)
		inst := e.instrumentMgr.Get(instId)
		return inst
	}
}

func (e *Exchange) checkAccountConfig() {
	resp, err := okexv5api.GetAccountConfig()
	if err == nil {
		if resp.Data[0].AccLevel != "2" { // "2"代表单币种保证金模式
			logger.LogPanic(logPrefix, "check account config failed：目前仅支持单币种保证金模式，请修改账户配置")
		} else if resp.Data[0].PosMode != "long_short_mode" {
			logger.LogPanic(logPrefix, "check account config failed：目前仅支持双向持仓模式，请修改账户配置")
		} else {
			logger.LogImportant(logPrefix, "check account config success")
		}
	} else {
		logger.LogPanic(logPrefix, "check account config failed! err:%s", err.Error())
	}
}

func (e *Exchange) CloseAllOrders() {
	for i := 0; ; i++ {
		resp, err := okexv5api.GetPendingOrders("")
		if err == nil {
			if resp.Code == "0" {
				orders := make([]okexv5api.OrderResp, 0)
				for _, d := range resp.Data {
					if d.Tag == orderTag() {
						orders = append(orders, d)
					}
				}

				if len(orders) == 0 {
					// 订单撤销完毕
					logger.LogImportant(logPrefix, "all pending orders closed")
					break
				} else {
					// 撤销这些订单
					cancelReqs := make([]okexv5api.CancelBatchOrderRestReq, 0, 20)
					for _, d := range orders {
						req := okexv5api.CancelBatchOrderRestReq{
							InstId:  d.InstId,
							OrderId: d.OrderId,
						}
						cancelReqs = append(cancelReqs, req)
					}
					respC, err := okexv5api.CancelOrderBatch(cancelReqs)
					if err != nil {
						logger.LogImportant(logPrefix, "cancel batch order failed, err=%s", err.Error())
					} else {
						if respC.Code != "0" {
							logger.LogImportant(logPrefix, "cancel batch order failed, resp=%v", resp)
						}
					}
				}
			} else {
				logger.LogImportant(logPrefix, "get pending order failed, resp=%v", resp)
			}
		} else {
			logger.LogImportant(logPrefix, "get pending order failed, err=%s", err.Error())
		}

		time.Sleep(time.Millisecond * 100)
	}
}

func (e *Exchange) registerTickerCallback(instId string, fn func(t okexv5api.TickerResp)) {
	e.muTickerCallback.Lock()
	defer e.muTickerCallback.Unlock()

	if cbs, ok := e.tickerCallbacks[instId]; ok {
		cbs = append(cbs, fn)
		e.tickerCallbacks[instId] = cbs
	} else {
		cbs := []func(t okexv5api.TickerResp){fn}
		e.tickerCallbacks[instId] = cbs
	}
}

func (e *Exchange) updateTickersByRest() {
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		<-ticker.C
		var toInvokeFn [][]func(t okexv5api.TickerResp)
		var toInvokeTk []okexv5api.TickerResp
		for instType := range e.tickerRestInstType {
			if resp, err := okexv5api.GetTickers(instType); err == nil {
				for _, tk := range resp.Data {
					if cbs, ok := e.tickerCallbacks[tk.InstId]; ok {
						toInvokeFn = append(toInvokeFn, cbs)
						toInvokeTk = append(toInvokeTk, tk)
					}
				}
				logger.LogInfo(logPrefix, "get %d tickers", len(resp.Data)) // debug
			} else {
				logger.LogImportant(logPrefix, "get ticker by instType failed, err=%s", err.Error())
			}
		}

		// 调用
		l := len(toInvokeFn)
		for i := 0; i < l; i++ {
			for _, fn := range toInvokeFn[i] {
				fn(toInvokeTk[i])
			}
		}
	}
}

// #endregion
