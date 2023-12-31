/*
 * @Author: aztec
 * @Date: 2022-04-06 13:19:27
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-12-17 16:45:34
 * @FilePath: \stratergyc:\work\svn\quant\go\src\dagger\api\okexv5api\response_private.go
 * @Description:okex的api返回数据。不对外公开，仅在包内做临时传递数据用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package okexv5api

import (
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

// 查询账户配置
type accountConfigRestResp struct {
	Code string `json:"code"`
	Data []struct {
		UID      string `json:"uid"`
		AccLevel string `json:"acctLv"`
		PosMode  string `json:"posMode"`
		Level    string `json:"level"`
		LevelTmp string `json:"levelTmp"`
	} `json:"data"`
}

// 设置杠杆倍率返回
type SetLeverRateRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Lever   string `json:"lever"`
		MgnMode string `json:"mgnMode"`
		InstId  string `json:"instId"`
		PosSide string `json:"posSide"`
	} `json:"data"`
}

// 账户信息
type AccountBalanceResp struct {
	Details []struct {
		Currency string `json:"ccy"`
		UTime    string `json:"uTime"`
		Eq       string `json:"eq"`
		Frozen   string `json:"frozenBal"`
		CashBal  string `json:"cashBal"`
	} `json:"details"`
}

type AccountBalanceRestResp struct {
	CommonRestResp
	Data []AccountBalanceResp `json:"data"`
}

type AccountBalanceWsResp struct {
	Data []AccountBalanceResp `json:"data"`
}

// 账户信息（资金账户）
type AssetBalanceResp struct {
	Currency  string `json:"ccy"`
	Balance   string `json:"bal"`
	Frozen    string `json:"frozenBal"`
	Available string `json:"availBal"`
}

type AssetBalanceRestResp struct {
	CommonRestResp
	Data []AssetBalanceResp `json:"data"`
}

// 划转请求
type TransferReq struct {
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	From     string `json:"from"`
	To       string `json:"to"`
	Type     string `json:"type"`
	ClientId string `json:"clientId"`
}

// 划转结果
type TransferResp struct {
	TransId  string `json:"transId"`
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	From     string `json:"from"`
	To       string `json:"to"`
	ClientId string `json:"clientId"`
}

type TransferRestResp struct {
	CommonRestResp
	Data []AssetBalanceResp `json:"data"`
}

// 提币请求
type WithdrawReq struct {
	Ccy      string `json:"ccy"`
	Amount   string `json:"amt"`
	Dest     string `json:"dest"` // 3=内部转账，4=链上提币
	ToAddr   string `json:"toAddr"`
	Fee      string `json:"fee"`
	Chain    string `json:"chain"`
	AreaCode string `json:"areaCode"` // 内部转账填写
	ClientId string `json:"clientId"`
}

// 提币结果返回
type WithdrawResp struct {
	CommonRestResp
}

// 查询提币返回
/*-3：撤销中
-2：已撤销
-1：失败
0：等待提币
1：提币中
2：提币成功
7: 审核通过
10: 等待划转
4, 5, 6, 8, 9, 12: 等待客服审核*/
type WithdrawStatus struct {
	ClientId string `json:"clientId"`
	State    string `json:"state"`
}
type WithdrawHistoryResp struct {
	CommonRestResp
	Data []WithdrawStatus `json:"data"`
}

// 仓位
type PositionWsResp struct {
	Data []struct {
		InstType string `json:"instType"`
		MgnMode  string `json:"mgnMode"`
		PosSide  string `json:"posSide"`
		InstId   string `json:"instId"`
		TradeId  string `json:"tradeId"`
		UTime    string `json:"uTime"`
		Pos      string `json:"pos"`
		AvailPos string `json:"availPos"`
		AvgPx    string `json:"avgPx"`
		LiqPx    string `json:"liqPx"`
		MarkPx   string `json:"markPx"`
	} `json:"data"`
}

// #region 订单相关
const (
	OrderStatus_Born            = "born"
	OrderStatus_Alive           = "alive"
	OrderStatus_Canceled        = "canceled"
	OrderStatus_PartiallyFilled = "partially_filled"
	OrderStatus_Filled          = "filled"
)

// 下单请求
type MakeorderRestReq struct {
	InstId        string `json:"instId"`
	TradeMode     string `json:"tdMode"`  // isolated：逐仓 cross：全仓 cash：非保证金
	ClientOrderId string `json:"clOrdId"` //
	Tag           string `json:"tag"`
	Side          string `json:"side"`    // buy sell
	PosSide       string `json:"posSide"` // long short
	OrderType     string `json:"ordType"` // limit post_only
	ReduceOnly    bool   `json:"reduceOnly"`
	Price         string `json:"px"`
	Size          string `json:"sz"`
}

// 下单返回
type MakeorderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 撤单返回
type CancelOrderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 批量撤单请求单元
type CancelBatchOrderRestReq struct {
	InstId  string `json:"instId"`
	OrderId string `json:"ordId"`
}

// 修改订单返回
type AmendOrderRestResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		ClientOrderId string `json:"clOrdId"`
		OrderId       string `json:"ordId"`
		SCode         string `json:"sCode"`
		SMsg          string `json:"sMsg"`
	} `json:"data"`
}

// 查询订单
type OrderResp struct {
	InstId        string `json:"instId"`
	OrderId       string `json:"ordId"`
	ClientOrderId string `json:"clOrdId"`
	Tag           string `json:"tag"`
	Price         string `json:"px"`
	Size          string `json:"sz"`
	AccFillSize   string `json:"accFillSz"`
	AvgPrice      string `json:"avgPx"`
	Status        string `json:"state"` // alive/canceled/partially_filled/filled
	UTime         string `json:"uTime"`
}

type OrderRestResp struct {
	CommonRestResp
	Data      []OrderResp `json:"data"`
	LocalTime time.Time
}

type OrderWsResp struct {
	LocalTime time.Time
	Data      []OrderResp `json:"data"`
}

// 查询成交
type Fills struct {
	InstType    string          `json:"instType"`
	InstId      string          `json:"instId"`
	Price       decimal.Decimal `json:"fillPx"`
	Size        decimal.Decimal `json:"fillSz"`
	Side        string          `json:"side"`
	FillTimeStr string          `json:"fillTime"`
	FillTime    time.Time
}

func (f *Fills) Parse() {
	f.FillTime = time.UnixMilli(util.String2Int64Panic(f.FillTimeStr))
}

type FillsResp struct {
	CommonRestResp
	Data []Fills `json:"data"`
}

func (f *FillsResp) parse() {
	for i := range f.Data {
		f.Data[i].Parse()
	}
}

type PositionCloseType int

const (
	PositionCloseType_None         PositionCloseType = 0
	PositionCloseType_PartlyClosed PositionCloseType = 1
	PositionCloseType_AllClosed    PositionCloseType = 2
	PositionCloseType_ForceClosed  PositionCloseType = 3
	PositionCloseType_ForceReduce  PositionCloseType = 4
	PositionCloseType_ADL          PositionCloseType = 5
)

type PositionHistory struct {
	InstType        string          `json:"instType"`
	InstId          string          `json:"instId"`
	TypeStr         string          `json:"type"`
	CreateTimeStamp string          `json:"ctime"`
	UpdateTimeStamp string          `json:"utime"`
	OpenAvgPrice    decimal.Decimal `json:"openAvgPx"`
	CloseAvgPrice   decimal.Decimal `json:"closeAvgPx"`
	OpenMaxPos      decimal.Decimal `json:"openMaxPos"`
	CloseTotalPos   decimal.Decimal `json:"closeTotalPos"`
	Fee             decimal.Decimal `json:"fee"`
	FundingFee      decimal.Decimal `json:"fundingFee"`
	RealizedPnl     decimal.Decimal `json:"realizedPnl"` // 总收益=平仓收益+资金费+手续费
	Pnl             decimal.Decimal `json:"pnl"`         // 平仓收益
	PnlRatio        decimal.Decimal `json:"pnlRatio"`    // ???计算方法未知
	Lever           decimal.Decimal `json:"lever"`
	Direction       string          `json:"direction"`
	DepositCurrency string          `json:"ccy"`

	Type       PositionCloseType
	CreateTime time.Time
	UpdateTime time.Time
}

func (p *PositionHistory) parse() {
	p.Type = PositionCloseType(util.String2IntPanic(p.TypeStr))
	p.CreateTime = time.UnixMilli(util.String2Int64Panic(p.CreateTimeStamp))
	p.UpdateTime = time.UnixMilli(util.String2Int64Panic(p.UpdateTimeStamp))
}

type PositionHistoryResp struct {
	CommonRestResp
	Data []PositionHistory `json:"data"`
}

func (p *PositionHistoryResp) parse() {
	for i := range p.Data {
		p.Data[i].parse()
	}
}

// #endregion 订单相关
