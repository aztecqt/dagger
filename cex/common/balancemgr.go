/*
 * @Author: aztec
 * @Date: 2023-02-20 11:14:24
 * @Description: 用户权益
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package common

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type BalanceMgr struct {
	needInit     bool
	balanceByCcy map[string] /*ccy*/ *BalanceImpl
	muBalance    sync.RWMutex
}

func NewBalanceMgr(needInit bool) *BalanceMgr {
	b := new(BalanceMgr)
	b.balanceByCcy = make(map[string]*BalanceImpl)
	b.needInit = needInit
	return b
}

// 设置权益
// 返回与本地的偏移值
func (e *BalanceMgr) RefreshBalance(ccy string, free, frozen decimal.Decimal, refreshTime time.Time) decimal.Decimal {
	bal := e.FindBalance(ccy)
	return bal.Refresh(free.Add(frozen), frozen, refreshTime)
}

// 获取所有资产
func (e *BalanceMgr) GetAllBalances() []*BalanceImpl {
	e.muBalance.Lock()
	defer e.muBalance.Unlock()

	bals := make([]*BalanceImpl, 0, len(e.balanceByCcy))
	for _, bi := range e.balanceByCcy {
		bals = append(bals, bi)
	}
	return bals
}

// 查找某币种的权益，指针可以长期保存使用
// 一般trader可以保存此指针
func (e *BalanceMgr) FindBalance(ccy string) *BalanceImpl {
	e.muBalance.Lock()
	defer e.muBalance.Unlock()

	var b *BalanceImpl

	if _, ok := e.balanceByCcy[ccy]; !ok {
		e.balanceByCcy[ccy] = NewBalanceImpl(ccy, e.needInit)
	}

	b = e.balanceByCcy[ccy]
	return b
}

// 调用这个，得手动Lock/Unlock
func (e *BalanceMgr) FindBalanceUnsafe(ccy string) *BalanceImpl {
	var b *BalanceImpl

	if _, ok := e.balanceByCcy[ccy]; !ok {
		e.balanceByCcy[ccy] = NewBalanceImpl(ccy, e.needInit)
	}

	b = e.balanceByCcy[ccy]
	return b
}

// 手动Lock/Unlock
func (e *BalanceMgr) Lock() {
	e.muBalance.Lock()
}

func (e *BalanceMgr) Unlock() {
	e.muBalance.Unlock()
}
