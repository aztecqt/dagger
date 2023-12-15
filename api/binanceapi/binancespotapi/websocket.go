/*
 * @Author: aztec
 * @Date: 2023-02-12 15:22:21
 * @Description: 币安的websocket，由多个stream构成
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package binancespotapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/api/binanceapi"
	"github.com/aztecqt/dagger/util/logger"
)

const wsLogPrefix = "binance_spot_ws"

type WsClient struct {
	userStream    *binanceapi.WsStream
	publicStreams map[string]*binanceapi.WsStream
}

func subscribeWithStream[T any](streamName string, fn api.OnRecvWSMsg) (*api.WsSubscriber, *binanceapi.WsStream) {
	stream := new(binanceapi.WsStream)
	s := stream.Start(streamName, func(rawMsg api.WSRawMsg) {
		if !strings.Contains(rawMsg.Str, "result") {
			// 将rawMsg序列化成对象，并返回
			t := new(T)
			err := json.Unmarshal(rawMsg.Data, t)
			if err == nil {
				fn(t)
			} else {
				logger.LogImportant(wsLogPrefix, err.Error())
			}
		}
	})

	return s, stream
}

func (ws *WsClient) Start() {
	logger.LogImportant(wsLogPrefix, "starting...")
	ws.publicStreams = make(map[string]*binanceapi.WsStream)
}

func (ws *WsClient) SubscribeTicker(pair string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@ticker", pair)
	s, stream := subscribeWithStream[binanceapi.WSPayload_Ticker](streamName, fn)
	ws.publicStreams[streamName] = stream
	return s
}

func (ws *WsClient) UnsubscribeTicker(pair string) {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@ticker", pair)
	if stream, ok := ws.publicStreams[streamName]; ok {
		stream.Stop()
		delete(ws.publicStreams, streamName)
	}
}

func (ws *WsClient) SubscribeMiniTicker(pair string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@miniTicker", pair)
	s, stream := subscribeWithStream[binanceapi.WSPayload_MiniTicker](streamName, fn)
	ws.publicStreams[streamName] = stream
	return s
}

func (ws *WsClient) UnsubscribeMiniTicker(pair string) {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@miniTicker", pair)
	if stream, ok := ws.publicStreams[streamName]; ok {
		stream.Stop()
		delete(ws.publicStreams, streamName)
	}
}

func (ws *WsClient) SubscribeDepth(pair string, fn api.OnRecvWSMsg) *api.WsSubscriber {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@depth10@100ms", pair)
	s, stream := subscribeWithStream[binanceapi.WSPayload_Depth](streamName, fn)
	ws.publicStreams[streamName] = stream
	return s
}

func (ws *WsClient) UnsubscribeDepth(pair string) {
	pair = strings.ToLower(pair)
	streamName := fmt.Sprintf("%s@depth10@100ms", pair)
	if stream, ok := ws.publicStreams[streamName]; ok {
		stream.Stop()
		delete(ws.publicStreams, streamName)
	}
}

// 订阅用户信息需要先获取ListenKey，并且每间隔一段时间就保活这个ListenKey
// 暂时每处理保活失败的情况，仅输出日志
func (ws *WsClient) SubscribeUserData(fnAccountUpdate, fnOrderUpdate api.OnRecvWSMsg) *api.WsSubscriber {
	resp, err := GetListenKey()
	if err != nil {
		logger.LogImportant(wsLogPrefix, "get listen-key failed, err=%s", err.Error())
		return nil
	} else if resp.Code != 0 {
		logger.LogImportant(wsLogPrefix, "get listen-key failed, code=%d, msg=%s", resp.Code, resp.Message)
		return nil
	} else if len(resp.ListenKey) == 0 {
		logger.LogImportant(wsLogPrefix, "get listen-key failed, no key")
		return nil
	} else {
		listenKey := resp.ListenKey
		if ws.userStream == nil {
			ws.userStream = new(binanceapi.WsStream)
			s := ws.userStream.Start(listenKey, func(rawMsg api.WSRawMsg) {
				localTime := time.Now()
				if !strings.Contains(rawMsg.Str, "result") {
					// 将rawMsg序列化成对象，并返回
					payload := binanceapi.WSPayload_Common{}
					json.Unmarshal(rawMsg.Data, &payload)
					if payload.EventType == binanceapi.WSPayloadEventType_AccountUpdate {
						au := binanceapi.WSPayload_AccountUpdate{}
						json.Unmarshal(rawMsg.Data, &au)
						if fnAccountUpdate != nil {
							fnAccountUpdate(au)
						}
					} else if payload.EventType == binanceapi.WSPayloadEventType_OrderUpdate {
						ou := binanceapi.WSPayload_OrderUpdate{}
						json.Unmarshal(rawMsg.Data, &ou)
						ou.LocalTime = localTime
						if fnOrderUpdate != nil {
							fnOrderUpdate(ou)
						}
					}
				}
			})

			go func() {
				for ws.userStream != nil /*代表没有反订阅*/ {
					time.Sleep(time.Minute * 10)
					KeepListenKey(listenKey)
				}
			}()

			return s
		} else {
			return nil
		}
	}
}

func (ws *WsClient) UnsubscribeUserData() {
	if ws.userStream != nil {
		ws.userStream.Stop()
		ws.userStream = nil
	}
}
