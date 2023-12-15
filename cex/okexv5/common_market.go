/*
 * @Author: aztec
 * @Date: 2022-04-19 11:06:38
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-10-03 12:15:39
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\okexv5\common_market.go
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"hash/crc32"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/shopspring/decimal"
)

type CommonMarket struct {
	ex              *Exchange
	ws              *okexv5api.WsClient
	instId          string
	inst            common.Instruments
	latestPrice     decimal.Decimal
	orderBook       *common.Orderbook
	depthFromTicker bool
	tickerFromRest  bool

	priceOK bool
	depthOK bool

	// 深度变化回调。策略的主要驱动之一
	depthObserversSet *hashset.Set
	depthObservers    []interface{}

	subscribing bool
}

func (m *CommonMarket) Init(ex *Exchange, instID string, depthFromTicker, tickerFromRest bool) {
	m.ex = ex
	m.ws = ex.ws
	m.instId = instID
	m.inst = *ex.instrumentMgr.Get(instID)
	m.depthFromTicker = depthFromTicker
	m.tickerFromRest = tickerFromRest
	m.orderBook = new(common.Orderbook)
	m.orderBook.Init()
	m.priceOK = false
	m.depthOK = false

	m.depthObserversSet = hashset.New()
	m.depthObservers = nil
	m.subscribing = false
}

func (m *CommonMarket) AddDepthObserver(obs common.DepthObserver) {
	m.depthObserversSet.Add(obs)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *CommonMarket) RemoveDepthObserver(obs common.DepthObserver) {
	m.depthObserversSet.Remove(obs)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *CommonMarket) subscribe(instID string) {
	m.subscribing = true

	// 订阅ticker
	go func() {
		if m.tickerFromRest {
			// 从统一的rest获取。5秒拉取不到，则超时
			timeOut := time.NewTicker(time.Second * 5)
			m.ex.registerTickerCallback(instID, func(t okexv5api.TickerResp) {
				m.onTickerResp(t)
				timeOut.Reset(time.Second * 5)
			})

			for {
				select {
				case <-timeOut.C:
					m.depthOK = false
				}
			}
		} else {
			// 从ws获取
			//（ticker服务器不保证推送间隔。30秒触发rest调用，60秒触发重新订阅）
			timeoutReSub := time.NewTicker(time.Second * 60)
			timeoutREST := time.NewTicker(time.Second * 30)
			updateTicker := time.NewTicker(time.Second)
			s := m.ws.SubscribeTicker(instID, func(resp interface{}) {
				ticker := resp.(okexv5api.TickerWsResp).Data[0]
				m.onTickerResp(ticker)
				timeoutReSub.Reset(time.Second * 60)
				timeoutREST.Reset(time.Second * 30)
				m.priceOK = true
			})

			for {
				select {
				case <-timeoutREST.C:
					resp, err := okexv5api.GetTicker(instID)
					if err == nil && resp.Code == "0" {
						m.onTickerResp(resp.Data[0])
						timeoutReSub.Reset(time.Second * 60)
						timeoutREST.Reset(time.Second * 30)
						m.priceOK = true
					}
				case <-timeoutReSub.C:
					m.priceOK = false
					s.Reset()
				case <-updateTicker.C:
					if !m.subscribing {
						break
					}
				}
			}
		}
	}()

	// 订阅深度（5秒没有盘口就判定失败。深度订阅成功后会立即发送一次盘口）
	if !m.depthFromTicker {
		go func() {
			timeout := time.NewTicker(time.Second * 5)
			chBadDepth := make(chan int, 1)
			updateTicker := time.NewTicker(time.Second)
			s := m.ws.SubscribeDepth5(instID, func(resp interface{}) {
				if m.onDepthResp(resp) {
					// 推送
					for _, observer := range m.depthObservers {
						observer.(common.DepthObserver).OnDepthChanged()
					}
					timeout.Reset(time.Second * 5)
					m.depthOK = true
				} else {
					m.depthOK = false
					chBadDepth <- 0
				}
			})

			for {
				select {
				case <-timeout.C:
					m.depthOK = false
					s.Reset()
				case <-chBadDepth:
					m.depthOK = false
					s.Reset()
				case <-updateTicker.C:
					if !m.subscribing {
						break
					}
				}

			}
		}()
	} else {
		m.depthOK = false
	}
}

func (m *CommonMarket) unsubscribe(instID string) {
	m.subscribing = false
	m.ws.UnsubscribeTicker(instID)
	if !m.depthFromTicker {
		m.ws.UnsubscribeDepth5(instID)
	}
}

func (m *CommonMarket) onTickerResp(ticker okexv5api.TickerResp) {
	m.latestPrice = util.String2DecimalPanic(ticker.Last) // 最新成交价

	// ticker模拟深度
	if m.depthFromTicker {
		buy1 := util.String2DecimalPanic(ticker.Buy1)
		sell1 := util.String2DecimalPanic(ticker.Sell1)
		m.orderBook.Clear()
		m.orderBook.UpdateBids(buy1, decimal.NewFromInt(1))
		m.orderBook.UpdateAsk(sell1, decimal.NewFromInt(1))
		m.depthOK = true
	}
}

func (m *CommonMarket) onDepthResp(resp interface{}) bool {
	r := resp.(okexv5api.DepthWsResp)

	if r.Action != "update" { // "snapshot"/""
		m.orderBook.Clear()
	}

	// 构建/更新depth
	for _, depthUnit := range r.Data[0].Asks {
		price := util.String2DecimalPanic(depthUnit[0])
		amount := util.String2DecimalPanic(depthUnit[1])
		m.orderBook.UpdateAsk(price, amount)
	}

	for _, depthUnit := range r.Data[0].Bids {
		price := util.String2DecimalPanic(depthUnit[0])
		amount := util.String2DecimalPanic(depthUnit[1])
		m.orderBook.UpdateBids(price, amount)
	}

	// 验证checksum
	remoteChecksum := uint32(r.Data[0].Checksum)
	if remoteChecksum > 0 {
		localChecksum := m.depthCheckSum()

		if remoteChecksum != localChecksum {
			logger.LogImportant(logPrefix, "%s depth checksum failed, re-subscribe it", m.instId)
			return false
		} else {
			return true
		}
	} else {
		return true
	}
}

func (m *CommonMarket) depthCheckSum() uint32 {
	m.orderBook.Lock()
	askPrices := m.orderBook.Asks.Keys()
	askAmounts := m.orderBook.Asks.Values()
	bidPrices := m.orderBook.Bids.Keys()
	bidAmounts := m.orderBook.Bids.Values()
	m.orderBook.Unlock()

	numbers := make([]string, 0, len(askPrices)+len(bidPrices))

	for i := 0; i < 25; i++ {
		if i < len(bidPrices) {
			numbers = append(numbers, bidPrices[i].(decimal.Decimal).String())
			numbers = append(numbers, bidAmounts[i].(decimal.Decimal).String())
		}

		if i < len(askPrices) {
			numbers = append(numbers, askPrices[i].(decimal.Decimal).String())
			numbers = append(numbers, askAmounts[i].(decimal.Decimal).String())
		}
	}

	str := strings.Join(numbers, ":")
	return crc32.ChecksumIEEE([]byte(str))
}

// #region 实现common.Common_Market
func (m *CommonMarket) Type() string {
	return m.instId
}

func (m *CommonMarket) LatestPrice() decimal.Decimal {
	return m.latestPrice
}

func (m *CommonMarket) OrderBook() *common.Orderbook {
	return m.orderBook
}

func (m *CommonMarket) AlignPriceNumber(price decimal.Decimal) decimal.Decimal {
	return m.ex.instrumentMgr.AlignPriceNumber(m.instId, price)
}

func (m *CommonMarket) AlignPrice(price decimal.Decimal, dir common.OrderDir, makeOnly bool) decimal.Decimal {
	if price.IsZero() {
		return price
	} else {
		return m.ex.instrumentMgr.AlignPrice(m.instId, price, dir, makeOnly, m.orderBook.Buy1(), m.orderBook.Sell1())
	}
}

func (m *CommonMarket) AlignSize(size decimal.Decimal) decimal.Decimal {
	if size.IsZero() {
		return size
	} else {
		return m.ex.instrumentMgr.AlignSize(m.instId, size)
	}
}

func (m *CommonMarket) MinSize() decimal.Decimal {
	return m.ex.instrumentMgr.MinSize(m.instId, m.orderBook.Buy1())
}

// #endregion
