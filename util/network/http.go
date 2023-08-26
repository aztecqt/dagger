/*
 * @Author: aztec
 * @Date: 2022-03-25 16:39:00
 * @LastEditors: aztec
 * @Description: fasthttp封装了一下，方便调用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package network

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
	"github.com/valyala/fasthttp"
)

var fastHttpClient = &fasthttp.Client{
	Name:                "http-utils",
	MaxConnsPerHost:     512,
	MaxIdleConnDuration: 20 * time.Second,
	ReadTimeout:         10 * time.Second,
	WriteTimeout:        10 * time.Second,
}

// Deprecated: 即将删除
func FastHttpCall(url string, method string, postData string, headers map[string]string, callback func(*fasthttp.Response, error)) {
	var req *fasthttp.Request
	var resp *fasthttp.Response
	defer func() {
		fasthttp.ReleaseResponse(resp)
		fasthttp.ReleaseRequest(req)
	}()

	req = fasthttp.AcquireRequest()
	if headers == nil {
		headers = map[string]string{}
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.SetMethod(method)
	req.SetRequestURI(url)
	req.SetBodyString(postData)
	resp = fasthttp.AcquireResponse()

	err := fastHttpClient.Do(req, resp)
	if err != nil {
		logger.LogImportant("http", "err=%s\n url=%s\n method=%s\n post=%s\n", err.Error(), url, method, postData)
		callback(nil, err)
	} else {
		if resp.StatusCode() != 200 {
			logger.LogImportant("http", "rep status code=%d(not 200)\n rsp=%s\n url=%s\n method=%s\n post=%s\n", resp.StatusCode(), string(resp.Body()), url, method, postData)
			callback(resp, errors.New("response error"))
		} else {
			callback(resp, nil)
		}
	}
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
		headers["content-type"] = "application/json"
	}

	HttpCall(url, method, postData, headers, func(resp *http.Response, err error) {
		t = new(T)
		var body []byte
		if err != nil {
			e = err
			logger.LogImportant(logPref, "%s http error, err=%s", funcName, err.Error())
		} else {
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				e = err
				logger.LogImportant(logPref, "read body error: %s", err.Error())
			} else {
				if EnableHttpLog && len(body) < 4096 { // 避免日志太长
					logger.LogDebug(logPref, "%s resp(%p): %s", funcName, body, string(body))
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

func JsonHeaders() map[string]string {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json; charset=utf-8"
	return headers
}
