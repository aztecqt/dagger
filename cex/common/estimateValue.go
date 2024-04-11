/*
- @Author: aztec
- @Date: 2024-03-29 09:32:30
- @Description: 用于记录一个“本地可以修改、但又可以受网络刷新”的值
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package common

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

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
