/*
 * @Author: aztec
 * @Date: 2022-04-19 12:13:25
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-10-03 10:57:02
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\cex\okexv5\spot_market.go
 * @Description: okex现货行情
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5

import (
	"bytes"
	"fmt"

	"github.com/aztecqt/dagger/util/logger"
)

type SpotMarket struct {
	CommonMarket
	baseCcy  string
	quoteCcy string
}

func (m *SpotMarket) Init(ex *Exchange, instID, baseCcy, quoteCcy string, detailedDepth, tickerFromRest bool) {
	m.CommonMarket.Init(ex, instID, detailedDepth, tickerFromRest)

	m.baseCcy = baseCcy
	m.quoteCcy = quoteCcy

	// 执行频道订阅
	m.subscribe(instID)
	logger.LogImportant(logPrefix, "spot market(%s) inited", instID)
}

func (m *SpotMarket) Uninit() {
	// 取消订阅
	m.unsubscribe(m.instId)
	logger.LogImportant(logPrefix, "spot market(%s) uninited", m.instId)
}

// #region 实现common.SpotMarket
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
	return m.baseCcy
}

func (m *SpotMarket) QuoteCurrency() string {
	return m.quoteCcy
}

// #endregion
