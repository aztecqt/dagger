/*
* @Author: aztec
* @Date: 2022-03-25 16:39:00
  - @LastEditors: Please set LastEditors

* @Description: fasthttp封装了一下，方便调用
*
* Copyright (c) 2022 by aztec, All Rights Reserved.
*/
package network

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

var cookies []http.Cookie

func AddCookie(cookie http.Cookie) {
	cookies = append(cookies, cookie)
}

func SetCookies(v []http.Cookie) {
	cookies = v
}

func ClearCoockies() {
	cookies = make([]http.Cookie, 0)
}

func HttpCall(url string, method string, postData string, headers map[string]string, callback func(*http.Response, error)) {
	logPrefix := "http"
	if callback == nil {
		logger.LogPanic(logPrefix, "no callback, url=%s", url)
	}

	req, err := http.NewRequest(method, url, strings.NewReader(postData))
	if err != nil {
		callback(nil, err)
		return
	}

	for _, c := range cookies {
		req.AddCookie(&c)
	}

	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		callback(nil, err)
		return
	} else {
		defer res.Body.Close()
		callback(res, err)
	}
}

var EnableHttpLog = true

func ParseHttpResult[T any](logPref, funcName, url, method, postData string, headers map[string]string, cbRaw func(resp *http.Response, body []byte), cbErr func(e error)) (t *T, e error) {
	defer util.DefaultRecover()

	if method == "GET" {
		if EnableHttpLog {
			logger.LogDebug(logPref, "%s GET from url: %s", funcName, url)
		}
	} else {
		if EnableHttpLog {
			logger.LogDebug(logPref, "%s POST from url: %s, postData: %s", funcName, url, postData)
		}

		if headers != nil {
			headers["content-type"] = "application/json"
		}
	}

	HttpCall(url, method, postData, headers, func(resp *http.Response, err error) {
		t = new(T)
		var body []byte
		if err != nil {
			e = err
			logger.LogImportant(logPref, "%s http error, err=%s", funcName, err.Error())
		} else {
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				e = err
				logger.LogImportant(logPref, "read body error: %s", err.Error())
			} else {
				if EnableHttpLog {
					if len(body) > 4096 {
						logger.LogDebug(logPref, "%s resp: %s ...(+%d)", funcName, string(body[4096]), len(body)-4096)
					} else {
						logger.LogDebug(logPref, "%s resp: %s", funcName, string(body))
					}
				}

				err = json.Unmarshal(body, t)
				if err != nil {
					e = err
					logger.LogImportant(logPref, "%s json unmarshal error, err=%s", funcName, err.Error())
				}
			}
		}

		if cbRaw != nil {
			cbRaw(resp, body)
		}

		if e != nil && cbErr != nil {
			cbErr(e)
		}
	})

	return
}
