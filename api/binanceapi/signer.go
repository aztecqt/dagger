/*
 * @Author: aztec
 * @Date: 2022-03-26 10:22:43
 * @Description: binance消息签名器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binanceapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/url"
	"time"

	"github.com/aztecqt/dagger/util/logger"
)

type signer struct {
	key        string
	secret     string
	serverTsFn func() int64
}

var SignerIns *signer
var signerLogPrefix = "bn_signer"

var inited bool = false

func Init(key string, secret string, serverTsFn func() int64) {
	SignerIns = new(signer)
	SignerIns.key = key
	SignerIns.secret = secret
	SignerIns.serverTsFn = serverTsFn

	// 获取服务器时间跟本地时间的差
	for {
		serverTime := serverTsFn()
		if serverTime <= 0 {
			logger.LogImportant(signerLogPrefix, "get server time failed...retry after 1 second")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	inited = true
}

func HasKey() bool {
	return len(SignerIns.key) > 0 && len(SignerIns.secret) > 0
}

func getParamHmacSHA256Sign(message string, secretKey string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, err := mac.Write([]byte(message))
	if err != nil {
		return "", err
	}

	str := fmt.Sprintf("%x", (mac.Sum(nil)))
	//str := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	//str = url.QueryEscape(str)
	return str, nil
}

func (s *signer) Sign(param url.Values) (header map[string]string, paramStr string, err error) {
	// 需要签名的参数，都要包含这两个东西
	param.Set("timestamp", fmt.Sprintf("%d", s.serverTsFn()))
	param.Set("recvWindow", "10000")
	payload := param.Encode()

	signature, err := getParamHmacSHA256Sign(payload, s.secret)
	if err != nil {
		logger.LogPanic(signerLogPrefix, "sign error!")
		return
	}

	param.Set("signature", signature)
	paramStr = param.Encode()

	header = make(map[string]string)
	header["X-MBX-APIKEY"] = s.key
	return
}

func (s *signer) Sign2(param url.Values) (header map[string]string, paramStr string, err error) {
	// 需要签名的参数，都要包含这两个东西
	param.Set("timestamp", fmt.Sprintf("%d", s.serverTsFn()))
	param.Set("recvWindow", "10000")
	payload := param.Encode()

	signature, err := getParamHmacSHA256Sign(payload, s.secret)
	if err != nil {
		logger.LogPanic(signerLogPrefix, "sign error!")
		return
	}

	signature = "xxx"
	param.Set("signature", signature)
	paramStr = param.Encode()

	header = make(map[string]string)
	header["X-MBX-APIKEY"] = s.key
	return
}

func (s *signer) HeaderWithApiKey() map[string]string {
	header := make(map[string]string)
	header["X-MBX-APIKEY"] = s.key
	return header
}
