/*
* @Author: aztec
* @Date: 2022-05-27 15:23:00
  - @LastEditors: Please set LastEditors

* @FilePath: \stratergyc:\svn\quant\go\src\dagger\api\coingeckoapi\rest.go
* @Description: coingecko获取价格
*
* Copyright (c) 2022 by aztec, All Rights Reserved.
*/
package coingeckoapi

import (
	"net/url"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/network"
)

const restRootURL = "https://api.coingecko.com/api/v3"
const restLogPrefix = "coingecko-rest"
const logPrefix = "coingecko"

// 获取所有支持的币种列表
type CoinIdSymbol struct {
	Id     string `json:"id"`
	Symbol string `json:"symbol"`
}

func GetCoinList(withCache bool) (*[]CoinIdSymbol, error) {
	action := "/coins/list"
	method := "GET"
	params := url.Values{}
	params.Set("include_platform", "false")
	action = action + "?" + params.Encode()
	url := restRootURL + action
	resp, err := network.ParseHttpResult[[]CoinIdSymbol](logPrefix, "GetCoinList", url, method, "", nil, nil, nil)
	return resp, err
}

// 获取指定币种的概要数据（价格、市值、交易量），以USD为对手币
type SimplePriceInfo struct {
	Price     float64
	Marketcap float64
	Volume24h float64
}

func GetSimplePriceInfo(coinIds []string) (map[string]SimplePriceInfo, error) {
	action := "/simple/price"
	method := "GET"
	params := url.Values{}
	params.Set("ids", strings.Join(coinIds, ","))
	params.Set("vs_currencies", "usd")
	params.Set("include_market_cap", "true")
	params.Set("include_24hr_vol", "true")
	action = action + "?" + params.Encode()
	url := restRootURL + action
	resp, err := network.ParseHttpResult[map[string]map[string]float64](logPrefix, "SimplePrice", url, method, "", nil, nil, nil)
	if err != nil {
		return nil, err
	} else {
		priceInfos := map[string]SimplePriceInfo{}
		for coinId, vals := range *resp {
			pi := SimplePriceInfo{}
			if v, ok := vals["usd"]; ok {
				pi.Price = v
			}

			if v, ok := vals["usd_market_cap"]; ok {
				pi.Marketcap = v
			}

			if v, ok := vals["usd_24h_vol"]; ok {
				pi.Volume24h = v
			}

			priceInfos[coinId] = pi
		}
		return priceInfos, nil
	}
}

// 根据symbol查询币种概要信息(btc,eth...)
// 注意symbol不可以直接用于查询，需要转换成coinID
// 而同一个symbol可能对应多个coinId，这里取市值最大的一个为准
func GetSimplePriceInfoBySymbol(symbols []string) (map[string]SimplePriceInfo, error) {
	if ciss, err := GetCoinList(true); err == nil {
		// 选出要查询的id
		id2Symbol := map[string]string{}
		ids := []string{}
		for _, cis := range *ciss {
			for _, symbol := range symbols {
				if cis.Symbol == symbol {
					id2Symbol[cis.Id] = symbol
					ids = append(ids, cis.Id)
				}
			}
		}

		// 因为不能一次查询太多，所以将ids分一下组
		maxSymbolLength := 4096
		curSymbolLength := 0
		idsGroups := [][]string{}
		curGroup := []string{}
		for _, id := range ids {
			curSymbolLength += len(id)
			curGroup = append(curGroup, id)
			if curSymbolLength >= maxSymbolLength {
				idsGroups = append(idsGroups, curGroup)
				curGroup = []string{}
				curSymbolLength = 0
			}
		}

		if len(curGroup) > 0 {
			idsGroups = append(idsGroups, curGroup)
		}

		// 查询基础信息，填充到结果中。symbol重复时，以市值较大的为准
		symbol2Pi := map[string]SimplePriceInfo{}
		for i, ids := range idsGroups {
			if pis, err := GetSimplePriceInfo(ids); err == nil {
				for id, spi := range pis {
					if symbol, ok := id2Symbol[id]; ok {
						if spiOrign, ok := symbol2Pi[symbol]; ok {
							if spi.Marketcap > spiOrign.Marketcap {
								symbol2Pi[symbol] = spi
							}
						} else {
							symbol2Pi[symbol] = spi
						}
					}
				}
			} else {
				return nil, err
			}

			if i < len(idsGroups)-1 {
				time.Sleep(time.Second)
			}
		}

		return symbol2Pi, nil
	} else {
		return nil, err
	}
}

func SimplePrice(coinId string) (price float64, err error) {
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
