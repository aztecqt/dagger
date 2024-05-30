/*
- @Author: aztec
- @Date: 2024-03-08
- @Description: 现货行情，如股票/ETF等
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/

package ibkrtws

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/ibkr/twsapi"
	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/shopspring/decimal"
)

type SpotMarket struct {
	ex             *Exchange
	c              *twsapi.Client
	inst           *common.Instruments
	contract       *twsmodel.Contract
	contractConfig *ContractConfig
	needResub      bool
	latestPrice    decimal.Decimal
	orderBook      *common.Orderbook
	askPrice       decimal.Decimal
	askSize        decimal.Decimal
	bidPrice       decimal.Decimal
	bidSize        decimal.Decimal
	baseCcy        string
	quoteCcy       string

	marketDataReqId      int
	msgHandlerRegisterId int
	onConnectRegisterId  int

	priceOk                  bool
	depthLastValidTime       time.Time
	resubForInvalidDepthTime time.Time

	// 深度变化回调
	depthObserversSet *hashset.Set
	depthObservers    []interface{}

	// 交易时间配置，从exchange获取
	tradingTimes       common.TradingTimes
	instrumentsVersion int
}

func (m *SpotMarket) init(ex *Exchange, c *twsapi.Client, inst *common.Instruments, contract *twsmodel.Contract, contractConfig *ContractConfig) {
	m.ex = ex
	m.c = c
	m.inst = inst
	m.contract = contract
	m.contractConfig = contractConfig
	m.orderBook = common.NewOrderBook()
	m.priceOk = false
	m.depthObserversSet = hashset.New()
	m.baseCcy, m.quoteCcy = InstIdToSpotType(inst.Id)

	m.msgHandlerRegisterId = m.c.RegisterMessageHandler(m.onTwsMessage)
	m.onConnectRegisterId = m.c.RegisterOnConnectCallback(func() {
		// 重新连接后，需要重新订阅市场行情
		logInfo(logPrefix, "tws connected, need resubscribe market data")
		m.needResub = true
	})

	m.initMarketData()
	m.subscribeMarketData()

	// 持续更新交易时间
	go func() {
		for {
			if m.instrumentsVersion != m.ex.instrumentsVersion {
				if v, ok := m.ex.findTradingTime(m.inst.Id); ok {
					m.tradingTimes = v
					m.instrumentsVersion = m.ex.instrumentsVersion
				} else {
					logError(logPrefix, "find trading time of %s failed", m.inst.Id)
				}
			}

			time.Sleep(time.Second * 10)
		}
	}()
}

// 初始化latestPrice
func (m *SpotMarket) initMarketData() {
	// 实时市场数据在程序启动时，可能由于停盘而无法获取任何数据。所以需要通过历史数据，拉一下上次开盘最后一刻的收盘价格/盘口价
	// 最新价格
	wathToShow := util.ValueIf(m.contract.SecType == "CRYPTO", "AGGTRADES", "TRADES")
	if resp := m.c.ReqHistoricalData(*m.contract, time.Now(), "3 D", "1 day", wathToShow, 1, false); resp.RespCode == twsapi.RespCode_Ok {
		if resp.Err == nil {
			fmt.Println(resp.HistoricalData.Bars)
			if len(resp.HistoricalData.Bars) > 0 {
				bars := resp.HistoricalData.Bars
				m.latestPrice = bars[len(bars)-1].Close // 不可设置priceOk
			} else {
				logError(logPrefix, "get history data failed, no data")
			}

		} else {
			logError(logPrefix, "get history data failed, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
		}
	} else {
		logError(logPrefix, "get history data failed")
	}

	// Ask
	if resp := m.c.ReqHistoricalData(*m.contract, time.Now(), "3 D", "1 day", "ASK", 1, false); resp.RespCode == twsapi.RespCode_Ok {
		if resp.Err == nil {
			fmt.Println(resp.HistoricalData.Bars)
			if len(resp.HistoricalData.Bars) > 0 {
				bars := resp.HistoricalData.Bars
				m.askPrice = bars[len(bars)-1].Close
				m.askSize = util.DecimalOne
			} else {
				logError(logPrefix, "get history data failed, no data")
			}

		} else {
			logError(logPrefix, "get history data failed, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
		}
	} else {
		logError(logPrefix, "get history data failed")
	}

	// Bid
	if resp := m.c.ReqHistoricalData(*m.contract, time.Now(), "3 D", "1 day", "BID", 1, false); resp.RespCode == twsapi.RespCode_Ok {
		if resp.Err == nil {
			fmt.Println(resp.HistoricalData.Bars)
			if len(resp.HistoricalData.Bars) > 0 {
				bars := resp.HistoricalData.Bars
				m.bidPrice = bars[len(bars)-1].Close
				m.bidSize = util.DecimalOne
			} else {
				logError(logPrefix, "get history data failed, no data")
			}

		} else {
			logError(logPrefix, "get history data failed, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
		}
	} else {
		logError(logPrefix, "get history data failed")
	}

	// rebuild（仅做rebuild，不设置depthValidTime，因为这可能不是最新的盘口数据）
	m.orderBook.Rebuild([]decimal.Decimal{m.askPrice, m.askSize}, []decimal.Decimal{m.bidPrice, m.bidSize})
}

// 执行订阅
func (m *SpotMarket) subscribeMarketData() {
	m.needResub = true

	go func() {
		for {
			if m.needResub {
				// 订阅实时市场数据
				if id, resp := m.c.ReqMarketData(*m.contract, "", false, false); resp.RespCode == twsapi.RespCode_Ok {
					if resp.Err == nil {
						m.marketDataReqId = id
						m.needResub = false
						logInfo(logPrefix, "subscribe market data success")
					} else {
						logError(logPrefix, "subscribe market data error, code=%d, msg=%s", resp.Err.ErrorCode, resp.Err.ErrorMessage)
						time.Sleep(time.Minute)
					}
				} else {
					logError(logPrefix, "subscribe market data failed!")
					if resp.RespCode != twsapi.RespCode_TimeOut {
						time.Sleep(time.Minute)
					}
				}
			} else {
				// 盘口无效超过1分钟，尝试重新订阅。最小间隔1分钟。
				if m.marketIsOpen() {
					depthInvalidSec := time.Since(m.depthLastValidTime).Seconds()
					if depthInvalidSec > 60 && time.Since(m.resubForInvalidDepthTime).Seconds() > 60 {
						m.needResub = true
						m.resubForInvalidDepthTime = time.Now()
						logInfo(logPrefix, "orderbook invalid for %.0f seconds, need resub", depthInvalidSec)
					}
				}
			}
			time.Sleep(time.Second)
		}
	}()
}

func (m *SpotMarket) onTwsMessage(msg twsapi.Message) {
	if msg.MsgId == twsapi.InCommingMessage_TickPrice {
		tpmsg := msg.Msg.(*twsapi.TickPriceMsg)
		if tpmsg.RequestId == m.marketDataReqId {
			orderBookNeedRebuild := false
			if tpmsg.TickType == twsmodel.TickType_Ask {
				if tpmsg.Price.IsPositive() && tpmsg.Size.IsPositive() {
					m.askPrice = tpmsg.Price
					m.askSize = tpmsg.Size
					orderBookNeedRebuild = true
				}
			} else if tpmsg.TickType == twsmodel.TickType_Bid {
				if tpmsg.Price.IsPositive() && tpmsg.Size.IsPositive() {
					m.bidPrice = tpmsg.Price
					m.bidSize = tpmsg.Size
					orderBookNeedRebuild = true
				}
			} else if tpmsg.TickType == twsmodel.TickType_Last {
				if tpmsg.Price.IsPositive() {
					m.latestPrice = tpmsg.Price
					if !m.priceOk {
						m.priceOk = true
					}
				}
			}

			if orderBookNeedRebuild {
				m.rebuildOrderBook()
			}
		}
	} else if msg.MsgId == twsapi.InCommingMessage_TickSize {
		tsmsg := msg.Msg.(*twsapi.TickSizeMsg)
		if tsmsg.RequestId == m.marketDataReqId {
			orderBookNeedRebuild := false
			if tsmsg.TickType == twsmodel.TickType_AskSize {
				if tsmsg.Size.IsPositive() {
					m.askSize = tsmsg.Size
					orderBookNeedRebuild = true
				}
			} else if tsmsg.TickType == twsmodel.TickType_BidSize {
				if tsmsg.Size.IsPositive() {
					m.bidSize = tsmsg.Size
					orderBookNeedRebuild = true
				}
			}

			if orderBookNeedRebuild {
				m.rebuildOrderBook()
			}
		}
	}
}

func (m *SpotMarket) rebuildOrderBook() {
	m.orderBook.Rebuild([]decimal.Decimal{m.askPrice, m.askSize}, []decimal.Decimal{m.bidPrice, m.bidSize})
	if m.askPrice.IsPositive() && m.askSize.IsPositive() && m.bidPrice.IsPositive() && m.bidSize.IsPositive() {
		m.depthLastValidTime = time.Now()
	}
}

func (m *SpotMarket) depthOk() bool {
	return time.Now().Sub(m.depthLastValidTime).Seconds() < 30
}

func (m *SpotMarket) marketIsOpen() bool {
	// 当前市场是不是在交易状态
	return m.tradingTimes != nil && m.tradingTimes.Contains(time.Now())
}

// #region 实现common.SpotMarket接口
func (m *SpotMarket) AddDepthObserver(o common.DepthObserver) {
	m.depthObserversSet.Add(o)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *SpotMarket) RemoveDepthObserver(o common.DepthObserver) {
	m.depthObserversSet.Remove(o)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *SpotMarket) Type() string {
	return m.inst.Id
}

func (m *SpotMarket) TradingTime() common.TradingTimes {
	return m.tradingTimes
}

func (m *SpotMarket) Ready() bool {
	return m.c.IsConnectOk() && m.priceOk && m.depthOk() && m.marketIsOpen()
}

func (m *SpotMarket) UnreadyReason() string {
	if !m.c.IsConnectOk() {
		return "connect lost"
	} else if !m.marketIsOpen() {
		return "not in trading time"
	} else if !m.priceOk {
		return "latest price not ready"
	} else if !m.depthOk() {
		return "depth not ready"
	} else {
		return ""
	}
}

func (m *SpotMarket) Uninit() {
	if m.msgHandlerRegisterId > 0 {
		m.c.UnregisterMessageHandler(m.msgHandlerRegisterId)
	}

	if m.onConnectRegisterId > 0 {
		m.c.UnregisterOnConnectCallback(m.onConnectRegisterId)
	}
}

func (m *SpotMarket) LatestPrice() decimal.Decimal {
	return m.latestPrice
}

func (m *SpotMarket) OrderBook() *common.Orderbook {
	return m.orderBook
}

func (m *SpotMarket) AlignPriceNumber(price decimal.Decimal) decimal.Decimal {
	return m.ex.instrumentMgr.AlignPriceNumber(m.inst.Id, price)
}

func (m *SpotMarket) AlignPrice(price decimal.Decimal, dir common.OrderDir, makeOnly bool) decimal.Decimal {
	if price.IsZero() {
		return price
	} else {
		return m.ex.instrumentMgr.AlignPrice(m.inst.Id, price, dir, makeOnly, m.orderBook.Buy1Price(), m.orderBook.Sell1Price())
	}
}

func (m *SpotMarket) AlignSize(size decimal.Decimal) decimal.Decimal {
	if size.IsZero() {
		return size
	} else {
		return m.ex.instrumentMgr.AlignSize(m.inst.Id, size)
	}
}

func (m *SpotMarket) MinSize() decimal.Decimal {
	return m.ex.instrumentMgr.MinSize(m.inst.Id, m.orderBook.Buy1Price())
}

func (m *SpotMarket) BaseCurrency() string {
	return m.baseCcy
}

func (m *SpotMarket) QuoteCurrency() string {
	return m.quoteCcy
}

func (m *SpotMarket) String() string {
	bb := bytes.Buffer{}
	bb.WriteString(fmt.Sprintf("\nspot market: %s\n", m.inst.Id))
	bb.WriteString(fmt.Sprintf("price: %s\n", m.latestPrice.String()))
	bb.WriteString("depth:\n")
	bb.WriteString(m.OrderBook().String(1))
	return bb.String()
}

// #endregion
