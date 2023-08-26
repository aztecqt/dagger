/*
 * @Author: aztec
 * @Date: 2022-03-28 18:00:04
 * @Description: binance ws的ping逻辑
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package api

import (
	"time"

	"aztecqt/dagger/util/logger"
)

type Pinger struct {
	wsConn *WsConnection
}

// pinger做这些事情：
// 每过一段时间发送指定的ping字符串
// 超过一段时间每收到任何数据，则重连
// 这个逻辑现在可以适配okx和bn的情况
func (p *Pinger) Start(wsConn *WsConnection, logPrfix, sendStr string, sendInterval, reconnectInterval int64) {
	p.wsConn = wsConn

	go func() {
		recvChan := make(chan WSRawMsg)
		connChan := make(chan int)
		wsConn.AddRecvChans(recvChan)
		wsConn.AddConnChans(connChan)
		defer wsConn.RemoveRecvChans(recvChan)
		defer wsConn.RemoveConnChans(connChan)
		tk := time.NewTicker(time.Second)
		lastRecvTime := time.Now()
		pingSended := false

		for {
			select {
			case <-tk.C:
				if wsConn.Connected() {
					deltaSeconds := time.Now().Unix() - lastRecvTime.Unix()
					if deltaSeconds > sendInterval && !pingSended {
						// 发送
						logger.LogInfo(logPrfix, "pinger: sending ping")
						wsConn.Send(sendStr)
						pingSended = true
					} else if deltaSeconds > reconnectInterval {
						// 重连
						wsConn.Reconnect("pong-time-out")
					}
				}
			case <-connChan:
				lastRecvTime = time.Now()
				logger.LogInfo(logPrfix, "pinger: connected")
				pingSended = false
			case <-recvChan:
				lastRecvTime = time.Now()
				pingSended = false
			}
		}
	}()
}
