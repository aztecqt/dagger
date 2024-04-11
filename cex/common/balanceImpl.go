/*
- @Author: aztec
- @Date: 2024-03-29 09:30:50
- @Description: 余额的通用实现
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
func (b *BalanceImpl) Refresh(rights, frozen decimal.Decimal, tm time.Time) {
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

func (b *BalanceImpl) Ready() bool {
	// 如果临时权益迟迟未被清除，则认为权益无效
	if b.temp.size > 0 && time.Now().UnixMilli()-b.lastTempTime.UnixMilli() > 5000 {
		return false
	}

	return true
}
