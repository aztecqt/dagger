/*
 * @Author: aztec
 * @Date: 2023-09-29 10:19:39
 * @Description: 结构体定义
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package follow

import (
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

type CommonRestResp struct {
	Code string `json:"code"`
}

type Trader struct {
	NickName string `json:"nickName"`
	TraderId string `json:"uniqueName"`
}

type TraderListResp struct {
	CommonRestResp
	Data []struct {
		Pages int      `json:"pages"`
		Ranks []Trader `json:"ranks"`
	} `json:"data"`
}

type PositionDetail struct {
	InstId          string          `json:"instId"`
	Size            decimal.Decimal `json:"availSubPos"`
	Side            string          `json:"posSide"`
	OpenAvgPrice    decimal.Decimal `json:"openAvgPx"`
	LatestPrice     decimal.Decimal `json:"last"`
	OpenTimeStamp   string          `json:"openTime"`
	UpdateTimeStamp string          `json:"uTime"`

	OpenTime   time.Time
	UpdateTime time.Time
}

func (p *PositionDetail) parse() {
	p.OpenTime = time.UnixMilli(util.String2Int64Panic(p.OpenTimeStamp))
	p.UpdateTime = time.UnixMilli(util.String2Int64Panic(p.UpdateTimeStamp))

	if p.Side == "short" {
		p.Size = p.Size.Neg()
	} else if p.Side != "long" && p.Side != "net" {
		logger.LogImportant(logPrefix, "invalid side: %s", p.Side)
	}

	if p.Size.IsZero() {
		logger.LogImportant(logPrefix, "invliad size: zero")
	}
}

type PositionDetailResp struct {
	CommonRestResp
	Data []PositionDetail `json:"data"`
}

func (p *PositionDetailResp) parse() {
	for i := range p.Data {
		p.Data[i].parse()
	}
}

type PositionHistory struct {
	InstId         string          `json:"instId"`
	Size           decimal.Decimal `json:"subPos"`
	Side           string          `json:"posSide"`
	OpenTimeStamp  string          `json:"openTime"`
	CloseTimeStamp string          `json:"uTime"`
	IdStr          string          `json:"id"`
	OpenAvgPrice   decimal.Decimal `json:"openAvgPx"`
	CloseAvgPrice  decimal.Decimal `json:"closeAvgPx"`
	Id             int64
	OpenTime       time.Time
	CloseTime      time.Time
}

func (p *PositionHistory) parse() {
	p.Id = util.String2Int64Panic(p.IdStr)
	p.OpenTime = time.UnixMilli(util.String2Int64Panic(p.OpenTimeStamp))
	p.CloseTime = time.UnixMilli(util.String2Int64Panic(p.CloseTimeStamp))

	if p.Side == "short" {
		p.Size = p.Size.Neg()
	} else if p.Side != "long" {
		logger.LogImportant(logPrefix, "invalid side: %s", p.Side)
	}

	if p.Size.IsZero() {
		logger.LogImportant(logPrefix, "invliad size: zero")
	}
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

type PositionDetailV2 struct {
	InstId        string          `json:"instId"`
	OpenTimeStamp string          `json:"cTime"`
	Side          string          `json:"posSide"`
	PosRatio      decimal.Decimal `json:"posSpace"`
	AvgPrice      decimal.Decimal `json:"avgPx"`

	OpenTime time.Time
}

func (p *PositionDetailV2) parse() {
	p.OpenTime = time.UnixMilli(util.String2Int64Panic(p.OpenTimeStamp))

	if p.Side == "short" {
		p.PosRatio = p.PosRatio.Neg()
	} else if p.Side != "long" && p.Side != "net" {
		logger.LogImportant(logPrefix, "invalid side: %s", p.Side)
	}

	if p.PosRatio.IsZero() {
		logger.LogImportant(logPrefix, "invliad size: zero")
	}
}

type PositionDetailRespV2 struct {
	CommonRestResp
	Data []struct {
		Detail []PositionDetailV2 `json:"posData"`
	} `json:"data"`
}

func (p *PositionDetailRespV2) parse() {
	for i := range p.Data {
		for i2 := range p.Data[i].Detail {
			p.Data[i].Detail[i2].parse()
		}
	}
}

type PositionHistoryV2 struct {
	InstId         string          `json:"instId"`
	ProfitRatio    decimal.Decimal `json:"closeUplRatio"`
	OpenTimeStamp  string          `json:"cTime"`
	CloseTimeStamp string          `json:"uTime"`
	IdStr          string          `json:"posId"`
	Id             int64
	OpenTime       time.Time
	CloseTime      time.Time
}

func (p *PositionHistoryV2) parse() {
	p.Id = util.String2Int64Panic(p.IdStr)
	p.OpenTime = time.UnixMilli(util.String2Int64Panic(p.OpenTimeStamp))
	p.CloseTime = time.UnixMilli(util.String2Int64Panic(p.CloseTimeStamp))
}

type PositionHistoryRespV2 struct {
	CommonRestResp
	Data []PositionHistoryV2 `json:"data"`
}

func (p *PositionHistoryRespV2) parse() {
	for i := range p.Data {
		p.Data[i].parse()
	}
}

type TradeRecord struct {
	InstId        string `json:"instId"`
	OpenTimeStamp string `json:"cTime"`
	PosSide       string `json:"posSide"`
	Side          string `json:"side"`
	OrderIdStr    string `json:"ordId"`
	OpenTime      time.Time
	OrderId       int64
}

func (t *TradeRecord) parse() {
	t.OpenTime = time.UnixMilli(util.String2Int64Panic(t.OpenTimeStamp))
	t.OrderId = util.String2Int64Panic(t.OrderIdStr)
}

type TradeRecordResp struct {
	CommonRestResp
	Data []TradeRecord `json:"data"`
}

func (t *TradeRecordResp) parse() {
	for i := range t.Data {
		t.Data[i].parse()
	}
}
