/*
 * @Author: aztec
 * @Date: 2022-10-27 09:04:53
 * @Description: etherscan的api
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package etherscanapi

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"aztecqt/dagger/util"
	"aztecqt/dagger/util/logger"
	"aztecqt/dagger/util/network"
	"github.com/shopspring/decimal"
)

const logPrefix = "etherscanapi"

type Network int

const (
	Network_EthMain Network = iota
	Network_Goerli
	Network_Sepolia
	Network_BSC
)

type Api struct {
	kg *util.KeyGroup
}

func (a *Api) Init(keys ...string) {
	a.kg = new(util.KeyGroup)
	a.kg.Init(200)
	for _, v := range keys {
		a.kg.AddKey(v)
	}
}

func (a *Api) rootUrl(networkid Network) string {
	switch networkid {
	case Network_EthMain:
		return "https://api.etherscan.io/api"
	case Network_Goerli:
		return "https://api-goerli.etherscan.io/api"
	case Network_Sepolia:
		return "https://api-sepolia.etherscan.io/api"
	case Network_BSC:
		return "https://api.bscscan.com/api"
	default:
		panic("invalid network id")
	}
}

func (a *Api) GetBlockNumber(networkid Network) uint64 {
	apikey := a.kg.GetKey()
	params := url.Values{}
	params.Set("module", "proxy")
	params.Set("action", "eth_blockNumber")
	params.Set("apikey", apikey)
	url := a.rootUrl(networkid) + "?" + params.Encode()
	resp, err := network.ParseHttpResult[EthBlockNumberResp](logPrefix, "etherscan_getBlockNumber", url, "GET", "", nil, nil, nil)
	if err == nil {
		blockno, ok := util.HexString2UInt64(resp.Result)
		if ok {
			return blockno
		} else {
			logger.LogImportant(logPrefix, "can't parse `%s` to int for block number", resp.Result)
			return 0
		}
	} else {
		logger.LogImportant(logPrefix, "error in eth_blockNumber: %s", err.Error())
		return 0
	}
}

func (a *Api) GetBlockByNumber(networkid Network, blockNumber uint64) (*EthGetBlockByNumberResp, error) {
	apikey := a.kg.GetKey()
	params := url.Values{}
	params.Set("module", "proxy")
	params.Set("action", "eth_getBlockByNumber")
	params.Set("tag", "0x"+strconv.FormatInt(int64(blockNumber), 16))
	params.Set("boolean", "true")
	params.Set("apikey", apikey)
	url := a.rootUrl(networkid) + "?" + params.Encode()
	resp, err := network.ParseHttpResult[EthGetBlockByNumberResp](logPrefix, "etherscan_getBlockByNumber", url, "GET", "", nil, nil, nil)
	if len(resp.Result.TimeStamp) > 0 && len(resp.Result.Number) > 0 {
		return resp, err
	} else {
		return nil, errors.New("invalid block")
	}
}

func (a *Api) GetBalanceByAddress(networkid Network, addr string) (decimal.Decimal, error) {
	apikey := a.kg.GetKey()
	url := fmt.Sprintf("%s?module=account&action=balance&address=%s&tag=latest&apikey=%s", a.rootUrl(networkid), addr, apikey)
	resp, err := network.ParseHttpResult[GetBalanceResp](logPrefix, "etherscan_getBalance", url, "GET", "", nil, nil, nil)
	if err == nil {
		raw, rawok := util.String2Float64(resp.Result)
		if rawok {
			d := decimal.NewFromFloat(raw)
			d = d.Shift(-18)
			return d, nil
		} else {
			return decimal.Zero, errors.New("convert to int failed")
		}

	} else {
		return decimal.Zero, err
	}
}
