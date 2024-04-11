/*
- @Author: aztec
- @Date: 2024-02-07 09:55:51
- @Description: websocket
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package binancefutureapi

import (
	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/util/logger"
)

const wsLogPrefixCm = "binance_cm_ws"
const wsLogPrefixUm = "binance_um_ws"

type WsClient struct {
	publicStreams map[string]*binanceapi.WsStream
}

func logPrefix(isUsdt bool) string {
	if isUsdt {
		return wsLogPrefixUm
	} else {
		return wsLogPrefixCm
	}
}

func baseUrl(isUsdt bool) string {
	if isUsdt {
		return binanceapi.UmBaseUrl
	} else {
		return binanceapi.CmBaseUrl
	}
}

func (ws *WsClient) Start() {
	logger.LogImportant(wsLogPrefixCm, "starting...")
	logger.LogImportant(wsLogPrefixUm, "starting...")
	ws.publicStreams = make(map[string]*binanceapi.WsStream)
}

func (ws *WsClient) SubscribeContractInfo(fn api.OnRecvWSMsg, isUsdt bool) *api.WsSubscriber {
	streamName := "!contractInfo"
	s, stream := binanceapi.SubscribeWithStream[binanceapi.WsPayload_ContractInfo](baseUrl(isUsdt), streamName, logPrefix(isUsdt), fn)
	ws.publicStreams[streamName] = stream
	return s
}
