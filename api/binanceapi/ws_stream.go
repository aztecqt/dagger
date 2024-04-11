/*
 * @Author: aztec
 * @Date: 2023-02-12 15:22:21
 * @Description: 币安的websocket-stream。由于每个频道(stream)的URL不同，所以每个频道需要一个单独的WsStream对象。
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binanceapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/util/logger"
)

const SpotBaseUrl = "wss://stream.binance.com:9443/ws/"
const CmBaseUrl = "wss://fstream.binance.com/ws/"
const UmBaseUrl = "wss://dstream.binance.com/ws/"
const wsLogPrefix = "binance_ws"

var wsSubscribeId int

type WsStream struct {
	wsConn api.WsConnection
}

func (ws *WsStream) Start(baseUrl, streamName string, fnOnRawMsg api.OnRecvWSRawMsg) *api.WsSubscriber {
	logger.LogImportant(wsLogPrefix, "starting...")
	url := fmt.Sprintf("%s%s", baseUrl, streamName)
	ws.wsConn.Start(url, wsLogPrefix, fnOnRawMsg)

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

func SubscribeWithStream[T any](baseUrl, streamName, logPrefix string, fn api.OnRecvWSMsg) (*api.WsSubscriber, *WsStream) {
	stream := new(WsStream)
	s := stream.Start(baseUrl, streamName, func(rawMsg api.WSRawMsg) {
		if !strings.Contains(rawMsg.Str, "result") {
			// 将rawMsg序列化成对象，并返回
			t := new(T)
			err := json.Unmarshal(rawMsg.Data, t)
			if err == nil {
				fn(t)
			} else {
				logger.LogImportant(logPrefix, err.Error())
			}
		}
	})

	return s, stream
}
