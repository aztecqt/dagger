/*
 * @Author: aztec
 * @Date: 2022-06-15 11:43
 * @LastEditors: aztec
 * @FilePath: \stratergyc:\svn\quant\go\src\dagger\util\udpsocket\socket.go
 * @Description:
 * 封装了一个带加密的socket连接
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package udpsocket

import (
	"encoding/json"
	"net"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
)

const logPrefix = "udp-socket"

var LogSocketDetail bool = false

type FnOnRecv func(op string, data []byte, addr *net.UDPAddr)

type Socket struct {
	socket *net.UDPConn
	onRecv FnOnRecv
}

func (s *Socket) Connect(serverAddr string, serverPort int, onRecv FnOnRecv) bool {
	// 解析服务器ip
	ip, err := net.ResolveIPAddr("ip", serverAddr)
	if err != nil {
		logger.LogImportant(logPrefix, "resolve server add(%s) failed!, err=%s", serverAddr, err.Error())
		return false
	} else {
		logger.LogInfo(logPrefix, "resolved server addr: %s:%s", ip.String(), serverAddr)
	}

	s.socket, err = net.DialUDP("udp", nil, &net.UDPAddr{IP: ip.IP, Port: serverPort})
	if err != nil {
		logger.LogImportant(logPrefix, "dial to server failed! err=%s", err)
		return false
	} else {
		logger.LogInfo(logPrefix, "dial to server success")
	}

	s.onRecv = onRecv
	go s.doRecv()
	return true
}

func (s *Socket) Listen(localPort int, onRecv FnOnRecv) bool {
	laddr := new(net.UDPAddr)
	laddr.Port = localPort
	sk, err := net.ListenUDP("udp", laddr)
	if err != nil {
		logger.LogImportant(logPrefix, "listen udp failed, err=%s", err.Error())
		return false
	} else {
		s.socket = sk
		logger.LogInfo(logPrefix, "start listen at %s", sk.LocalAddr().String())
	}

	s.onRecv = onRecv
	go s.doRecv()
	return true
}

func (s *Socket) Close() {
	s.socket.Close()
}

func (s *Socket) Send(data []byte) {
	defer util.DefaultRecover()
	if s.socket != nil {
		_, err := s.socket.Write(data)
		if err != nil {
			logger.LogInfo(logPrefix, "write to socket err: %s", err.Error())
		} else if LogSocketDetail {
			logger.LogDebug(logPrefix, "send to %s: %s", s.socket.RemoteAddr().String(), string(data))
		}
	}
}

func (s *Socket) SendString(str string) {
	s.Send([]byte(str))
}

func (s *Socket) SendObj(obj interface{}) {
	b, err := json.Marshal(obj)
	if err == nil {
		s.Send(b)
	}
}

func (s *Socket) SendTo(data []byte, addr *net.UDPAddr) {
	_, err := s.socket.WriteToUDP(data, addr)
	if err != nil {
		logger.LogInfo(logPrefix, "write to socket err: %s", err.Error())
	} else if LogSocketDetail {
		logger.LogDebug(logPrefix, "send to %s: %s", addr.String(), string(data))
	}
}

func (s *Socket) SendStringTo(str string, addr *net.UDPAddr) {
	s.SendTo([]byte(str), addr)
}

func (s *Socket) SendObjTo(obj interface{}, addr *net.UDPAddr) {
	b, err := json.Marshal(obj)
	if err == nil {
		s.SendTo(b, addr)
	}
}

func (s *Socket) doRecv() {
	data := make([]byte, 4096)
	for {
		func() {
			defer util.DefaultRecover()
			n, addr, err := s.socket.ReadFromUDP(data)

			// TODO：是否需要将data拼接成一个完整的json字符串?
			if err != nil {
				logger.LogInfo(logPrefix, "receive from socket err: %s", err.Error())
			} else {
				h := Header{}
				err := json.Unmarshal(data[:n], &h)
				if err != nil {
					logger.LogInfo(logPrefix, "unmarshal msg failed, err: %s", err.Error())
				} else {
					// TODO:MD5验证
					// 回调外部
					s.onRecv(h.OP, data[:n], addr)
				}

				if LogSocketDetail {
					logger.LogDebug(logPrefix, "recv from %s: %s", addr.String(), string(data[:n]))
				}
			}
		}()
	}
}
