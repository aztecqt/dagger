/*
 * @Author: aztec
 * @Date: 2022-05-30 10:56
  - @LastEditors: Please set LastEditors
  * @FilePath: \dagger\util\apikey\protocol.go
 * @Description: 协议定义
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package apikey

import (
	"encoding/json"

	"github.com/aztecqt/dagger/util/udpsocket"
)

const opGetKeyReq = "get_key_req"
const opGetKeyAck = "get_key_ack"
const opReleaseKeyReq = "release_key_req"
const opKeepKeyReq = "keep_key_req"
const opKeepKeyAck = "keep_key_ack"

type getKeyReq struct {
	udpsocket.Header
	Exchange string `json:"ex"`
	Account  string `json:"acc"`
	User     string `json:"user"`
	Share    bool   `json:"share"`
}

type getKeyAck struct {
	udpsocket.Header
	Result   bool   `json:"rst"`
	Message  string `json:"msg"`
	Key      string `json:"key"`
	Secret   string `json:"secret"`
	Password string `json:"password"`
}

type releaseKeyReq struct {
	udpsocket.Header
	User string `json:"user"`
}

type keepKeyReq struct {
	udpsocket.Header
	Exchange     string `json:"ex"`
	Account      string `json:"acc"`
	User         string `json:"user"`
	KeyEncrypted string `json:"key"`
	Share        bool   `json:"share"`
}

type keepKeyAck struct {
	udpsocket.Header
	Result  bool   `json:"rst"`
	Message string `json:"msg"`
	Key     string `json:"key"`
}

func newGetKeyReq(acc, ex, user string, share bool) string {
	req := getKeyReq{}
	req.OP = opGetKeyReq
	req.Account = acc
	req.Exchange = ex
	req.User = user
	req.Share = share
	b, err := json.Marshal(req)
	if err == nil {
		return string(b)
	} else {
		return ""
	}
}

func newReleaseKeyReq(user string) string {
	req := releaseKeyReq{}
	req.OP = opReleaseKeyReq
	req.User = user
	b, err := json.Marshal(req)
	if err == nil {
		return string(b)
	} else {
		return ""
	}
}

func newKeepKeyReq(acc, ex, user, keyenc string, share bool) string {
	req := keepKeyReq{}
	req.OP = opKeepKeyReq
	req.Account = acc
	req.Exchange = ex
	req.User = user
	req.Share = share
	req.KeyEncrypted = keyenc
	b, err := json.Marshal(req)
	if err == nil {
		return string(b)
	} else {
		return ""
	}
}
