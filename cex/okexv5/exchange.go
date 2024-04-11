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
	"github.com/aztecqt/dagger/api/okexv5api/cachedok"
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
	excfg        ExchangeConfig
	singleMargin bool

	// 所有行情和交易器
	futureMarkets      map[string] /*instId*/ *FutureMarket
	futureTraders      map[string] /*instId*/ *FutureTrader
	futureMarketsSlice []common.FutureMarket
	futureTradersSlice []common.FutureTrader

	spotMarkets      map[string]*SpotMarket
	spotTraders      map[string]*SpotTrader
	spotMarketsSlice []common.SpotMarket
	spotTradersSlice []common.SpotTrader

	finance *Finance

	fundingFeeObserver *FundingFeeObserver

	// contractType->observer
	contractObservers map[string]*ContractObserver

	// 交易品种
	instrumentMgr *common.InstrumentMgr

	// 用户权益，主要针对现货，系统会自主计算币种余额，并跟交易所对齐
	balanceMgr *common.BalanceMgr

	// 交易所返回的账号信息，这里主要是为了取统一账户的几个关键数据
	accountBal okexv5api.AccountBalanceResp

	// 仓位
	// instId->pos
	ctPositions       map[string]*common.PositionImpl
	muPosition        sync.RWMutex
	positionInstTypes map[string]int

	// 最大可用，分为AvailBuy和AvailSell
	// 对于现货/杠杆，AvailBuy/AvailSell就是可用U/可用币（杠杆的话还包括借币的额度）
	// 对于合约，就是开多/开空所能使用的最大保证金数量
	// 现货可以直接算出这些值，但诸如组合保证金模式下的合约、杠杆等，很难计算这些值，所以改用交易所接口
	maxAvailable               map[string]okexv5api.MaxAvailableSizeResp
	spotInstIdsForMaxAvail     []string
	contractInstIdsForMaxAvail []string
	muMaxAvailable             sync.RWMutex

	// 订单。订单保存在trader中，ex不需要持有
	// ex负责分发现货订单更新
	// instId->fn
	orderSnapshotFns map[string]OnOrderSnapshotFn
	muOSFn           sync.RWMutex

	// 所有交易对的行情信息的拉取和通知。根据配置决定是否启用
	// 行情推送给注册过的回调函数
	tickerCallbacks           map[string][]func(tk okexv5api.TickerResp)
	tickerCallbacksOfInstType map[string][]func(tks []okexv5api.TickerResp)
	muTickerCallback          sync.Mutex
	tickerRestInstType        map[string]int
}

func (e *Exchange) Init(key, secret, pass string, excfg *ExchangeConfig, ecb func(e error)) {
	logger.LogImportant(logPrefix, "exchange starting...")
	e.excfg = newExchangeConfig()
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
	e.positionInstTypes = make(map[string]int)
	e.orderSnapshotFns = make(map[string]OnOrderSnapshotFn)
	e.contractObservers = make(map[string]*ContractObserver)
	e.tickerCallbacks = make(map[string][]func(t okexv5api.TickerResp))
	e.tickerCallbacksOfInstType = make(map[string][]func(tks []okexv5api.TickerResp))
	e.tickerRestInstType = make(map[string]int)
	e.maxAvailable = make(map[string]okexv5api.MaxAvailableSizeResp)

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
		// account数据由exchange统一订阅并处理
		go e.updateAccount(&wg)

		// 订阅position，处理逻辑跟account一样
		go e.updatePosition(&wg)

		// 订阅订单，处理逻辑类似。区别是instId放在每个order数据单元里，而不是消息头部
		go e.updateOrders()

		// 订阅市场爆仓订单
		go e.updateLiquidationOrders()

		// 单币种证金模式下，maxAvailable可以直接计算出来，其他模式下需要从api获取
		if !e.isSingleMarginMode() {
			go e.updateMaxAvalilable()
		}

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

func (e *Exchange) GetSpotInstrument(baseCcy, quoteCcy string) *common.Instruments {
	return e.instrumentMgr.Get(SpotTypeToInstId(baseCcy, quoteCcy))
}

func (e *Exchange) GetFutureInstrument(symbol, contractType string) *common.Instruments {
	return e.instrumentMgr.Get(CCyCttypeToInstId(symbol, contractType))
}

func (e *Exchange) GetUniAccRisk() common.UniAccRisk {
	// 根据全仓账户的维持保证金率来计算
	// 目前三档的标准是写死的。将来有需求，可以改为配置
	risk := common.UniAccRisk{
		Details:        make(map[string]string),
		PositionValue:  e.accountBal.PositionValue,
		MaintainMargin: e.accountBal.MaintainMargin,
		TotalMargin:    e.accountBal.AdjEq,
	}

	uniMmr := e.accountBal.MarginRatio.InexactFloat64()
	if uniMmr > 10 {
		risk.Level = common.UniAccRiskLevel_Safe
	} else if uniMmr > 3 {
		risk.Level = common.UniAccRiskLevel_Warning
	} else {
		risk.Level = common.UniAccRiskLevel_Danger
	}

	risk.Details["margin"] = fmt.Sprintf("$%.2f", e.accountBal.AdjEq.InexactFloat64())
	risk.Details["maintain margin"] = fmt.Sprintf("$%.2f", e.accountBal.MaintainMargin.InexactFloat64())
	risk.Details["mmr"] = fmt.Sprintf("%.1f%%", e.accountBal.MarginRatio.InexactFloat64()*100)
	risk.Details["total equity"] = fmt.Sprintf("$%.2f", e.accountBal.TotalEq.InexactFloat64())
	risk.Details["leverage"] = fmt.Sprintf("%.2f", e.accountBal.PositionValue.Div(e.accountBal.AdjEq).InexactFloat64())
	return risk
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
			m.Init(e, *inst, e.excfg.DepthFromTicker, e.excfg.TickerFromRest)
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

	if contractType == "usdt_swap" || contractType == "usd_swap" {
		e.positionInstTypes["SWAP"] = 1
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

			// 准备刷新其最大可用（U本位查一次就好）
			if strings.Contains(instId, "USDT") {
				e.contractInstIdsForMaxAvail = append(e.contractInstIdsForMaxAvail, "BTC-USDT-SWAP")
			} else {
				e.contractInstIdsForMaxAvail = append(e.contractInstIdsForMaxAvail, instId)
			}

			// 等最大可用就绪
			for {
				if _, ok := e.getMaxAvailable(instId); ok {
					break
				} else {
					time.Sleep(time.Millisecond * 100)
				}
			}

			return t
		}
	}
}

func (e *Exchange) UseSpotMarket(baseCcy string, quoteCcy string) common.SpotMarket {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)

	if e.excfg.TickerFromRest {
		e.tickerRestInstType["SPOT"] = 1
	}

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
			m.Init(e, *inst, inst.BaseCcy, inst.QuoteCcy, e.excfg.DepthFromTicker, e.excfg.TickerFromRest)
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
		mkt := e.UseSpotMarket(baseCcy, quoteCcy)
		if mkt == nil {
			return nil
		} else {
			skmt := mkt.(*SpotMarket)
			t.Init(e, orderTag(), skmt)
			e.spotTraders[instId] = t
			e.spotTradersSlice = append(e.spotTradersSlice, t)

			// 准备刷新其最大可用
			e.spotInstIdsForMaxAvail = append(e.spotInstIdsForMaxAvail, instId)

			// 等最大可用就绪
			for {
				if _, ok := e.getMaxAvailable(instId); ok {
					break
				} else {
					time.Sleep(time.Millisecond * 100)
				}
			}

			return t
		}
	}
}

// 获取金融接口
func (e *Exchange) GetFinance() common.Finance {
	if e.finance == nil {
		e.finance = &Finance{}
		e.finance.init()
	}

	return e.finance
}

// 获取全部合约仓位
func (e *Exchange) GetAllPositions() []common.Position {
	e.muPosition.Lock()
	defer e.muPosition.Unlock()
	positions := make([]common.Position, 0, len(e.ctPositions))
	for _, pi := range e.ctPositions {
		positions = append(positions, pi)
	}
	return positions
}

// 获取全部资产
func (e *Exchange) GetAllBalances() []common.Balance {
	balImpls := e.balanceMgr.GetAllBalances()
	bals := make([]common.Balance, 0, len(balImpls))
	for _, bi := range balImpls {
		bals = append(bals, bi)
	}
	return bals
}

// 使用FundingfeeObserver，则必须启用ticker_from_rest
func (e *Exchange) UseFundingFeeInfoObserver() common.FundingFeeObserver {
	if e.fundingFeeObserver == nil {
		if !e.excfg.TickerFromRest {
			logger.LogPanic(logPrefix, "usage of fundingfee-observer reqire ticker_from_rest")
		}

		// 将现货和对应合约加入ticker拉取
		e.tickerRestInstType["SPOT"] = 1
		e.tickerRestInstType["SWAP"] = 1

		e.fundingFeeObserver = new(FundingFeeObserver)
		e.fundingFeeObserver.init(e)
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
	return GetSpotKline(baseCcy, quoteCcy, t0, t1, intervalSec)
}

func (e *Exchange) GetFutureKline(symbol, contractType string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	return GetFutureKline(symbol, contractType, t0, t1, intervalSec)
}

func GetSpotKline(baseCcy, quoteCcy string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)
	return GetKline(instId, t0, t1, intervalSec)
}

func GetFutureKline(symbol, contractType string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	instId := CCyCttypeToInstId(symbol, contractType)
	return GetKline(instId, t0, t1, intervalSec)
}

func GetKline(instId string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	if kusRaw, ok := cachedok.GetKline(instId, t0, t1, intervalSec, nil); ok {
		kus := make([]common.KUnit, 0)
		for _, ku := range kusRaw {
			kus = append(kus, common.KUnit{
				Time:         ku.Time,
				OpenPrice:    ku.Open,
				ClosePrice:   ku.Close,
				HighestPrice: ku.High,
				LowestPrice:  ku.Low,
				VolumeUSD:    ku.VolumeUSD,
			})
		}
		return kus
	}
	return nil
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
		e.accountBal = r.Data[0] // 整个账户数据存储下来备用
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
				b.Refresh(rights, frozen, updateTime)
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
		symbol := InstId2Symbol(instId)
		contractType := InstId2ContractType(instId)
		e.ctPositions[instId] = common.NewPositionImpl(instId, symbol, contractType)
	}
	p = e.ctPositions[instId]

	return p
}

func (e *Exchange) updatePosition(wg *sync.WaitGroup) {
	index := 0
	timeout := time.NewTicker(time.Second * 20) // 20秒收不到仓位，重新订阅
	tRest := time.NewTicker(time.Minute)        // 1分钟固定rest更新
	posOk := false

	// 订阅websocket
	s := e.ws.SubscribePosition(func(resp interface{}) {
		r := resp.(okexv5api.PositionWsResp)

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

			e.processPositionUnits(r.Data)
		}()

		if !posOk {
			posOk = true
			wg.Done()
		}
	})

	for {
		select {
		case <-timeout.C:
			logger.LogInfo(logPrefix, "position time out, re-subscribe it")
			s.Reset()
		case <-tRest.C:
			// 目前只取永续合约的仓位
			if resp, err := okexv5api.GetPositions("SWAP", ""); err == nil {
				if resp.Code == "0" {
					e.processPositionUnits(resp.Data)
				} else {
					logger.LogImportant(logPrefix, "get position from rest failed: code=%v, msg=%v", resp.Code, resp.Msg)
				}
			} else {
				logger.LogImportant(logPrefix, "get positions from rest failed: %s", err.Error())
			}
		}

	}
}

func (e *Exchange) processPositionUnits(posUnits []okexv5api.PositionUnit) {
	for i := 0; i < len(posUnits); i++ {
		d := posUnits[i]
		if d.MgnMode == "cross" && (d.InstType == "SWAP" || d.InstType == "FUTURES") {
			position := e.findPosition(d.InstId)
			size := util.String2DecimalPanic(d.Pos)
			avgPx := util.String2DecimalPanicUnless(d.AvgPx, "")
			time := util.ConvetUnix13StrToTimePanic(d.UTime)
			if d.PosSide == "net" {
				// 仅支持双向模式
				if size.IsPositive() {
					position.RefreshLong(size, avgPx, time)
					position.RefreshShort(decimal.Zero, decimal.Zero, time)
				} else {
					position.RefreshShort(size.Neg(), avgPx, time)
					position.RefreshLong(decimal.Zero, decimal.Zero, time)
				}
			} else if d.PosSide == "long" {
				position.RefreshLong(size, avgPx, time)
			} else if d.PosSide == "short" {
				position.RefreshShort(size, avgPx, time)
			}
		}
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

func (e *Exchange) updateMaxAvalilable() {
	// 每3秒刷新一次
	for {
		e.updateMaxAvalilableOfInstIds(e.spotInstIdsForMaxAvail)
		e.updateMaxAvalilableOfInstIds(e.contractInstIdsForMaxAvail)
		time.Sleep(time.Second * 3)
	}
}

func (e *Exchange) updateMaxAvalilableOfInstIds(instIds []string) bool {
	allOk := true
	for i := 0; i < len(instIds); i += 5 {
		instIdGroup := strings.Join(instIds[i:util.MinInt(i+5, len(instIds))], ",")
		if resp, err := okexv5api.GetMaxAvailableSize(instIdGroup, string(e.excfg.SpotTradeMode), false); err != nil {
			allOk = false
			logger.LogImportant(logPrefix, "get max available of %s failed: %s", instIdGroup, err.Error())
		} else {
			if resp.Code != "0" {
				allOk = false
				logger.LogImportant(logPrefix, "get max available of %s failed: %s(%s)", instIdGroup, resp.Msg, resp.Code)
			} else {
				e.muMaxAvailable.Lock()
				for _, masr := range resp.Data {
					e.maxAvailable[masr.InstId] = masr
				}
				e.muMaxAvailable.Unlock()
			}
		}
		time.Sleep(time.Millisecond * 50)
	}
	return allOk
}

// #endregion

// #region 其他
// 是否为单币种保证金模式。
// 单币种保证金模式跟跨币种、组合保证金模式的处理方式有很多不同
func (e *Exchange) isSingleMarginMode() bool {
	return e.singleMargin
}

func (e *Exchange) initInstruments() {
	logger.LogImportant(logPrefix, "fetching instruments...SPOT")
	e.processInstruments("SPOT")
	logger.LogImportant(logPrefix, "fetching instruments...SWAP")
	e.processInstruments("SWAP")
	logger.LogImportant(logPrefix, "fetching instruments...FUTURES")
	e.processInstruments("FUTURES")
}

func (e *Exchange) updateLiquidationOrders() {
	if !e.excfg.SubscribeLiquidationOrders {
		return
	}

	s := e.ws.SubscribeLiquidationOrders("SWAP", func(i interface{}) {
		resp := i.(okexv5api.LiquidationOrderWsResp)
		for _, v := range resp.Data {
			if m, ok := e.futureMarkets[v.InstId]; ok {
				for _, lod := range v.Details {
					m.onLiquidationOrder(lod.BrokenPrice, lod.Size, util.ValueIf(lod.Side == "buy", common.OrderDir_Buy, common.OrderDir_Sell))
				}
			}
		}
	})

	// 固定10分钟重新订阅一次
	tResub := time.NewTicker(time.Minute * 10)

	for {
		select {
		case <-tResub.C:
			s.Reset()
		}
	}
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
				ins.CtType = common.ContractType_UsdSwap
				ins.IsUsdtContract = false
			} else if strings.Contains(instId, "USDT-SWAP") {
				// usdt合约
				ins.CtSymbol = ins.CtValCcy
				ins.CtType = common.ContractType_UsdtSwap
				ins.IsUsdtContract = true
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
		okxCfg := resp.Data[0]
		if e.excfg.AccLevel != okexv5api.AccLevel(okxCfg.AccLevel) {
			logger.LogPanic(logPrefix, "check account config failed：账户交易模式不匹配。需要：%s, 实际：%s。请修改账户配置", e.excfg.AccLevel, okxCfg.AccLevel)
		} else {
			if okxCfg.AccLevel == okexv5api.AccLevel_SingleCcy {
				e.singleMargin = true
				logger.LogImportant(logPrefix, "Ocurrent account mode: single-currency margin")
			} else if okxCfg.AccLevel == okexv5api.AccLevel_MultiCcy {
				e.singleMargin = false
				logger.LogImportant(logPrefix, "current account mode: multi-currency margin")
			} else if okxCfg.AccLevel == okexv5api.AccLevel_Portfolio {
				e.singleMargin = false
				logger.LogImportant(logPrefix, "current account mode: portfolio margin")
			} else {
				// 非保证金模式不支持
				logger.LogPanic(logPrefix, "unsupported account mode")
			}
		}

		if okxCfg.PosMode == "long_short_mode" {
			e.excfg.PositionMode = okexv5api.PositonMode_LS
			logger.LogImportant(logPrefix, "current position mode: long_short_mode")
		} else if okxCfg.PosMode == "net_mode" {
			e.excfg.PositionMode = okexv5api.PositionMode_Net
			logger.LogImportant(logPrefix, "current position mode: net_mode")
		} else {
			logger.LogPanic(logPrefix, "check account config failed：仓位模式不支持，请修改账户配置")
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

func (e *Exchange) registerTickerCallbackOfInstType(instType string, fn func(tks []okexv5api.TickerResp)) {
	e.muTickerCallback.Lock()
	defer e.muTickerCallback.Unlock()

	if cbs, ok := e.tickerCallbacksOfInstType[instType]; ok {
		cbs = append(cbs, fn)
		e.tickerCallbacksOfInstType[instType] = cbs
	} else {
		cbs := []func(tks []okexv5api.TickerResp){fn}
		e.tickerCallbacksOfInstType[instType] = cbs
	}
}

func (e *Exchange) updateTickersByRest() {
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		<-ticker.C
		for instType := range e.tickerRestInstType {
			if resp, err := okexv5api.GetTickers(instType); err == nil {
				for _, tk := range resp.Data {
					// 回调
					e.muTickerCallback.Lock()
					var cbs []func(tk okexv5api.TickerResp)
					if v, ok := e.tickerCallbacks[tk.InstId]; ok {
						cbs = v
					}
					e.muTickerCallback.Unlock()

					for _, fn := range cbs {
						fn(tk)
					}
				}

				// 回调
				if cbs, ok := e.tickerCallbacksOfInstType[instType]; ok {
					for _, fn := range cbs {
						fn(resp.Data)
					}
				}
				logger.LogInfo(logPrefix, "get %d tickers", len(resp.Data)) // debug
			} else {
				logger.LogImportant(logPrefix, "get ticker by instType failed, err=%s", err.Error())
			}
		}
	}
}

func (e *Exchange) getMaxAvailable(instId string) (okexv5api.MaxAvailableSizeResp, bool) {
	// usdt合约只查询一次，统一按btc来
	if strings.Contains(instId, "USDT-SWAP") {
		instId = "BTC-USDT-SWAP"
	}

	v, ok := e.maxAvailable[instId]
	return v, ok
}

// #endregion
