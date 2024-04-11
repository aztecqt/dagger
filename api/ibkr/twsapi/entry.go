/*
- @Author: aztec
- @Date: 2024-02-27 16:15:23
- @Description: 入口
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

import "fmt"

// 日志接口
type FnLog func(msg string)

var logInfoFn FnLog
var logDebugFn FnLog
var logErrorFn FnLog

func logDebug(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logDebugFn(msg)
}

func logInfo(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logInfoFn(msg)
}

func logError(logPrefix string, format string, params ...interface{}) {
	msg := fmt.Sprintf(format, params...)
	msg = fmt.Sprintf("[%s] %s", logPrefix, msg)
	logErrorFn(msg)
}

func Init(logInfo, logDebug, logError FnLog) {
	logInfoFn = logInfo
	logDebugFn = logDebug
	logErrorFn = logError
}

func NewClient(addr string, port int) *Client {
	return &Client{
		addr:              addr,
		port:              port,
		clientId:          1, // hard code
		msgHandlers:       make(map[int]MessageHandler),
		onConnectHandlers: make(map[int]OnConnectHandler),
		sendBuffer:        make([]byte, 1024),
		recvBuffer:        make([]byte, 1024*1024*32)}
}
