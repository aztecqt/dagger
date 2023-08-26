/*
 * @Author: aztec
 * @Date: 2022-05-27 15:23:00
 * @LastEditors: aztec
 * @FilePath: \stratergyc:\svn\quant\go\src\dagger\api\coingeckoapi\rest.go
 * @Description: coingecko获取价格
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package coingeckoapi

import (
	"errors"
	"net/url"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
	"aztecqt/dagger/util/network"
)

const restRootURL = "https://api.coingecko.com/api/v3"
const restLogPrefix = "coingecko-rest"
const logPrefix = "coingecko"

func SimplePrice(coinId string) (price float64, err error) {
	defer util.DefaultRecoverWithCallback(func(errstr string) {
		price = 0
		err = errors.New(errstr)
	})

	action := "/simple/price"
	method := "GET"
	params := url.Values{}
	params.Set("ids", coinId)
	params.Set("vs_currencies", "usd")
	action = action + "?" + params.Encode()
	url := restRootURL + action
	result, e := network.ParseHttpResult[map[string]map[string]float64](logPrefix, "SimplePrice", url, method, "", nil, nil, nil)
	if e != nil {
		err = e
		price = 0
	} else {
		price = (*result)[coinId]["usd"]
		logger.LogDebug(logPrefix, "price of %s = %v", coinId, price)
	}

	return
}
