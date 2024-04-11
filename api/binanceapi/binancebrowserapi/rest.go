/*
- @Author: aztec
- @Date: 2024-02-22 17:40:43
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package binancebrowserapi

import (
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/crypto"
	"github.com/aztecqt/dagger/util/network"
	"github.com/google/uuid"
)

const rootUrl = "https://www.binance.com"
const restLogPrefix = "binance_bapi"

var rc *util.RedisClient
var headers map[string]string
var headersVer int64
var cookiesVer int64
var account string
var password string

func Init(redisAddr, redisPass string, redisDb int, acc, pwd string) {
	rc = new(util.RedisClient)
	rc.Init(redisAddr, redisPass, redisDb, false)
	account = acc
	password = pwd
}

func refreshCookiesAndHeaders() bool {
	if verstr, ok := rc.HGet("headersVer", account); ok {
		if ver, ok := util.String2Int64(verstr); ok {
			if ver != headersVer {
				// headers有新版本
				if strEncrypted, ok := rc.HGet("headers", account); ok {
					if bDecoded, err := hex.DecodeString(strEncrypted); err == nil {
						decrypted := crypto.DesCBCDecrypter(bDecoded, []byte(password), []byte(password))
						h := map[string]string{}
						if err := util.ObjectFromString(string(decrypted), &h); err == nil {
							headers = h
							headersVer = ver
						} else {
							fmt.Println(err.Error())
							return false
						}
					} else {
						fmt.Println(err.Error())
						return false
					}
				} else {
					return false
				}
			}
		}
	} else {
		return false
	}

	if verstr, ok := rc.HGet("cookiesVer", account); ok {
		if ver, ok := util.String2Int64(verstr); ok {
			if ver != cookiesVer {
				// cookies有新版本
				if strEncrypted, ok := rc.HGet("cookies", account); ok {
					if bDecoded, err := hex.DecodeString(strEncrypted); err == nil {
						decrypted := crypto.DesCBCDecrypter(bDecoded, []byte(password), []byte(password))
						pcookies := []network.PuppeteerCookie{}
						if err := util.ObjectFromString(string(decrypted), &pcookies); err == nil {
							cookies := []http.Cookie{}
							for _, pc := range pcookies {
								cookies = append(cookies, pc.ToCookie())
							}
							cookiesVer = ver
							network.SetCookies(cookies)
						} else {
							fmt.Println(err.Error())
							return false
						}
					} else {
						fmt.Println(err.Error())
						return false
					}
				} else {
					return false
				}
			}
		}
	} else {
		return false
	}

	return true
}

// 生成header
func genNewHeader(refererUrl string) map[string]string {
	h := maps.Clone(headers)
	uuid := uuid.New().String()
	h["Referer"] = refererUrl
	h["X-Trace-Id"] = uuid
	h["X-Ui-Request-Trace"] = uuid
	h["Accept-Encoding"] = ""
	return h
}

// 认证（可以用来测试api是否可以工作）
func Auth() (*ErrorMessage, error) {
	action := "/bapi/accounts/v1/public/authcenter/auth"
	method := "POST"
	ep := fmt.Sprintf("%s%s", rootUrl, action)

	if !refreshCookiesAndHeaders() {
		return nil, errors.New("refresh cookies/headers failed")
	}

	headers := genNewHeader("https://www.binance.com/zh-CN/my/dashboard")
	resp, err := network.ParseHttpResult[ErrorMessage](
		restLogPrefix,
		"Auth",
		ep,
		method,
		"",
		headers,
		nil, nil)

	return resp, err
}

// 杠杆下单
func PlaceMarginOrder(req PlaceMarginOrderReq) (*PlaceMarginOrderResp, error) {
	action := "/bapi/margin/v1/private/margin/place-order"
	method := "POST"
	ep := fmt.Sprintf("%s%s", rootUrl, action)

	headers := genNewHeader(fmt.Sprintf("https://www.binance.com/zh-CN/trade/%s?type=cross", req.Symbol))

	resp, err := network.ParseHttpResult[PlaceMarginOrderResp](
		restLogPrefix,
		"PlaceMarginOrder",
		ep,
		method,
		util.Object2String(req),
		headers,
		nil, nil)

	return resp, err
}

// 杠杆撤销订单
func CancelMarginOrder(symbol string, orderIds ...int64) (*CancelMarginOrderResp, error) {
	action := "/bapi/margin/v1/private/margin/cancel-order"
	method := "POST"
	ep := fmt.Sprintf("%s%s", rootUrl, action)

	headers := genNewHeader(fmt.Sprintf("https://www.binance.com/zh-CN/trade/%s?type=cross", symbol))

	req := CancelMarginOrderReq{}
	req.Symbols = append(req.Symbols, symbol)
	req.OrderIds = append(req.OrderIds, orderIds...)

	resp, err := network.ParseHttpResult[CancelMarginOrderResp](
		restLogPrefix,
		"CancelMarginOrder",
		ep,
		method,
		util.Object2String(req),
		headers,
		nil, nil)

	return resp, err
}

// 杠杆撤销全部订单
func CancelAllMarginOrders() (*CancelMarginOrderResp, error) {
	action := "/bapi/margin/v1/private/margin/cancel-all-order"
	method := "POST"
	ep := fmt.Sprintf("%s%s", rootUrl, action)

	headers := genNewHeader(fmt.Sprintf("https://www.binance.com/zh-CN/trade/%s?type=cross", "BTC_USDT"))

	resp, err := network.ParseHttpResult[CancelMarginOrderResp](
		restLogPrefix,
		"CancelAllMarginOrders",
		ep,
		method,
		"{}",
		headers,
		nil, nil)

	return resp, err
}
