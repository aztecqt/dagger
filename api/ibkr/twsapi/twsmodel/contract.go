/*
- @Author: aztec
- @Date: 2024-03-01 12:13:14
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import (
	"slices"
	"strings"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type Contract struct {
	ConId                        int      // 数字id
	Symbol                       string   // 名字
	SecType                      string   // 类型
	LastTradeDateOrContractMonth string   // 交割日期，格式为"YYYYMM"（合约月份）或者YYYYMMDD（最后交易日）
	Strike                       float64  // for期权
	Right                        string   // for期权 P/PUT/C/CALL
	Multiplier                   string   // 不懂
	Exchange                     string   // 所在交易所
	Currency                     string   // ??
	LocalSymbol                  string   // ??
	PrimaryExch                  string   // 主要交易所??
	TradingClass                 string   // ??
	IncludeExpired               bool     // ??
	SecIdType                    string   // ??
	SecId                        string   // ??
	Description                  string   // 描述
	IssuerId                     string   // ??
	ComboLegsDescription         string   // ??
	ComboLegs                    []string // ??
}

func (c Contract) ToParamArray() []interface{} {
	return []interface{}{
		c.ConId,
		c.Symbol,
		c.SecType,
		c.LastTradeDateOrContractMonth,
		c.Strike,
		c.Right,
		c.Multiplier,
		c.Exchange,
		c.PrimaryExch,
		c.Currency,
		c.LocalSymbol,
		c.TradingClass,
	}
}

type TagValue struct {
	Tag   string
	Value string
}

type ContractDetail struct {
	Contract               Contract
	MarketName             string          // 跟exchange有啥区别？
	MinTick                decimal.Decimal // 价格精度
	PriceMagnifier         int             // ??
	OrderTypes             string          // 支持的订单类型
	ValidExchanges         string          // 支持的交易所
	MarketRuleIds          string          // 市场规则列表，跟ValidExchange对应
	UnderConId             int             // 对于衍生品，应该指其基础合约ID
	UnderSymbol            string          //
	UnderSecType           string          //
	LongName               string          // 描述名?
	ContractMonth          string          // 是几月份的合约（对于合约来说）
	Industry               string          // 应该是行业
	Category               string          // 应该是行业细分
	Subcategory            string          // 同上
	TimeZoneId             string          // 交易时区
	TradingHours           string          // 交易时间。如20180323:0400-20180323:2000;20180326:0400-20180326:2000
	LiquidHours            string          // 高流动时间。如20180323:0930-20180323:1600;20180326:0930-20180326:1600
	EvRule                 string          // ??
	EvMultiplier           float64         // ??
	AggGroup               int             // 聚合分组??
	SecIdList              []TagValue      // ??
	RealExpirationDate     string          // 实际有效期限
	LastTradeTime          string          // 最后交易时间?
	StockType              string          //
	Cusip                  string          // 债券的一个什么9个字符id
	Ratings                string          // 信用评级（债券）
	DescAppend             string          // 额外描述。债券专用
	BondType               string          // 债券类型
	Notes                  string          // 债券专用
	MinSize                decimal.Decimal // 最小订单数量
	SizeIncrement          decimal.Decimal // 最小订单增量
	SuggestedSizeIncrement decimal.Decimal // 哈?
}

func (c ContractDetail) GetRuleOfExchange(ex string) int {
	exs := strings.Split(c.ValidExchanges, ",")
	if i := slices.Index(exs, ex); i >= 0 {
		ruleIds := strings.Split(c.MarketRuleIds, ",")
		if i < len(ruleIds) {
			if id, ok := util.String2Int(ruleIds[i]); ok {
				return id
			}
		}
	}

	return -1
}
