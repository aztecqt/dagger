/*
 * @Author: aztec
 * @Date: 2023-01-02 11:56:44
 * @Description: 合约行情观察器的实现
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package okexv5

import (
	"fmt"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/shopspring/decimal"
)

type ContractObserver struct {
	logPrefix     string
	okxInstType   string
	contractInfos map[string]*common.ContractInfo
	currencys     []string
	contractType  string
}

func (c *ContractObserver) Currencys() []string {
	return c.currencys
}

func (c *ContractObserver) GetContractInfo(ccy string) *common.ContractInfo {
	if v, ok := c.contractInfos[ccy]; ok {
		return v
	} else {
		return nil
	}
}

func (c *ContractObserver) Init(contractType string) {
	c.logPrefix = fmt.Sprintf("%s-ContractObserver-%s", logPrefix, contractType)
	c.contractType = contractType
	c.okxInstType = ContractType2OkxInstType(contractType)

	// 初始化所有合约类型
	c.contractInfos = make(map[string]*common.ContractInfo)
	if resp, err := okexv5api.GetInstruments(c.okxInstType); err != nil {
		logger.LogPanic(c.logPrefix, "get instruments failed: %s", err.Error())
	} else {
		for _, v := range resp.Data {
			if contractType == "usd_swap" && !strings.Contains(v.InstID, "USD-SWAP") ||
				contractType == "usdt_swap" && strings.Contains(v.InstID, "USDT-SWAP") {
				ctinfo := new(common.ContractInfo)
				ctinfo.ValueAmount, _ = util.String2Decimal(v.CtVal)
				ctinfo.ValueCurrency = v.CtValCcy
				ctinfo.LatestPrice = decimal.Zero
				ctinfo.Depth = common.Orderbook{}
				ctinfo.Depth.Init()
				c.contractInfos[InstId2Symbol(v.InstID)] = ctinfo
				c.currencys = append(c.currencys, InstId2Symbol(v.InstID))
			}
		}
	}

	// 启动循环
	go c.updateTicker()
	go c.updateDepth()
}

// 每秒刷新一次
func (c *ContractObserver) updateTicker() {
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		<-ticker.C
		resp, err := okexv5api.GetTickers(c.okxInstType)
		if err != nil {
			logger.LogImportant(c.logPrefix, "get ticker failed: %s", err.Error())
		} else {
			for _, tr := range resp.Data {
				if v, ok := c.contractInfos[InstId2Symbol(tr.InstId)]; ok {
					v.LatestPrice = util.String2DecimalPanic(tr.Last)
				}
			}
		}
	}
}

// 定频逐个轮询，慢点没关系
func (c *ContractObserver) updateDepth() {
	for {
		for ccy, ci := range c.contractInfos {
			instId := CCyCttypeToInstId(ccy, c.contractType)
			if resp, err := okexv5api.GetDepth(instId, 25); err != nil {
				logger.LogImportant(c.logPrefix, "get depth failed: %s", err.Error())
			} else {
				if len(resp.Data) > 0 {
					// 更新depth
					d := resp.Data[0]
					asks := make([]decimal.Decimal, 0, len(d.Asks))
					bids := make([]decimal.Decimal, 0, len(d.Bids))
					for _, depthUnit := range d.Asks {
						price := util.String2DecimalPanic(depthUnit[0])
						amount := util.String2DecimalPanic(depthUnit[1])
						asks = append(asks, price, amount)
					}

					for _, depthUnit := range d.Bids {
						price := util.String2DecimalPanic(depthUnit[0])
						amount := util.String2DecimalPanic(depthUnit[1])
						bids = append(bids, price, amount)
					}
					ci.Depth.Rebuild(asks, bids)
				}
			}

			time.Sleep(time.Millisecond * 500)
		}
	}
}
