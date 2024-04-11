/*
 * @Author: aztec
 * @Date: 2022-04-01 13:56:29
 * @Description: 其他各种数据定义
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package common

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceRecord struct {
	Price decimal.Decimal
	Time  time.Time
}

type KUnit struct {
	Time         time.Time
	OpenPrice    decimal.Decimal
	ClosePrice   decimal.Decimal
	HighestPrice decimal.Decimal
	LowestPrice  decimal.Decimal
	VolumeUSD    decimal.Decimal // 以USD计算的交易量
}

type ContractType string

const (
	ContractType_UsdSwap  ContractType = "usd_swap"
	ContractType_UsdtSwap              = "usdt_swap"
)

type TickSizeMode int

const (
	TickSizeMode_Unknown  TickSizeMode = 0 // 还未计算
	TickSizeMode_Standard TickSizeMode = 1 // 标准模式，如0.01, 0.0001之类
	TickSizeMode_Any      TickSizeMode = 2 // 非标准模式，如0.5， 0.025之类
)

type Instruments struct {
	Id             string
	BaseCcy        string          // 币币中的交易货币币种，如BTC-USDT中的BTC
	QuoteCcy       string          // 币币中的计价货币币种，如BTC-USDT中的USDT
	CtSymbol       string          // 表示是那个币种的合约（btc_usdt_swap就是btc，btc_usd_swap也是btc）
	CtType         ContractType    // 合约类型 枚举：ContractType
	IsUsdtContract bool            // 是否为U本位合约
	CtSettleCcy    string          // 盈亏结算和保证金币种（btc_usdt_swap是usdt，btc_usd_swap是btc）
	CtValCcy       string          // 合约面值计价币种（btc_usdt_swap是btc，btc_usd_swap是usdt）
	CtVal          decimal.Decimal // 合约面值
	ExpTime        time.Time       // 交割日期（交割合约、期权）
	Lever          int             // 最大杠杆倍率
	TickSize       decimal.Decimal // 下单价格精度
	TickSizeMode   TickSizeMode    // 精度模式
	LotSize        decimal.Decimal // 下单数量精度
	MinSize        decimal.Decimal // 最小下单数量
	MinValue       decimal.Decimal // 最小下单价值
}

func (i *Instruments) refreshTickSizeMode() {
	if i.TickSizeMode == TickSizeMode_Unknown {
		str := i.TickSize.String()
		c := 0
		for _, r := range str {
			if r == '1' {
				c++
			}
		}

		if c == 1 {
			i.TickSizeMode = TickSizeMode_Standard
		} else {
			i.TickSizeMode = TickSizeMode_Any
		}
	}
}
