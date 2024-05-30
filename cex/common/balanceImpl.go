/*
- @Author: aztec
- @Date: 2024-03-29 09:30:50
- @Description: 余额的通用实现
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package common

import (
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

type BalanceImpl struct {
	inited bool
	ccy    string
	rights decimal.Decimal // 权益（推送）
	total  decimal.Decimal // 权益（总计）
	temp   estimateValue   // 权益（预估）
	mu     sync.Mutex

	frozen       decimal.Decimal // 冻结权益
	updateTime   time.Time       // 余额刷新时间
	lastTempTime time.Time

	maxPitchAllowed  decimal.Decimal // 容许最大偏移。如果偏移超出此值，则置为not ready，策略会停下来
	maxPitchAppeared decimal.Decimal // 出现过的最大偏移
}

func NewBalanceImpl(ccy string, needInit bool) *BalanceImpl {
	b := new(BalanceImpl)
	b.ccy = ccy
	if !needInit {
		b.inited = true
	}
	return b
}

// 设置最大容忍偏移
func (b *BalanceImpl) SetMaxPitchAllowed(v decimal.Decimal) {
	b.maxPitchAllowed = v
}

// 记录一项临时权益增减
func (b *BalanceImpl) RecordTempRights(r decimal.Decimal, t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.temp.RecordValue(r, t)
	b.total = b.rights.Add(b.temp.val)
	b.lastTempTime = t
	logger.LogDebug(b.ccy, "record temp: rights:%v, frozen:%v, temp:%v, tempDetail:%v", b.rights, b.frozen, b.temp.val, b.temp.String())
}

// 刷新权益，同时清理预估数据
// 返回权益偏移值
func (b *BalanceImpl) Refresh(rights, frozen decimal.Decimal, tm time.Time) decimal.Decimal {
	b.mu.Lock()
	defer b.mu.Unlock()
	rightsOrign := b.rights
	tempOrign := b.temp.val
	b.temp.ClearTill(tm)
	b.rights = rights
	b.frozen = frozen
	b.total = b.rights.Add(b.temp.val)
	b.updateTime = tm
	pitch := decimal.Zero
	if b.inited {
		pitch = b.rights.Sub(rightsOrign).Sub(tempOrign.Sub(b.temp.val))
	}

	if pitch != decimal.Zero {
		logger.LogDebug(b.ccy, "refreshing: rights:%v, frozen:%v, temp:%v, tempDetail:%v", b.rights, b.frozen, b.temp.val, b.temp.String())
	} else {
		if pitch.GreaterThan(b.maxPitchAppeared) {
			b.maxPitchAppeared = pitch
		}

		logger.LogDebug(b.ccy, "refreshing: rights:%v, frozen:%v, temp:%v, tempDetail:%v, pitch:%v", b.rights, b.frozen, b.temp.val, b.temp.String(), pitch)
	}

	b.inited = true
	return pitch
}

func (b *BalanceImpl) Ccy() string {
	return b.ccy
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

func (b *BalanceImpl) Ready() (bool, string) {
	if !b.inited {
		return false, "not inited"
	}

	// 如果临时权益迟迟未被清除，则认为权益无效
	if b.temp.size > 0 && time.Now().UnixMilli()-b.lastTempTime.UnixMilli() > 5000 {
		return false, fmt.Sprintf("temp rights not cleared till %s", b.lastTempTime.Format(time.DateTime))
	}

	// 如果偏移超出容忍，则无效
	if b.maxPitchAllowed.IsPositive() && b.maxPitchAppeared.GreaterThan(b.maxPitchAllowed) {
		return false, fmt.Sprintf("rights pitch too large! allow=%v, real=%v", b.maxPitchAllowed, b.maxPitchAppeared)
	}

	return true, ""
}
