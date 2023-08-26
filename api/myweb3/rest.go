/*
 * @Author: aztec
 * @Date: 2023-05-08
 * @Description: 我自己定义的基于web3的接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package myweb3

import (
	"fmt"

	"aztecqt/dagger/util/network"
)

const logPrefix = "myweb3"

type Api struct {
	rootUrl string
}

func (a *Api) Init(rootUrl string) {
	a.rootUrl = rootUrl
}

func (a *Api) GetBalanceByAddress(networkid, addr string) (*GetBalanceResponse, error) {
	url := fmt.Sprintf("%s/balance?addr=%s&network=%s", a.rootUrl, addr, networkid)
	return network.ParseHttpResult[GetBalanceResponse](logPrefix, "myweb3_getBalance", url, "GET", "", nil, nil, nil)
}
