/*
- @Author: aztec
- @Date: 2024-04-12 12:19:16
- @Description: 通用协议定义
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package webservice

import (
	"fmt"
	"io"
	"net/http"

	"github.com/aztecqt/dagger/util"
)

type WebpHead struct {
	Ok  bool   `json:"ok"`
	Msg string `json:"msg"`
}

func WebpHeadSuccess(format string, params ...interface{}) WebpHead {
	msg := fmt.Sprintf(format, params...)
	return WebpHead{Ok: true, Msg: msg}
}

func WebpHeadFailed(msg string) WebpHead {
	return WebpHead{Msg: msg}
}

func WriteError(w http.ResponseWriter, msg string) {
	io.WriteString(w, util.Object2String(WebpHeadFailed(msg)))
}
