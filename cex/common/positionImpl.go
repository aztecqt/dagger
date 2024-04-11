/*
- @Author: aztec
- @Date: 2024-03-29 09:29:11
- @Description: 仓位的通用实现
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package common

import (
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

type PositionImpl struct {
	instId       string
	symbol       string
	contractType string
	_logPrefix   string

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

func NewPositionImpl(instId, symbol, contractType string) *PositionImpl {
	p := new(PositionImpl)
	p.instId = instId
	p.symbol = symbol
	p.contractType = contractType
	return p
}

func (p *PositionImpl) Symbol() string {
	return p.symbol
}

func (p *PositionImpl) ContractType() string {
	return p.contractType
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
	changed := !long.Equal(p.longAmount) || !avgPx.Equal(p.longAvgPx)
	amoutOrign := p.longAmount
	tempOrign := p.longTemp.val
	p.longTemp.ClearTill(tm)
	p.longAmount = long
	p.longTotal = p.longAmount.Add(p.longTemp.val)
	p.longAvgPx = avgPx
	p.updateTime = tm
	if changed {
		logger.LogDebug(p.logPrefix(), "refreshing long pos, sz:%v, ts:%d", long, tm.UnixMilli())
		logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%v", p.longAmount, p.longTemp.val, p.longTemp.String())
		logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%v", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
		logger.LogDebug(p.logPrefix(), "accurate:%v", p.longAmount.Sub(amoutOrign).Sub(tempOrign.Sub(p.longTemp.val)))
	}
}

func (p *PositionImpl) RefreshShort(short decimal.Decimal, avgPx decimal.Decimal, tm time.Time) {
	p.shortMu.Lock()
	defer p.shortMu.Unlock()
	changed := !short.Equal(p.shortAmount) || !avgPx.Equal(p.shortAvgPx)
	amoutOrign := p.shortAmount
	tempOrign := p.shortTemp.val
	p.shortTemp.ClearTill(tm)
	p.shortAmount = short
	p.shortAvgPx = avgPx
	p.shortTotal = p.shortAmount.Add(p.shortTemp.val)
	p.updateTime = tm
	if changed {
		logger.LogDebug(p.logPrefix(), "refreshing short pos, sz:%v, ts:%d", short, tm.UnixMilli())
		logger.LogDebug(p.logPrefix(), "long : size:%v, temp:%v, tempDetail:%v", p.longAmount, p.longTemp.val, p.longTemp.String())
		logger.LogDebug(p.logPrefix(), "short: size:%v, temp:%v, tempDetail:%v", p.shortAmount, p.shortTemp.val, p.shortTemp.String())
		logger.LogDebug(p.logPrefix(), "accurate:%v", p.shortAmount.Sub(amoutOrign).Sub(tempOrign.Sub(p.shortTemp.val)))
	}
}
