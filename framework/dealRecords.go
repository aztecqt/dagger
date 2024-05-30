/*
- @Author: aztec
- @Date: 2024-05-05 15:35:55
- @Description: 成交记录
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package framework

import (
	"bufio"
	"os"
	"time"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/shopspring/decimal"
)

// 成交
type DealRecord struct {
	Type      string          `json:"type"`
	TimeStamp int64           `json:"ts"`
	Dir       common.OrderDir `json:"dir"`
	Price     decimal.Decimal `json:"price"`
	Amount    decimal.Decimal `json:"amount"`
}

// 一组成交记录，有最大长度和最久保存时间。带持久化
type DealRecords struct {
	logPrefix     string
	path          string
	Deals         []DealRecord
	maxCount      int
	maxTimeLenSec int64
}

// 初始化
func (d *DealRecords) Init(maxCount int, maxTimeLenSec int64, path, logPrefix string) {
	d.path = path
	d.logPrefix = logPrefix
	d.Deals = make([]DealRecord, 0)
	d.maxCount = maxCount
	d.maxTimeLenSec = maxTimeLenSec
	d.load()
}

// 加载
func (d *DealRecords) load() {
	util.MakeSureDirForFile(d.path)
	if file, err := os.OpenFile(d.path, os.O_RDONLY, os.ModePerm); err == nil {
		reader := bufio.NewReader(file)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				break
			}

			dr := DealRecord{}
			if util.ObjectFromString(string(line), &dr) == nil {
				d.Deals = append(d.Deals, dr)
			}
		}
		file.Close()
		logger.LogImportant(d.logPrefix, "loaded %d deal records", len(d.Deals))
	} else {
		logger.LogImportant(d.logPrefix, "load deal records failed: %s", err.Error())
	}
}

// 记录一个新的dealRecord
// 正常情况下，只追加保存一个deal
// 当数量超过最大数量的150%，或者时间超过最长保存时间的150%时，裁剪内存数据，并执行一次全量保存
func (d *DealRecords) AddDeal(dr DealRecord) {
	d.Deals = append(d.Deals, dr)
	if d.needCut() {
		d.cut()
		d.saveAll()
	} else {
		d.save(dr)
	}
}

// 追加保存一个成交
func (d *DealRecords) save(dr DealRecord) {
	if file, err := os.OpenFile(d.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm); err == nil {
		writer := bufio.NewWriter(file)
		writer.WriteString(util.Object2StringWithoutIntent(dr))
		writer.WriteString("\n")
		writer.Flush()
		file.Close()
		logger.LogImportant(d.logPrefix, "save dr to %s success", d.path)
	} else {
		logger.LogImportant(d.logPrefix, "save dr failed: %s", err.Error())
	}
}

// 保存所有成交
func (d *DealRecords) saveAll() {
	if exist, _ := util.PathExists(d.path); exist {
		os.Remove(d.path)
	}

	if file, err := os.OpenFile(d.path, os.O_CREATE|os.O_WRONLY, os.ModePerm); err == nil {
		writer := bufio.NewWriter(file)
		for _, dr := range d.Deals {
			writer.WriteString(util.Object2StringWithoutIntent(dr))
			writer.WriteString("\n")
		}
		writer.Flush()
		file.Close()
		logger.LogImportant(d.logPrefix, "save all dr to %s success", d.path)
	} else {
		logger.LogImportant(d.logPrefix, "save all dr failed: %s", err.Error())
	}
}

// 是否需要裁剪
func (d *DealRecords) needCut() bool {
	if len(d.Deals) > int(float64(d.maxCount)*1.5) {
		return true
	}

	if len(d.Deals) > 0 {
		if time.Now().UnixMilli()-d.Deals[0].TimeStamp > d.maxTimeLenSec*1000 {
			return true
		}
	}

	return false
}

// 裁剪
func (d *DealRecords) cut() {
	startIndex := 0
	if len(d.Deals) > d.maxCount {
		startIndex = len(d.Deals) - d.maxCount
	}

	minMs := time.Now().UnixMilli() - d.maxTimeLenSec*1000
	for i := startIndex; i < len(d.Deals); i++ {
		if d.Deals[i].TimeStamp >= minMs {
			startIndex = i
			break
		}
	}

	d.Deals = d.Deals[startIndex:]
}
