/*
- @Author: aztec
- @Date: 2024-02-27 16:06:17
- @Description: tws客户端。维持一个本地的TCP连接。按照tws的规则进行消息的收发
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const logPrefix = "twsapi"
const maxTimeOutCount = 3

type Message struct {
	MsgId IncommingMessage
	Msg   interface{}
}

type MessageHandler func(Message)
type OnConnectHandler func()

type Client struct {
	sync.Mutex
	connected bool // 此标志为true时，表示conn已经成功创建，且完成了握手过程
	conn      net.Conn
	addr      string
	port      int

	// 用于区分不同客户端
	// 目前统一传1
	clientId int

	// 服务器返回数据
	accounts      []string
	serverVersion int
	nextOrderId   int

	// 发送缓存和接收缓存
	sendBuffer           []byte
	recvBuffer           []byte
	optionalCapabilities string

	// reqId自动累加
	reqId int

	// 连续超时逻辑
	// 遇到过一些情况下，tws客户端运行正常，底层tcp连接没有中断，但是发什么都是超时
	// 所以记录一下连续超时的次数。当连续超时大于n次后，尝试重新建立tcp连接，或许能解决这个问题
	timeOutCountSeq int

	// 消息处理
	msgHandlerNextId int
	msgHandlers      map[int]MessageHandler

	// 连接回调
	onConnectHandlerNextId int
	onConnectHandlers      map[int]OnConnectHandler
}

// tws登录握手过程：
// tcp连接
// 发送API连接消息
// 收到服务器版本号/服务器时间等
// 发送StartApi消息
// 收到Account列表/NextOrderId
// 握手结束
func (c *Client) Connect() {
	go func() {
		for {
			logInfo(logPrefix, "connecting")
			if !c.doConnect() {
				logInfo(logPrefix, "connect failed, retry after 3 seconds")
				time.Sleep(time.Second * 3)
			}

			for c.IsConnectOk() {
				time.Sleep(time.Second)
			}
		}
	}()

	for !c.IsConnectOk() {
		time.Sleep(time.Millisecond)
	}
}

func (c *Client) doConnect() bool {
	c.serverVersion = 0
	c.nextOrderId = 0
	c.reqId = 0
	logInfo(logPrefix, "dialing to %s:%d", c.addr, c.port)
	if conn, e := net.Dial("tcp", fmt.Sprintf("%s:%d", c.addr, c.port)); e == nil {
		logInfo(logPrefix, "dial success")
		c.conn = conn
		go c.doRecv(conn)
		logInfo("tcp connected to %s:%d", c.addr, c.port)
		c.connectApi()

		connOk := false
		for i := 0; i < 100; i++ {
			if c.nextOrderId > 0 && len(c.accounts) > 0 {
				connOk = true
				break
			} else {
				time.Sleep(time.Millisecond * 100)
			}
		}

		if connOk {
			logInfo(logPrefix, "connect success!")
			c.connected = true
			for _, fn := range c.onConnectHandlers {
				fn()
			}
			return true
		} else {
			logInfo(logPrefix, "get nextOrderId time out")
		}
	} else {
		logInfo("dial failed: %s", e.Error())
	}

	return false
}

func (c *Client) Reconnect(reason string) {
	logInfo(logPrefix, "need reconnect, reason: %s", reason)
	c.connected = false
	c.conn = nil
	c.nextOrderId = 0
}

func (c *Client) IsConnectOk() bool {
	return c.connected
}

func (c *Client) Disconnect() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

// 注册消息回调
func (c *Client) RegisterMessageHandler(fn MessageHandler) int {
	c.Lock()
	defer c.Unlock()
	c.msgHandlerNextId++
	c.msgHandlers[c.msgHandlerNextId] = fn
	return c.msgHandlerNextId
}

func (c *Client) UnregisterMessageHandler(id int) {
	c.Lock()
	defer c.Unlock()
	delete(c.msgHandlers, id)
}

// 注册断线重连的回调
func (c *Client) RegisterOnConnectCallback(fn func()) int {
	c.Lock()
	defer c.Unlock()
	c.onConnectHandlerNextId++
	c.onConnectHandlers[c.onConnectHandlerNextId] = fn
	return c.onConnectHandlerNextId
}

func (c *Client) UnregisterOnConnectCallback(id int) {
	c.Lock()
	defer c.Unlock()
	delete(c.onConnectHandlers, id)
}

func (c *Client) onMessage(msgId IncommingMessage, msg interface{}) {
	c.Lock()
	handlers := make([]MessageHandler, 0, len(c.msgHandlers))
	for _, v := range c.msgHandlers {
		handlers = append(handlers, v)
	}
	c.Unlock()

	for _, fn := range handlers {
		fn(Message{MsgId: msgId, Msg: msg})
	}
}

func (c *Client) connectApi() {
	// 发送连接请求
	// v100..176为客户端可接受的版本号。这里简化处理，不再保留原版的一大堆常量定义
	// 原始实现见c#例子中的sendConnectRequest()函数
	c.sendWithPrefix("API", false, "v100..176", c.optionalCapabilities)
}

func (c *Client) startApi() {
	// 注意这里不能checkConnection，因为登录握手过程还没完全结束，check不会成功
	ver := 2
	c.send(false, OutgoingMessage_StartApi, ver, c.clientId, c.optionalCapabilities)
}

func (c *Client) nextReqId() int {
	c.reqId++
	id := c.reqId
	return id
}

func (c *Client) NextOrderId() int {
	id := c.nextOrderId
	c.nextOrderId++
	return id
}

func (c *Client) Accounts() []string {
	return c.accounts
}
