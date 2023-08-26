/*
 * @Author: aztec
 * @Date: 2022-03-26 10:22:43
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-08-26 13:10:17
 * @FilePath: \dagger\api\okexv5api\signer.go
 * @Description: okex消息签名器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package okexv5api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

type signer struct {
	key               string
	secret            string
	pass              string
	serverTimeDeltaMS int64 // 服务器时间差
}

var signerIns *signer
var signerLogPrefix = "okexv5_signer"

var inited bool = false

func Init(key string, secret string, pass string) {
	signerIns = new(signer)
	signerIns.key = key
	signerIns.secret = secret
	signerIns.pass = pass

	// 获取服务器时间跟本地时间的差
	timeOk := false
	go func() {
		for {
			serverTime := GetServerTS()
			if serverTime > 0 {
				signerIns.serverTimeDeltaMS = serverTime - util.TimeNowUnix13()
				timeOk = true
				time.Sleep(time.Minute)
			} else {
				logger.LogImportant(signerLogPrefix, "get server time failed...retry after 1 second")
				time.Sleep(time.Second)
			}
		}
	}()

	for {
		if timeOk {
			break
		} else {
			time.Sleep(time.Millisecond * 100)
		}
	}

	inited = true
}

func HasKey() bool {
	return len(signerIns.key) > 0 && len(signerIns.secret) > 0 && len(signerIns.pass) > 0
}

func getParamHmacSHA256Sign(message string, secretKey string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, err := mac.Write([]byte(message))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *signer) shar256(timestamp string, method string, action string, body string) string {
	if !inited {
		logger.LogPanic(signerLogPrefix, "not inited")
	}

	if len(s.key) == 0 || len(s.secret) == 0 {
		logger.LogPanic(signerLogPrefix, "no valid key")
	}

	bb := bytes.Buffer{}
	bb.WriteString(timestamp)
	bb.WriteString(method)
	bb.WriteString(action)
	bb.WriteString(body)

	sign, err := getParamHmacSHA256Sign(bb.String(), s.secret)
	if err == nil {
		return sign
	} else {
		logger.LogPanic(signerLogPrefix, "sign error!")
		return ""
	}
}

func (s *signer) serverTimeUnix13() int64 {
	return util.TimeNowUnix13() + s.serverTimeDeltaMS
}

func (s *signer) serverTimeUnix11() int64 {
	return (util.TimeNowUnix13() + s.serverTimeDeltaMS) / 1000
}

func (s *signer) signWithIsoTs(method string, action string, body string) (string, string) {
	timestamp := util.ConvetUnix13ToIsoTime(s.serverTimeUnix13())
	return s.shar256(timestamp, method, action, body), timestamp
}

func (s *signer) signWithUnix11Ts(method string, action string, body string) (string, string) {
	timestamp := strconv.FormatInt(s.serverTimeUnix11(), 10)
	return s.shar256(timestamp, method, action, body), timestamp
}

func (s *signer) getHttpHeaderWithSign(method string, action string, body string) map[string]string {
	sign, timestamp := s.signWithIsoTs(method, action, body)

	headers := map[string]string{}
	headers["OK-ACCESS-KEY"] = s.key
	headers["OK-ACCESS-SIGN"] = sign
	headers["OK-ACCESS-TIMESTAMP"] = timestamp
	headers["OK-ACCESS-PASSPHRASE"] = s.pass
	headers["Content-Type"] = "application/json"

	return headers
}
