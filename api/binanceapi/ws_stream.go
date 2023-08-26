/*
 * @Author: aztec
 * @Date: 2023-02-12 15:22:21
 * @Description: 币安的websocket-stream。由于每个频道(stream)的URL不同，所以每个频道需要一个单独的WsStream对象。
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binanceapi

import (
	"fmt"

	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/util/logger"
)

const baseURL = "wss://stream.binance.com:9443/ws/"
const wsLogPrefix = "binance_ws"

var wsSubscribeId int

type WsStream struct {
	wsConn api.WsConnection
}

func (ws *WsStream) Start(streamName string, fnOnRawMsg api.OnRecvWSRawMsg) *api.WsSubscriber {
	logger.LogImportant(wsLogPrefix, "starting...")
	url := fmt.Sprintf("%s%s", baseURL, streamName)
	ws.wsConn.Start(url, wsLogPrefix, fnOnRawMsg)
	// p1 := api.Pinger{}
	// p1.Start(&ws.wsConn, wsLogPrefix, `{"pong"}`, 60*3, 60*6)

	id := wsSubscribeId
	wsSubscribeId++

	s := new(api.WsSubscriber)
	s.Init(
		streamName,
		fmt.Sprintf(`{"method":"SUBSCRIBE","params":["%s"],"id": %d}`, streamName, id),
		true,
		nil,
		[]string{fmt.Sprintf(`"id":%d`, id), `"result":null`})
	ws.wsConn.Subscribe(s)
	return s
}

func (ws *WsStream) Stop() {
	ws.wsConn.Stop()
}
