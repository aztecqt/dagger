/*
 * @Author: aztec
 * @Date: 2022-03-27 17:49:32
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2024-03-02 18:02:15
 * @FilePath: \stratergyc:\work\svn\go\src\dagger\api\ws_subscriber.go
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package api

import (
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"
)

type SubscribeTextGen func() string

const (
	Subscriber_status_not_started = iota
	Subscriber_status_subscribing
	Subscriber_status_successed
)

type WsSubscriber struct {
	name       string
	text       string
	actionName string
	gen        SubscribeTextGen
	succKeys   []string
	status     int // Subscriber_status_xxx
	onRecv     chan WSRawMsg
}

func (s *WsSubscriber) Init(name string, text string, isSubscriber bool, gen SubscribeTextGen, successKeys []string) {
	s.name = name
	s.text = text
	if isSubscriber {
		s.actionName = "subscribe"
	} else {
		s.actionName = "unsubscribe"
	}

	s.gen = gen
	s.succKeys = successKeys
	s.status = Subscriber_status_not_started
	s.onRecv = make(chan WSRawMsg)
}

func (s *WsSubscriber) startSubscribing() {
	s.status = Subscriber_status_subscribing
}

func (s *WsSubscriber) Reset() {
	s.status = Subscriber_status_subscribing
}

func (s *WsSubscriber) Subscribing() bool {
	return s.status == Subscriber_status_subscribing
}

func (s *WsSubscriber) Successed() bool {
	return s.status == Subscriber_status_successed
}

// 订阅器的逻辑：
// ws连接成功后，发送订阅字符串
// 然后等待服务器的特定返回
// 若等不到，则过段时间再发一次
// 成功订阅后，不再发送
// 直到被reset后，重复上述过程
func (s *WsSubscriber) run(ws *WsConnection) {
	s.status = Subscriber_status_not_started
	lastSendTime := time.Unix(0, 0)
	ticker := time.NewTicker(time.Millisecond * 100)

	for {
		select {
		case msg := <-s.onRecv:
			if s.status == Subscriber_status_subscribing {
				allMatch := true // 检测收到的消息是否匹配关键字
				for _, s := range s.succKeys {
					if !strings.Contains(msg.Str, s) {
						allMatch = false
					}
				}
				if allMatch {
					s.status = Subscriber_status_successed
					ws.RemoveRecvChans(s.onRecv)
					logger.LogInfo(ws.logPrefix, "%s [%s] success", s.actionName, s.name)
				}
			}
		case <-ticker.C:
			if s.status == Subscriber_status_subscribing &&
				ws.Connected() &&
				time.Now().UnixMilli()-lastSendTime.UnixMilli() > 5000 {
				logger.LogInfo(ws.logPrefix, "%s [%s] trying...", s.actionName, s.name)
				ws.AddRecvChans(s.onRecv)
				if s.gen != nil {
					ws.Send(s.gen())
				} else {
					ws.Send(s.text)
				}
				lastSendTime = time.Now()
			}
		}
	}
}
