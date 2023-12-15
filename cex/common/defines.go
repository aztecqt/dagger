/*
 * @Author: aztec
 * @Date: 2022-04-01 13:56:29
 * @Description: 其他各种数据定义
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package common

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/util"
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
	ContractType_UsdSwap     ContractType = "usd_swap"
	ContractType_UsdtSwap    ContractType = "usdt_swap"
	ContractType_ThisWeek    ContractType = "this_week"
	ContractType_NextWeek    ContractType = "next_week"
	ContractType_Quarter     ContractType = "quarter"
	ContractType_NextQuarter ContractType = "next_quarter"
)

type Instruments struct {
	Id          string
	BaseCcy     string          // 币币中的交易货币币种，如BTC-USDT中的BTC
	QuoteCcy    string          // 币币中的计价货币币种，如BTC-USDT中的USDT
	CtSymbol    string          // 表示是那个币种的合约（btc_usdt_swap就是btc，btc_usd_swap也是btc）
	CtType      string          // 合约类型 枚举：ContractType
	CtSettleCcy string          // 盈亏结算和保证金币种（btc_usdt_swap是usdt，btc_usd_swap是btc）
	CtValCcy    string          // 合约面值计价币种（btc_usdt_swap是btc，btc_usd_swap是usdt）
	CtVal       decimal.Decimal // 合约面值
	ExpTime     time.Time       // 交割日期（交割合约、期权）
	Lever       int             // 最大杠杆倍率
	TickSize    decimal.Decimal // 下单价格精度
	LotSize     decimal.Decimal // 下单数量精度
	MinSize     decimal.Decimal // 最小下单数量
	MinValue    decimal.Decimal // 最小下单价值
}

type estimateValueUnit struct {
	v decimal.Decimal
	t time.Time
}

type estimateValue struct {
	slc  []estimateValueUnit
	val  decimal.Decimal
	size int
	mu   sync.Mutex
}

func (e *estimateValue) RecordValue(v decimal.Decimal, t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ev := estimateValueUnit{v: v, t: t}
	e.slc = append(e.slc, ev)
	e.val = e.val.Add(v)
	e.size = len(e.slc)
}

func (e *estimateValue) ClearTill(till time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := 0; i < len(e.slc); {
		ms := e.slc[i].t.UnixMilli()
		tillMs := till.UnixMilli()
		nowMs := time.Now().UnixMilli()
		if tillMs >= ms || nowMs-ms > 1000*10 /*最多等10秒，10秒后还没有被刷新则强制丢弃*/ {
			e.val = e.val.Sub(e.slc[i].v)
			e.slc = util.SliceRemoveAt(e.slc, i)
		} else {
			i++
		}
	}

	if len(e.slc) == 0 {
		e.val = decimal.Zero
	}

	e.size = len(e.slc)
}

func (e *estimateValue) String() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var bb bytes.Buffer
	bb.WriteString("[")
	for _, ev := range e.slc {
		bb.WriteString(fmt.Sprintf("%v@%d ", ev.v, ev.t.UnixMilli()))
	}
	bb.WriteString("]")
	return bb.String()
}

type BalanceImpl struct {
	ccy    string
	rights decimal.Decimal // 权益（推送）
	total  decimal.Decimal // 权益（总计）
	temp   estimateValue   // 权益（预估）
	mu     sync.Mutex

	frozen       decimal.Decimal // 冻结权益
	updateTime   time.Time       // 余额刷新时间
	lastTempTime time.Time
}

func NewBalanceImpl(ccy string) *BalanceImpl {
	b := new(BalanceImpl)
	b.ccy = ccy
	return b
}

// 记录一项临时权益增减
func (b *BalanceImpl) RecordTempRights(r decimal.Decimal, t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.temp.RecordValue(r, t)
	b.total = b.rights.Add(b.temp.val)
	b.lastTempTime = t
	logger.LogDebug(b.ccy, "rights:%v, frozen:%v, temp:%v, tempDetail:%v", b.rights, b.frozen, b.temp.val, b.temp.String())
}

// 刷新权益，同时清理预估数据
func (b *BalanceImpl) RefreshRights(rights, frozen decimal.Decimal, tm time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	rightsOrign := b.rights
	tempOrign := b.temp.val
	b.temp.ClearTill(tm)
	b.rights = rights
	b.frozen = frozen
	b.total = b.rights.Add(b.temp.val)
	b.updateTime = tm
	accurate := b.rights.Sub(rightsOrign).Sub(tempOrign.Sub(b.temp.val))
	logger.LogDebug(b.ccy, "rights:%v, frozen:%v, temp:%v, tempDetail:%v, accurate:%v", b.rights, b.frozen, b.temp.val, b.temp.String(), accurate)
}

func (b *BalanceImpl) Rights() decimal.Decimal {
	return b.total
}

func (b *BalanceImpl) Frozen() decimal.Decimal {
	return b.frozen
}

func (b *BalanceImpl) Available() decimal.Decimal {
	return b.total.Sub(b.frozen)
}

func (b *BalanceImpl) Ready() bool {
	// 如果临时权益迟迟未被清除，则认为权益无效
	if b.temp.size > 0 && time.Now().UnixMilli()-b.lastTempTime.UnixMilli() > 5000 {
		return false
	}

	return true
}

type PositionImpl struct {
	instId     string
	_logPrefix string

	longAmount         decimal.Decimal // 多仓（推送）
	longTotal          decimal.Decimal // 多仓（总计）
	longTemp           estimateValue   // 多仓（预估）
	longAvgPx          decimal.Decimal // 多仓均价
	longMu             sync.RWMutex
	longTempLatestTime time.Time

	shortAmount         decimal.Decimal // 空仓（推送）
	shortTotal          decimal.Decimal // 空仓（总计）
	shortTemp           estimateValue   // 空仓（预估）
	shortAvgPx          decimal.Decimal // 空仓均价
	shortMu             sync.RWMutex
	shortTempLatestTime time.Time

	updateTime time.Time // 仓位刷新时间
}

func NewPositionImpl(instId string) *PositionImpl {
	p := new(PositionImpl)
	p.instId = instId
	return p
}

func (p *PositionImpl) Long() decimal.Decimal {
	return p.longTotal
}

func (p *PositionImpl) Short() decimal.Decimal {
	return p.shortTotal
}

func (p *PositionImpl) Net() decimal.Decimal {
	return p.longTotal.Sub(p.shortTotal)
}

func (p *PositionImpl) LongAvgPx() decimal.Decimal {
	return p.longAvgPx
}

func (p *PositionImpl) ShortAvgPx() decimal.Decimal {
	return p.shortAvgPx
}

func (p *PositionImpl) Ready() bool {
	// 如果临时仓位迟迟未被清除，则认为权益无效
	if p.longTemp.size > 0 && time.Now().UnixMilli()-p.longTempLatestTime.UnixMilli() > 5000 {
		return false
	}

	if p.shortTemp.size > 0 && time.Now().UnixMilli()-p.shortTempLatestTime.UnixMilli() > 5000 {
		return false
	}

	return true
}

func (p *PositionImpl) logPrefix() string {
	if len(p._logPrefix) == 0 {
		p._logPrefix = p.instId + "-pos"
	}

	return p._logPrefix
}

// 记录仓位增减
func (p *PositionImpl) RecordTempLong(long decimal.Decimal, t time.Time) {
	p.longMu.Lock()
	defer p.longMu.Unlock()

	// 过滤掉过时的数据
	if t.UnixMilli() > p.updateTime.UnixMilli() {
		p.longTemp.RecordValue(long, t)
		p.longTotal = p.longAmount.Add(p.longTemp.val)
		p.longTempLatestTime = t
		logger.LogDebug(p.logPrefix(), "recording temp long, temp=%v, time=%d", long, t.UnixMilli())
		logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%s", p.longAmount, p.longTemp.val, p.longTemp.String())
		logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%s", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
	}
}

func (p *PositionImpl) RecordTempShort(short decimal.Decimal, t time.Time) {
	p.shortMu.Lock()
	defer p.shortMu.Unlock()

	// 过滤掉过期的数据
	if t.UnixMilli() > p.updateTime.UnixMilli() {
		p.shortTemp.RecordValue(short, t)
		p.shortTotal = p.shortAmount.Add(p.shortTemp.val)
		p.shortTempLatestTime = t
		logger.LogDebug(p.logPrefix(), "recording temp short, temp=%v, time=%d", short, t.UnixMilli())
		logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%v", p.longAmount, p.longTemp.val, p.longTemp.String())
		logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%v", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
	}
}

// 刷新仓位，同时清理预估数据
func (p *PositionImpl) RefreshLong(long decimal.Decimal, avgPx decimal.Decimal, tm time.Time) {
	p.longMu.Lock()
	defer p.longMu.Unlock()
	amoutOrign := p.longAmount
	tempOrign := p.longTemp.val
	p.longTemp.ClearTill(tm)
	p.longAmount = long
	p.longTotal = p.longAmount.Add(p.longTemp.val)
	p.longAvgPx = avgPx
	p.updateTime = tm
	logger.LogDebug(p.logPrefix(), "refreshing long pos, sz:%v, ts:%d", long, tm.UnixMilli())
	logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%v", p.longAmount, p.longTemp.val, p.longTemp.String())
	logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%v", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
	logger.LogDebug(p.logPrefix(), "accurate:%v", p.longAmount.Sub(amoutOrign).Sub(tempOrign.Sub(p.longTemp.val)))
}

func (p *PositionImpl) RefreshShort(short decimal.Decimal, avgPx decimal.Decimal, tm time.Time) {
	p.shortMu.Lock()
	defer p.shortMu.Unlock()
	amoutOrign := p.shortAmount
	tempOrign := p.shortTemp.val
	p.shortTemp.ClearTill(tm)
	p.shortAmount = short
	p.shortAvgPx = avgPx
	p.shortTotal = p.shortAmount.Add(p.shortTemp.val)
	p.updateTime = tm
	logger.LogDebug(p.logPrefix(), "refreshing short pos, sz:%v, ts:%d", short, tm.UnixMilli())
	logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%v", p.longAmount, p.longTemp.val, p.longTemp.String())
	logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%v", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
	logger.LogDebug(p.logPrefix(), "accurate:%v", p.shortAmount.Sub(amoutOrign).Sub(tempOrign.Sub(p.shortTemp.val)))
}
