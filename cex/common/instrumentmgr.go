/*
 * @Author: aztec
 * @Date: 2023-02-20 15:00
 * @Description: 交易对类型信息管理器
  * Copyright (c) 2022 by aztec, All Rights Reserved.
*/
package common

import (
	"github.com/aztecqt/dagger/util/logger"

	"github.com/shopspring/decimal"
)

type InstrumentMgr struct {
	logPrefix       string
	instrumentsById map[string] /*instId*/ *Instruments
	instruments     []*Instruments
}

func NewInstrumentMgr(logPrefix string) *InstrumentMgr {
	i := new(InstrumentMgr)
	i.logPrefix = logPrefix
	i.instrumentsById = make(map[string]*Instruments)
	i.instruments = make([]*Instruments, 0)
	return i
}

func (i *InstrumentMgr) Set(instId string, ins *Instruments) {
	i.instrumentsById[instId] = ins
	i.instruments = append(i.instruments, ins)
}

func (i *InstrumentMgr) Get(instId string) *Instruments {
	if v, ok := i.instrumentsById[instId]; ok {
		return v
	} else {
		return nil
	}
}

func (i *InstrumentMgr) GetAll() []*Instruments {
	return i.instruments
}

func (i *InstrumentMgr) AlignPriceNumber(instId string, price decimal.Decimal) decimal.Decimal {
	if inst, ok := i.instrumentsById[instId]; ok {
		price = price.Round(-inst.TickSize.Exponent())
		return price
	} else {
		logger.LogPanic(i.logPrefix, "unknown instid:%s", instId)
		return decimal.Zero
	}
}

func (i *InstrumentMgr) AlignPrice(instId string, price decimal.Decimal, dir OrderDir, makeOnly bool, buy1, sell1 decimal.Decimal) decimal.Decimal {
	if inst, ok := i.instrumentsById[instId]; ok {
		if dir == OrderDir_Buy {
			price = price.RoundDown(-inst.TickSize.Exponent())
		} else {
			price = price.RoundUp(-inst.TickSize.Exponent())
		}

		if makeOnly {
			if dir == OrderDir_Buy && price.GreaterThanOrEqual(buy1) {
				price = buy1
			} else if dir == OrderDir_Sell && price.LessThanOrEqual(sell1) {
				price = sell1
			}
		}
		return price
	} else {
		logger.LogPanic(i.logPrefix, "unknown instid:%s", instId)
		return decimal.Zero
	}
}

func (i *InstrumentMgr) AlignSize(instId string, size decimal.Decimal) decimal.Decimal {
	if inst, ok := i.instrumentsById[instId]; ok {
		// 精度对齐
		c := size.Div(inst.LotSize).IntPart()
		size = inst.LotSize.Mul(decimal.NewFromInt(c))
		return size
	} else {
		logger.LogPanic(i.logPrefix, "unknown instid:%s", instId)
		return decimal.Zero
	}
}

func (i *InstrumentMgr) MinSize(instId string, price decimal.Decimal) decimal.Decimal {
	if inst, ok := i.instrumentsById[instId]; ok {
		// 兼顾MinSize和MinValue
		return i.AlignSize(instId, decimal.Max(inst.MinSize, inst.MinValue.Div(price)))
	} else {
		logger.LogPanic(i.logPrefix, "unknown instid:%s", instId)
		return decimal.Zero
	}
}
