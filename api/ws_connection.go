/*
 * @Author: aztec
 * @Date: 2022-03-27 08:51:12

 * @FilePath: \dagger\api\ws_connection.go
 * @Description:
 * 对ws的封装，包括conn的默认配置、订阅、重连等逻辑。不包含具体业务逻辑
 * 具体实现时，可组合这个struct以获取其能力
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/gorilla/websocket"
)

type WSRawMsg struct {
	LocalTime time.Time
	Data      []byte
	Str       string
}

type OnRecvWSRawMsg func(WSRawMsg)
type OnRecvWSMsg func(interface{})

var LogWebsocketDetail bool = false

// 主结构
type WsConnection struct {
	url         string
	logPrefix   string
	needStop    bool
	reConnCount int

	// ws连接
	Conn   *websocket.Conn
	muConn sync.Mutex

	// 订阅器
	subLogin  *WsSubscriber
	subOthers []*WsSubscriber
	muSubs    sync.Mutex
	needResub bool

	// 通道集合，主要用于跟subscriber交互
	muChans     sync.Mutex
	onRecvChans map[chan WSRawMsg]bool
	onConnChans map[chan int]bool

	// 消息接收回调，主要用于给消息处理
	onRecv OnRecvWSRawMsg
}

// 启动
func (ws *WsConnection) Start(url string, logPrefix string, onRecv OnRecvWSRawMsg) {
	ws.url = url
	ws.logPrefix = logPrefix
	ws.subOthers = make([]*WsSubscriber, 0)
	ws.onRecvChans = make(map[chan WSRawMsg]bool)
	ws.onConnChans = make(map[chan int]bool)
	ws.onRecv = onRecv

	// 启动主循环
	logger.LogImportant(logPrefix, "websocket starting...")
	go ws.keepConnecting()
	go ws.keepSubscribing()
}

func (ws *WsConnection) Stop() {
	logger.LogImportant(ws.logPrefix, "stopping...")
	ws.needStop = true
	if ws.Conn != nil {
		ws.Conn.Close()
		ws.Conn = nil
	}
}

func (ws *WsConnection) Connected() bool {
	return ws.Conn != nil
}

func (ws *WsConnection) Reconnect(reason string) {
	logger.LogImportant(ws.logPrefix, "need reconnect, reason=[%s], close current connection", reason)
	if ws.Conn != nil {
		ws.Conn.Close() // 关闭当前连接就会导致重连
		ws.Conn = nil
	}
}

func (ws *WsConnection) Ready() bool {
	if !ws.Connected() {
		return false
	}

	for _, s := range ws.subOthers {
		if !s.Successed() {
			return false
		}
	}

	return true
}

// 订阅
func (ws *WsConnection) Login(s *WsSubscriber) {
	go s.run(ws)
	ws.subLogin = s
}

func (ws *WsConnection) Subscribe(s *WsSubscriber) {
	go s.run(ws)
	ws.muSubs.Lock()
	ws.subOthers = append(ws.subOthers, s)
	ws.muSubs.Unlock()
}

// 连接到服务器。连接不成功则一直连接
func (ws *WsConnection) connect() {
	if ws.Conn != nil {
		ws.Conn.Close()
		ws.Conn = nil
	}

	logger.LogImportant(ws.logPrefix, "connecting...(%d) url=%s", ws.reConnCount, ws.url)
	ws.reConnCount++

	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 5 * time.Second}
	for i := 0; ; i++ {
		logger.LogImportant(ws.logPrefix, "dialing....(%d)", i)
		c, _, err := dialer.Dial(ws.url, nil)
		if err == nil {
			logger.LogImportant(ws.logPrefix, "connect success, local addr:%s, remote addr: %s", c.LocalAddr().String(), c.RemoteAddr().String())
			c.SetReadDeadline(time.Time{}) // 读取永不超时
			c.SetPingHandler(func(appData string) error {
				return c.WriteMessage(websocket.PongMessage, nil)
			})
			ws.Conn = c
			break
		} else {
			logger.LogImportant(ws.logPrefix, "dailing failed, retry in 5 seconds...")
			logger.LogImportant(ws.logPrefix, "err=%s", err.Error())
			time.Sleep(time.Second * 5)
		}
	}
}

// 发送消息
func (ws *WsConnection) Send(msg string) {
	if ws.Conn != nil {
		func() {
			ws.muConn.Lock()
			defer ws.muConn.Unlock()
			defer util.DefaultRecover()
			err := ws.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				logger.LogImportant(ws.logPrefix, "send message failed, msg=%s, err=%s", msg, err.Error())
			} else {
				if LogWebsocketDetail {
					logger.LogDebug(ws.logPrefix, "send: %s", msg)
				}
			}
		}()
	} else {
		logger.LogImportant(ws.logPrefix, "conn not ready yet")
	}
}

// 接收消息
func (ws *WsConnection) readMessage() {
	for {
		needReconnect := func() bool {
			defer util.DefaultRecover()
			if ws.Conn != nil {
				messageType, msgData, err := ws.Conn.ReadMessage()
				if err != nil {
					logger.LogImportant(ws.logPrefix, "readMessage error: %s", err.Error())
					logger.LogImportant(ws.logPrefix, "reconnect...")
					return true
				} else {
					var msgStr string
					switch messageType {
					case websocket.TextMessage: // 文本消息
						msgStr = string(msgData)
					case websocket.BinaryMessage: // 压缩消息
						msgDecode, err := util.GzipDecode(msgData)
						if err == nil {
							msgStr = string(msgDecode)
						} else {
							logger.LogImportant(ws.logPrefix, "readMessage decode error: %s", err.Error())
						}
					}

					msg := WSRawMsg{LocalTime: time.Now(), Data: msgData, Str: msgStr}

					if LogWebsocketDetail {
						logger.LogDebug(ws.logPrefix, "recv: %s", msgStr)
					}

					ws.onRecv(msg)
					ws.notifyMessageToChans(msg)
				}

				return false
			} else {
				return true
			}
		}()

		if needReconnect {
			break
		}
	}
}

// 订阅器逻辑循环
func (ws *WsConnection) keepSubscribing() {
	for {
		// 重新订阅
		if ws.needResub {
			if ws.subLogin != nil {
				ws.subLogin.Reset()
			}

			ws.muSubs.Lock()
			for _, s := range ws.subOthers {
				s.Reset()
			}
			ws.muSubs.Unlock()
			ws.needResub = false
		}

		// 优先保证login成功
		processOthers := false
		if ws.subLogin == nil {
			processOthers = true
		} else if ws.subLogin.Successed() {
			processOthers = true
		} else if !ws.subLogin.Subscribing() && !ws.subLogin.Successed() {
			ws.subLogin.startSubscribing(ws)
		}

		// 然后保证其他订阅器成功
		if processOthers {
			ws.muSubs.Lock()
			for _, s := range ws.subOthers {
				if !s.Subscribing() && !s.Successed() {
					s.startSubscribing(ws)
				}
			}

			// 清理订阅完毕的订阅器
			// 为了简化处理，一个轮询只清理一个
			for i, s := range ws.subOthers {
				if s.Successed() {
					util.SliceRemoveAt(ws.subOthers, i)
					break
				}
			}
			ws.muSubs.Unlock()
		}

		time.Sleep(time.Millisecond * 100)
	}
}

// 连接逻辑循环
func (ws *WsConnection) keepConnecting() {
	for {
		if !ws.needStop {
			ws.connect()
			ws.needResub = true
			ws.notifyConnectingToChans()
			ws.readMessage()
		} else {
			break
		}
	}
}

// #region 通道操作
func (ws *WsConnection) AddRecvChans(onRecv chan WSRawMsg) {
	ws.muChans.Lock()
	defer ws.muChans.Unlock()
	ws.onRecvChans[onRecv] = true
}

func (ws *WsConnection) AddConnChans(onConn chan int) {
	ws.muChans.Lock()
	defer ws.muChans.Unlock()
	ws.onConnChans[onConn] = true
}

func (ws *WsConnection) RemoveRecvChans(onRecv chan WSRawMsg) {
	ws.muChans.Lock()
	defer ws.muChans.Unlock()
	delete(ws.onRecvChans, onRecv)
}

func (ws *WsConnection) RemoveConnChans(onConn chan int) {
	ws.muChans.Lock()
	defer ws.muChans.Unlock()
	delete(ws.onConnChans, onConn)
}

func (ws *WsConnection) notifyConnectingToChans() {
	ws.muChans.Lock()
	chans := make([]chan int, 0, len(ws.onConnChans))
	for c := range ws.onConnChans {
		chans = append(chans, c)
	}
	ws.muChans.Unlock()

	for _, c := range chans {
		c <- ws.reConnCount
	}
}

func (ws *WsConnection) notifyMessageToChans(msg WSRawMsg) {
	ws.muChans.Lock()
	chans := make([]chan WSRawMsg, 0, len(ws.onRecvChans))
	for c := range ws.onRecvChans {
		chans = append(chans, c)
	}
	ws.muChans.Unlock()

	for _, c := range chans {
		c <- msg
	}
}

// #endregion
