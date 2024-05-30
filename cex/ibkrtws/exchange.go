/*
- @Author: aztec
- @Date: 2024-03-08 09:28:13
- @Description: 基于tws的ibrk交易所定义。目前只支持现货。InstId格式：IBIT-USD
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

	"github.com/aztecqt/dagger/api/ibkr/twsapi"
	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

const logPrefix = "IBRK-TWS"
const exchangeName = "ibkrtws"

type Exchange struct {
	c            *twsapi.Client
	accountName  string
	msgHandlerId int

	exAccountDataReady bool
	exInstrumentReady  bool
	exExited           bool

	// 交易所配置
	excfg ExchangeConfig

	// 所有行情和交易器
	spotMarkets      map[string]*SpotMarket
	spotTraders      map[string]*SpotTrader
	spotMarketsSlice []common.SpotMarket
	spotTradersSlice []common.SpotTrader

	// 交易品种
	muInstruments         sync.Mutex
	instrumentMgr         *common.InstrumentMgr
	instId2Contract       map[string]*twsmodel.Contract
	instId2ContractConfig map[string]*ContractConfig
	instId2TradingTimes   map[string]common.TradingTimes
	instrumentsVersion    int

	// 用户权益，主要针对现货，系统会自主计算币种余额，并跟交易所对齐
	// ccy格式为小写
	balanceMgr *common.BalanceMgr

	// 每个币种的冻结余额，由订单计算而来
	// 由于tws不提供已冻结资产的数据刷新，因此，这项数据需要自己计算
	muFrozenBalance      sync.Mutex
	freezedBalance       map[string]decimal.Decimal         // ccy->frozen
	freezedBalanceDetail map[int]map[string]decimal.Decimal // orderId->ccy->frozen

	// 订单更新推送
	// clientOrderId->fn
	orderStatusHandlerMu sync.Mutex
	orderStatusHandler   map[int]func(*twsapi.OrderStatusMsg, *twsapi.OpenOrdersMsg)
}

func (e *Exchange) Init(excfg ExchangeConfig, logInfo, logDebug, logError fnLog) {
	excfg.parse()
	e.excfg = excfg
	e.spotMarkets = make(map[string]*SpotMarket)
	e.spotTraders = make(map[string]*SpotTrader)
	e.spotMarketsSlice = make([]common.SpotMarket, 0)
	e.spotTradersSlice = make([]common.SpotTrader, 0)
	e.instrumentMgr = common.NewInstrumentMgr(logPrefix)
	e.instId2Contract = map[string]*twsmodel.Contract{}
	e.instId2ContractConfig = map[string]*ContractConfig{}
	e.instId2TradingTimes = make(map[string]common.TradingTimes)
	e.freezedBalance = make(map[string]decimal.Decimal)
	e.freezedBalanceDetail = make(map[int]map[string]decimal.Decimal)
	e.orderStatusHandler = make(map[int]func(*twsapi.OrderStatusMsg, *twsapi.OpenOrdersMsg))
	e.balanceMgr = common.NewBalanceMgr(true)

	// 资产最大容许偏移量设置
	if excfg.MaxPitch != nil {
		for ccy, maxPitch := range excfg.MaxPitch {
			e.balanceMgr.FindBalance(ccy).SetMaxPitchAllowed(maxPitch)
		}
	}

	// 初始化api
	if logInfo == nil {
		logInfo = func(msg string) { logger.LogInfo("twsapi", msg) }
	}

	if logDebug == nil {
		logDebug = func(msg string) { logger.LogDebug("twsapi", msg) }
	}

	if logError == nil {
		logError = func(msg string) { logger.LogImportant("twsapi", msg) }
	}

	logInfoFn = logInfo
	logDebugFn = logDebug
	logErrorFn = logError

	twsapi.LogMessage = false
	twsapi.Init(twsapi.FnLog(logInfo), twsapi.FnLog(logDebug), twsapi.FnLog(logError))
	e.c = twsapi.NewClient(excfg.Addr, excfg.Port)
	e.msgHandlerId = e.c.RegisterMessageHandler(e.onMessage)

	// 连接成功后需要订阅的频道
	// 断线重连后，也会执行相同的逻辑
	e.c.RegisterOnConnectCallback(func() {
		// 订阅账户更新
		if len(e.c.Accounts()) > 0 {
			e.accountName = e.c.Accounts()[0]
			e.accountName = e.c.Accounts()[0]
		} else {
			logError("no account data")
		}

		e.c.ReqAccountUpdates(e.accountName)
	})

	e.c.Connect()

	// 加载instruments
	e.loadInstruments()

	go func() {
		lastTime := time.Now()
		for {
			now := time.Now()
			if now.Hour() == 8 && lastTime.Hour() != 8 {
				// 之后每日8点更新一次instruments
				e.loadInstruments()
			}
			lastTime = now
			time.Sleep(time.Minute)
		}
	}()
}

func (e *Exchange) ReconnectApi(reason string) {
	e.c.Reconnect(reason)
}

func (e *Exchange) ready() bool {
	return e.exAccountDataReady && e.exInstrumentReady && !e.exExited
}

func (e *Exchange) unreadyReason() string {
	if !e.exAccountDataReady {
		return "ex account data not ready"
	} else if !e.exInstrumentReady {
		return "ex instruments not ready"
	} else {
		return ""
	}
}

// 加载所有Contract，并转换为Instrument数据
func (e *Exchange) loadInstruments() {
	i := 0
	for i < len(e.excfg.Contracts) {
		if !e.loadInstrumentSingle(e.excfg.Contracts[i]) {
			time.Sleep(time.Second)
		} else {
			i++
		}
	}
	e.exInstrumentReady = true
}

func (e *Exchange) loadInstrumentSingle(ct ContractConfig) bool {
	// 获取ContractDetail
	if respCtDetail := e.c.ReqContractDetails(ct.toTwsContract()); respCtDetail != nil {
		if respCtDetail.RespCode != twsapi.RespCode_Ok {
			logError(logPrefix, "req contract detail failed, code is not ok")
			return false
		} else if respCtDetail.Err != nil {
			logError(logPrefix, "req contract detail failed, code=%d, msg=%s", respCtDetail.Err.ErrorCode, respCtDetail.Err.ErrorMessage)
			return false
		} else {
			if len(respCtDetail.MatchedDetails) == 0 {
				logError(logPrefix, "req contract detail failed, no data")
				return false
			} else if len(respCtDetail.MatchedDetails) > 1 {
				logError(logPrefix, "contract seed is ambiguous, need more accurate definition")
				return false
			} else {
				// 找出对应的ruleId
				d := respCtDetail.MatchedDetails[0].Detail
				ruleId := d.GetRuleOfExchange(ct.Exchange)

				// 获取marketRule
				var rule twsapi.MarketRuleMsg
				marketRules := map[int]twsapi.MarketRuleMsg{}
				if v, ok := marketRules[ruleId]; ok {
					rule = v
				} else {
					if respRule := e.c.ReqMarketRule(ruleId); respRule != nil {
						rule = *respRule.MarketRule
						marketRules[ruleId] = rule
					} else {
						logError(logPrefix, "req market rule failed")
						return false
					}
				}

				// 构建Instrument
				inst := common.Instruments{
					Id:       SpotTypeToInstId(d.Contract.Symbol, d.Contract.Currency),
					BaseCcy:  d.Contract.Symbol,
					QuoteCcy: d.Contract.Currency,
					TickSize: d.MinTick,
					LotSize:  d.SizeIncrement,
					MinSize:  d.MinSize,
					MinValue: decimal.Zero,
				}

				// 解析交易时间
				tradingTimes := common.TradingTimes{}
				tradingTimesFromString(&tradingTimes, d.LiquidHours)

				e.muInstruments.Lock()
				e.instrumentMgr.Set(inst.Id, &inst)
				e.instId2Contract[inst.Id] = &d.Contract
				e.instId2ContractConfig[inst.Id] = &ct
				e.instId2TradingTimes[inst.Id] = tradingTimes
				e.instrumentsVersion++
				e.muInstruments.Unlock()
				logInfo(logPrefix, "loaded instrument")
			}
		}
	}

	return true
}

// 查询交易时间配置
func (e *Exchange) findTradingTime(instId string) (common.TradingTimes, bool) {
	e.muInstruments.Lock()
	defer e.muInstruments.Unlock()
	if v, ok := e.instId2TradingTimes[instId]; ok {
		return v, true
	} else {
		return nil, false
	}
}

// 订单更新注册
func (e *Exchange) registerOrderStatusHandler(clientOrderId int, fn func(*twsapi.OrderStatusMsg, *twsapi.OpenOrdersMsg)) {
	e.orderStatusHandlerMu.Lock()
	defer e.orderStatusHandlerMu.Unlock()
	e.orderStatusHandler[clientOrderId] = fn
}

func (e *Exchange) unregisterOrderStatusHandler(clientOrderId int) {
	e.orderStatusHandlerMu.Lock()
	defer e.orderStatusHandlerMu.Unlock()
	delete(e.orderStatusHandler, clientOrderId)
}

// 冻结资产管理
func (e *Exchange) setFrozenBalance(orderId int, ccy string, frozenBal decimal.Decimal) {
	e.muFrozenBalance.Lock()
	defer e.muFrozenBalance.Unlock()

	if _, ok := e.freezedBalanceDetail[orderId]; !ok {
		e.freezedBalanceDetail[orderId] = map[string]decimal.Decimal{ccy: frozenBal}
	} else {
		m := e.freezedBalanceDetail[orderId]
		m[ccy] = frozenBal
		e.freezedBalanceDetail[orderId] = m
	}
}

func (e *Exchange) clearFrozenBalance(orderId int) {
	e.muFrozenBalance.Lock()
	defer e.muFrozenBalance.Unlock()

	delete(e.freezedBalanceDetail, orderId)
}

func (e *Exchange) getFrozenBalance(ccy string) decimal.Decimal {
	e.muFrozenBalance.Lock()
	defer e.muFrozenBalance.Unlock()

	totalFrozen := decimal.Zero
	for _, m := range e.freezedBalanceDetail {
		if f, ok := m[ccy]; ok {
			totalFrozen = totalFrozen.Add(f)
		}
	}
	return totalFrozen
}

// #region tws消息处理
func (e *Exchange) onMessage(m twsapi.Message) {
	switch m.MsgId {
	case twsapi.InCommingMessage_AccountValue:
		e.onMsg_AccountValue(m.Msg.(*twsapi.AccountValueMsg))
	case twsapi.InCommingMessage_PortfolioValue:
		e.onMsg_PortfolioValue(m.Msg.(*twsapi.PortfolioValueMsg))
	case twsapi.InCommingMessage_OrderStatus:
		e.onMsg_OrderStatus(m.Msg.(*twsapi.OrderStatusMsg))
	case twsapi.InCommingMessage_OpenOrder:
		e.onMsg_OpenOrderMsg(m.Msg.(*twsapi.OpenOrdersMsg))
	case twsapi.InCommingMessage_Error:
		e.onMsg_ErrorMsg(m.Msg.(*twsapi.ErrorMsg))
	case twsapi.InCommingMessage_AccountDownloadEnd:
		e.exAccountDataReady = true
	}
}

func (e *Exchange) onMsg_AccountValue(msg *twsapi.AccountValueMsg) {
	// 仅处理Key为TotalCashBalance的、且在配置中指定过的Currency
	if msg.AccountName == e.accountName && msg.Key == "TotalCashBalance" && slices.Contains(e.excfg.Currencys, msg.Currency) {
		ccy := strings.ToLower(msg.Currency)
		pitch := e.balanceMgr.RefreshBalance(
			ccy,
			util.String2DecimalPanic(msg.Value),
			decimal.Zero, // 目前似乎并不能知道被冻结的资产余额
			time.Now())

		if !pitch.IsZero() {
			logError(logPrefix, "%s balance has pitch: %v", ccy, pitch)
		}
	}
}

func (e *Exchange) onMsg_PortfolioValue(msg *twsapi.PortfolioValueMsg) {
	// 仅处理配置中指定过的、匹配Contract.Symbol的项目
	if msg.AccountName == e.accountName && slices.Contains(e.excfg.Symbols, msg.Contract.Symbol) {
		ccy := strings.ToLower(msg.Contract.Symbol)
		pitch := e.balanceMgr.RefreshBalance(
			ccy,
			msg.Position,
			decimal.Zero,
			time.Now())

		if !pitch.IsZero() {
			logError(logPrefix, "%s balance has pitch: %v", ccy, pitch)
		}
	}
}

func (e *Exchange) onMsg_OrderStatus(msg *twsapi.OrderStatusMsg) {
	e.orderStatusHandlerMu.Lock()
	defer e.orderStatusHandlerMu.Unlock()

	if fn, ok := e.orderStatusHandler[msg.OrderId]; ok {
		fn(msg, nil)
	}
}

func (e *Exchange) onMsg_OpenOrderMsg(msg *twsapi.OpenOrdersMsg) {
	e.orderStatusHandlerMu.Lock()
	defer e.orderStatusHandlerMu.Unlock()

	if fn, ok := e.orderStatusHandler[msg.Order.OrderId]; ok {
		fn(nil, msg)
	}
}

// 对所有不认识的错误，都进行报警
func (e *Exchange) onMsg_ErrorMsg(msg *twsapi.ErrorMsg) {
	switch msg.ErrorCode {
	case 2104: // connection is OK
		return
	case 2106: // connection is OK
		return
	case 2158: // connection is OK
		return
	case 202: // Order Canceled - reason:
		return
	case 10148: // OrderId xxx that needs to be cancelled can not be cancelled, state: Cancelled
		return
	}

}

// #endregion

// #region 实现common.CEx接口
func (e *Exchange) Name() string {
	return exchangeName
}

func (e *Exchange) Instruments() []*common.Instruments {
	e.muInstruments.Lock()
	defer e.muInstruments.Unlock()
	return e.instrumentMgr.GetAll()
}

func (e *Exchange) GetSpotInstrument(baseCcy, quoteCcy string) *common.Instruments {
	e.muInstruments.Lock()
	defer e.muInstruments.Unlock()
	return e.instrumentMgr.Get(SpotTypeToInstId(baseCcy, quoteCcy))
}

func (e *Exchange) GetFutureInstrument(symbol, contractType string) *common.Instruments {
	return nil
}

func (e *Exchange) GetUniAccRisk() common.UniAccRisk {
	return common.UniAccRisk{Level: common.UniAccRiskLevel_Safe}
}

func (e *Exchange) FutureMarkets() []common.FutureMarket {
	return []common.FutureMarket{}
}

func (e *Exchange) FutureTraders() []common.FutureTrader {
	return []common.FutureTrader{}
}

func (e *Exchange) UseFutureMarket(symbol, contractType string) common.FutureMarket {
	return nil
}

func (e *Exchange) UseFutureTrader(symbol, contractType string, lever int) common.FutureTrader {
	return nil
}

func (e *Exchange) SpotMarkets() []common.SpotMarket {
	return e.spotMarketsSlice
}

func (e *Exchange) SpotTraders() []common.SpotTrader {
	return e.spotTradersSlice
}

func (e *Exchange) UseSpotMarket(baseCcy, quoteCcy string) common.SpotMarket {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)

	if m, ok := e.spotMarkets[instId]; ok {
		return m
	} else {
		e.muInstruments.Lock()
		inst := e.instrumentMgr.Get(instId)
		contract := e.instId2Contract[instId]
		cc := e.instId2ContractConfig[instId]
		e.muInstruments.Unlock()

		if inst == nil || contract == nil {
			logError(logPrefix, "unknown instId:%s", instId)
			return nil
		}

		m := new(SpotMarket)
		m.init(e, e.c, inst, contract, cc)
		e.spotMarkets[instId] = m
		e.spotMarketsSlice = append(e.spotMarketsSlice, m)
		return m
	}
}

func (e *Exchange) UseSpotTrader(baseCcy, quoteCcy string) common.SpotTrader {
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
			if cs, ok := e.excfg.findContractSeed(baseCcy, quoteCcy); ok {
				smkt := mkt.(*SpotMarket)
				t.Init(e, smkt, cs.Tif)
				e.spotTraders[instId] = t
				e.spotTradersSlice = append(e.spotTradersSlice, t)
				return t
			} else {
				logError(logPrefix, "spot trader not exist in exchange config, base=%s, quote=%s", baseCcy, quoteCcy)
				return nil
			}
		}
	}
}

func (e *Exchange) GetFinance() common.Finance {
	return nil
}

// 获取全部合约仓位
func (e *Exchange) GetAllPositions() []common.Position {
	return nil
}

// 获取全部资产
func (e *Exchange) GetAllBalances() []common.Balance {
	balImpls := e.balanceMgr.GetAllBalances()
	bals := make([]common.Balance, len(balImpls))
	for _, bi := range balImpls {
		bals = append(bals, bi)
	}
	return bals
}

func (e *Exchange) Exit() {
	e.exExited = false
	e.c.UnregisterMessageHandler(e.msgHandlerId)
}

func (e *Exchange) UseFundingFeeInfoObserver() common.FundingFeeObserver {
	return nil
}

func (e *Exchange) FundingFeeInfoObserver() common.FundingFeeObserver {
	return nil
}

func (e *Exchange) UseContractObserver(contractType string) common.ContractObserver {
	return nil
}

// 查询k线
// 注意，tws只能以当前时间作为t1
func (e *Exchange) GetSpotKline(baseCcy, quoteCcy string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	instId := SpotTypeToInstId(baseCcy, quoteCcy)
	barSize := ""
	switch intervalSec {
	case 1:
		barSize = "1 sec"
	case 5:
		barSize = "5 secs"
	case 15:
		barSize = "15 secs"
	case 30:
		barSize = "30 secs"
	case 60:
		barSize = "1 min"
	case 120:
		barSize = "2 mins"
	case 180:
		barSize = "3 mins"
	case 300:
		barSize = "5 mins"
	case 900:
		barSize = "15 mins"
	case 1800:
		barSize = "30 mins"
	case 3600:
		barSize = "1 hour"
	case 86400:
		barSize = "1 day"
	default:
		logError(logPrefix, "unsupported interval: %d", intervalSec)
	}

	dursec := (t1.Unix() - t0.Unix())
	durstr := ""
	if dursec <= 86400 {
		durstr = fmt.Sprintf("%d S", dursec)
	} else {
		durstr = fmt.Sprintf("%d D", dursec/86400)
	}

	if cont, ok := e.instId2Contract[instId]; ok {
		resp := e.c.ReqHistoricalData(*cont, t1, durstr, barSize, "MIDPOINT", 1 /*useRTH*/, false)
		if resp != nil && resp.RespCode == twsapi.RespCode_Ok && resp.Err == nil {
			kus := []common.KUnit{}
			for _, bar := range resp.HistoricalData.Bars {
				if bar.Time.UnixMilli() >= t0.UnixMilli() && bar.Time.UnixMilli() <= t1.UnixMilli() {
					kus = append(
						kus,
						common.KUnit{
							Time:         bar.Time.Local(),
							OpenPrice:    bar.Open,
							ClosePrice:   bar.Close,
							HighestPrice: bar.High,
							LowestPrice:  bar.Low,
							VolumeUSD:    bar.Volume})
				}
			}
			return kus
		}
	} else {
		logError(logPrefix, "can't find contract for %s", instId)
	}

	return nil
}

func (e *Exchange) GetFutureKline(symbol, contractType string, t0, t1 time.Time, intervalSec int) []common.KUnit {
	return nil
}

// 查询历史成交记录
func (e *Exchange) GetSpotDealHistory(baseCcy, quoteCcy string, t0, t1 time.Time) []common.DealHistory {
	return nil
}

func (e *Exchange) GetFutureDealHistory(symbol, contractType string, t0, t1 time.Time) []common.DealHistory {
	return nil
}

// #endregion
