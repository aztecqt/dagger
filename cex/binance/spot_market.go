/*
 * @Author: aztec
 * @Date: 2023-02-27 10:33:40
 * @Description: 币安的现货行情。实现common.Market
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binance

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/api/binanceapi/binancespotapi"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/shopspring/decimal"
)

type SpotMarket struct {
	ex            *Exchange
	ws            *binancespotapi.WsClient
	instId        string
	inst          common.Instruments
	latestPrice   decimal.Decimal
	orderBook     *common.Orderbook
	detailedDepth bool

	priceOK bool
	depthOK bool

	// 深度变化回调。策略的主要驱动之一
	depthObserversSet *hashset.Set
	depthObservers    []interface{}

	subscribing bool
}

func (m *SpotMarket) Init(ex *Exchange, instID string, detailedDepth bool) {
	m.ex = ex
	m.ws = ex.wsSpot
	m.instId = instID
	m.inst = *ex.instrumentMgr.Get(instID)
	m.detailedDepth = detailedDepth
	m.orderBook = new(common.Orderbook)
	m.orderBook.Init()
	m.priceOK = false
	m.depthOK = false

	m.depthObserversSet = hashset.New()
	m.depthObservers = nil
	m.subscribing = false

	// 执行频道订阅
	m.subscribe(instID)
	logger.LogImportant(logPrefix, "spot market(%s) inited", instID)
}

func (m *SpotMarket) Uninit() {
	m.unsubscribe(m.instId)
	logger.LogImportant(logPrefix, "spot market(%s) uninited", m.instId)
}

func (m *SpotMarket) AddDepthObserver(obs common.DepthObserver) {
	m.depthObserversSet.Add(obs)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *SpotMarket) RemoveDepthObserver(obs common.DepthObserver) {
	m.depthObserversSet.Remove(obs)
	m.depthObservers = m.depthObserversSet.Values()
}

func (m *SpotMarket) subscribe(instID string) {
	m.subscribing = true

	// 订阅ticker（币安的ticker每秒发一次，30秒没收到就重新订阅一下）
	// 详细盘口模式订阅miniticker，简略盘口模式订阅完整ticker
	go func() {
		timeoutReSub := time.NewTicker(time.Second * 30)
		updateTicker := time.NewTicker(time.Second)
		var s *api.WsSubscriber
		if m.detailedDepth {
			s = m.ws.SubscribeMiniTicker(instID, func(resp interface{}) {
				ticker := resp.(*binanceapi.WSPayload_MiniTicker)
				m.onMiniTickerResp(ticker)
				timeoutReSub.Reset(time.Second * 30)
				m.priceOK = true
			})
		} else {
			s = m.ws.SubscribeTicker(instID, func(resp interface{}) {
				ticker := resp.(*binanceapi.WSPayload_Ticker)
				m.onTickerResp(ticker)
				timeoutReSub.Reset(time.Second * 30)
				m.priceOK = true
				m.depthOK = true
			})
		}

		for {
			select {
			case <-timeoutReSub.C:
				m.priceOK = false
				if !m.detailedDepth {
					m.depthOK = false
				}
				s.Reset()
			case <-updateTicker.C:
				if !m.subscribing {
					break
				}
			}
		}
	}()

	// 订阅深度（10秒没有盘口就判定失败）
	if m.detailedDepth {
		go func() {
			timeout := time.NewTicker(time.Second * 10)
			updateTicker := time.NewTicker(time.Second)
			s := m.ws.SubscribeDepth(instID, func(resp interface{}) {
				depth := resp.(*binanceapi.WSPayload_Depth)
				m.onDepthResp(depth)
				// 推送
				for _, observer := range m.depthObservers {
					observer.(common.DepthObserver).OnDepthChanged()
				}
				timeout.Reset(time.Second * 10)
				m.depthOK = true
			})

			for {
				select {
				case <-timeout.C:
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
		m.depthOK = true
	}
}

func (m *SpotMarket) unsubscribe(instID string) {
	m.subscribing = false
	if m.detailedDepth {
		m.ws.UnsubscribeMiniTicker(instID)
		m.ws.UnsubscribeDepth(instID)
	} else {
		m.ws.UnsubscribeTicker(instID)
	}

}

func (m *SpotMarket) onTickerResp(ticker *binanceapi.WSPayload_Ticker) {
	m.latestPrice = ticker.LatestPrice // 最新成交价

	// ticker模拟深度
	if !m.detailedDepth {
		m.orderBook.UpdateAsk(ticker.Buy1, ticker.Buy1Size)
		m.orderBook.UpdateAsk(ticker.Sell1, ticker.Sell1Size)
	}
}

func (m *SpotMarket) onMiniTickerResp(ticker *binanceapi.WSPayload_MiniTicker) {
	m.latestPrice = ticker.LatestPrice // 最新成交价
}

func (m *SpotMarket) onDepthResp(resp *binanceapi.WSPayload_Depth) {
	m.orderBook.Clear()

	// 构建/更新depth
	for _, depthUnit := range resp.Asks {
		m.orderBook.UpdateAsk(depthUnit[0], depthUnit[1])
	}

	for _, depthUnit := range resp.Bids {
		m.orderBook.UpdateBids(depthUnit[0], depthUnit[1])
	}
}

// #region 实现common.Common_Market
func (m *SpotMarket) Type() string {
	return m.instId
}

func (m *SpotMarket) String() string {
	bb := bytes.Buffer{}
	bb.WriteString(fmt.Sprintf("\nspot market: %s\n", m.instId))
	bb.WriteString(fmt.Sprintf("price: %s\n", m.latestPrice.String()))
	bb.WriteString("depth:\n")
	bb.WriteString(m.OrderBook().String(5))
	return bb.String()
}

func (m *SpotMarket) Ready() bool {
	return m.depthOK
}

func (m *SpotMarket) ReadyStr() string {
	return fmt.Sprintf("depth_ok:%v", m.depthOK)
}

func (m *SpotMarket) BaseCurrency() string {
	return m.inst.BaseCcy
}

func (m *SpotMarket) QuoteCurrency() string {
	return m.inst.QuoteCcy
}

func (m *SpotMarket) LatestPrice() decimal.Decimal {
	return m.latestPrice
}

func (m *SpotMarket) OrderBook() *common.Orderbook {
	return m.orderBook
}

func (m *SpotMarket) AlignPriceNumber(price decimal.Decimal) decimal.Decimal {
	return m.ex.instrumentMgr.AlignPriceNumber(m.instId, price)
}

func (m *SpotMarket) AlignPrice(price decimal.Decimal, dir common.OrderDir, makeOnly bool) decimal.Decimal {
	if price.IsZero() {
		return price
	} else {
		return m.ex.instrumentMgr.AlignPrice(m.instId, price, dir, makeOnly, m.orderBook.Buy1(), m.orderBook.Sell1())
	}
}

func (m *SpotMarket) AlignSize(size decimal.Decimal) decimal.Decimal {
	if size.IsZero() {
		return size
	} else {
		return m.ex.instrumentMgr.AlignSize(m.instId, size)
	}
}

func (m *SpotMarket) MinSize() decimal.Decimal {
	return m.ex.instrumentMgr.MinSize(m.instId, m.orderBook.Buy1())
}

// #endregion
