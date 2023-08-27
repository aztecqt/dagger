/*
 * @Author: aztec
 * @Date: 2022-05-27 10:14:00
 * @LastEditors: Please set LastEditors
 * @FilePath: \dagger\util\apikey\requester.go
 * @Description: 从apikey服务器请求一个符合条件的Key
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package apikey

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/crypto"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/udpsocket"
)

type FnKeyAquired func(k, s, p string)

type Requester struct {
	logPrefix string
	Key       string
	Secret    string
	Password  string
	key_enc   string // 加密过的key，用于keepkey
	deskey    string //解密密钥
	chStop    chan int
	fn        FnKeyAquired
}

func (r *Requester) Go(exchange, account, user string, serverAddr string, serverPort int, fn FnKeyAquired) {
	r.logPrefix = fmt.Sprintf("apikey-%s-%s", exchange, account)
	logger.LogInfo(r.logPrefix, "start")

	us := udpsocket.Socket{}
	if !us.Connect(serverAddr, serverPort, r.onRecvUDPMessage) {
		logger.LogPanic(r.logPrefix, "connect to key server failed!")
	}

	r.fn = fn
	r.chStop = make(chan int)

	if b, err := os.ReadFile("des.private"); err == nil {
		r.deskey = string(b)
	} else {
		logger.LogPanic(r.logPrefix, "read deskey failed")
	}
	go r.update(exchange, account, user, &us)
}

func (r *Requester) onRecvUDPMessage(op string, data []byte, addr *net.UDPAddr) {
	switch op {
	case opGetKeyAck:
		ack := getKeyAck{}
		err := json.Unmarshal(data, &ack)
		if err != nil {
			logger.LogInfo(r.logPrefix, "unmarshal msg failed: err: %s", err.Error())
		}
		if ack.Result {
			bkey, err1 := hex.DecodeString(ack.Key)
			bsec, err2 := hex.DecodeString(ack.Secret)
			bpass, err3 := hex.DecodeString(ack.Password)
			if err1 == nil && err2 == nil && err3 == nil {
				key := r.deskey
				r.Key = string(crypto.DesCBCDecrypter(bkey, []byte(key), []byte(key)))
				r.Secret = string(crypto.DesCBCDecrypter(bsec, []byte(key), []byte(key)))
				r.Password = string(crypto.DesCBCDecrypter(bpass, []byte(key), []byte(key)))
				r.key_enc = ack.Key
				logger.LogInfo(r.logPrefix, "get key success!")

				if r.fn != nil {
					r.fn(r.Key, r.Secret, r.Password)
				}
			} else {
				logger.LogInfo(r.logPrefix, "decrypt msg failed")
			}
		}
	case opKeepKeyAck:
		ack := keepKeyAck{}
		err := json.Unmarshal(data, &ack)
		if err != nil {
			logger.LogInfo(r.logPrefix, "unmarshal msg failed: err: %s", err.Error())
		}

		// TODO：更安全的处理方式
		if !ack.Result {
			logger.LogPanic(r.logPrefix, "keep apikey failed!")
		} else {
			logger.LogInfo(r.logPrefix, "keep apikey success")
		}
	}
}

func (r *Requester) update(exchange, account, user string, us *udpsocket.Socket) {
	// 先请求一次
	req := newGetKeyReq(account, exchange, user)
	us.SendString(req)

	// 每5秒请求/维持一次
	ticker := time.NewTicker(time.Second * 5)
	for {
		func() {
			defer util.DefaultRecover()

			select {
			case <-ticker.C:
				if len(r.Key) == 0 {
					req := newGetKeyReq(account, exchange, user)
					us.SendString(req)
				} else {
					req := newKeepKeyReq(account, exchange, user, r.key_enc)
					us.SendString(req)
				}
			case <-r.chStop:
				// 归还key
				if len(r.Key) > 0 {
					req := newReleaseKeyReq(account, exchange, user)
					us.SendString(req)
					logger.LogInfo(r.logPrefix, "key released")
				}

				us.Close()
				return
			}
		}()
	}
}
