/*
 * @Author: aztec
 * @Date: 2022-04-02 10:03:20
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-03-13 09:28:30
 * @FilePath: \dagger\cex\okexv5\defines.go
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package okexv5

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/api/okexv5api"
	"github.com/aztecqt/dagger/util"

	"github.com/shopspring/decimal"
)

// 交易所配置
type ExchangeConfig struct {
	// 是否从ticker来生成Depth数据。true则不订阅depth，而是ticker
	DepthFromTicker bool `json:"depth_from_ticker"`

	// 是否通过rest拉取ticker。是的话，由exchange统一拉取所有ticker，否则各个交易对自行订阅
	TickerFromRest bool `json:"ticker_from_rest"`

	// 是否订阅市场爆仓数据
	SubscribeLiquidationOrders bool `json:"sub_liq_orders"`

	// 是否订阅限价数据
	SubscribePriceLimit bool `json:"sub_price_limit"`

	// 是否订阅标记价格
	SubscribeMarkPrice bool `json:"sub_mark_price"`

	// 是否订阅资金费率
	SubscribeFundingFeeRate bool `json:"sub_ffr"`

	// 账号模式。见相应枚举
	AccLevel okexv5api.AccLevel `json:"acc_level"`

	// 现货/合约交易模式。cash/cross，isolated暂不支持
	SpotTradeMode     okexv5api.TradeMode `json:"spot_trade_mode"`
	ContractTradeMode okexv5api.TradeMode `json:"contract_trade_mode"`

	// 仓位模式。ok支持net_mode/long_short_mode
	// 由于两者都可以兼容，所以在配置文件里不做指定，而是记录交易所发过来的值
	PositionMode okexv5api.PositionMode

	// 费率观察器设置
	FundingFeeObserver struct {
		UsdtSwap bool `json:"usdt_swap"`
	} `json:"ff_obv"`
}

func newExchangeConfig() ExchangeConfig {
	cfg := ExchangeConfig{
		DepthFromTicker:   true,
		TickerFromRest:    false,
		AccLevel:          okexv5api.AccLevel_MultiCcy,
		SpotTradeMode:     "cash",
		ContractTradeMode: "cross",
	}
	return cfg
}

// 订单快照，用于订单刷新
type orderSnapshot struct {
	localTime  time.Time
	id         int64
	clientId   string
	tag        string
	price      decimal.Decimal
	size       decimal.Decimal
	filled     decimal.Decimal
	avgPrice   decimal.Decimal
	status     string
	updateTime time.Time
	source     string
}

func (os *orderSnapshot) Parse(resp okexv5api.OrderResp, source string) {
	os.source = source
	os.id = util.String2Int64Panic(resp.OrderId)
	os.clientId = resp.ClientOrderId
	os.tag = resp.Tag
	os.price = util.String2DecimalPanicUnless(resp.Price, "")
	os.size = util.String2DecimalPanic(resp.Size)
	os.filled = util.String2DecimalPanic(resp.AccFillSize)
	os.avgPrice = util.String2DecimalPanicUnless(resp.AvgPrice, "")
	os.status = resp.Status
	os.updateTime = util.ConvetUnix13StrToTimePanic(resp.UTime)
}

func (os *orderSnapshot) String() string {
	return fmt.Sprintf(
		"(from %s)[id:%d clientId:%s price:%v size:%v filled:%v avgPrice:%v status:%s uTime:%v]",
		os.source,
		os.id,
		os.clientId,
		os.price,
		os.size,
		os.filled,
		os.avgPrice,
		os.status,
		os.updateTime.UnixMilli())
}
