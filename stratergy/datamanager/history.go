/*
 * @Author: aztec
 * @Date: 2022-12-30 10:39:03
 * @Description: 历史数据获取器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package datamanager

import (
	"encoding/json"
	"fmt"
	"time"

	"aztecqt/dagger/stratergy"
	"aztecqt/dagger/util/influxdb"
	"aztecqt/dagger/util/logger"
	"github.com/influxdata/influxdb/client/v2"
)

// 历史行情获取器
type History interface {
	GetFuturePrice(ccy, contractType string, t0, t1 time.Time, dl *stratergy.DataLine)
	GetSpotPrice(baseCcy, quoteCcy string, t0, t1 time.Time, dl *stratergy.DataLine)
}

var history History

func GetHistoryFuturePrice(ccy, contractType string, t0, t1 time.Time, dl *stratergy.DataLine) {
	logPrefix := "history"
	if history == nil {
		logger.LogPanic(logPrefix, "history not inited")
	} else {
		history.GetFuturePrice(ccy, contractType, t0, t1, dl)
	}
}
func GetHistorySpotPrice(baseCcy, quoteCcy string, t0, t1 time.Time, dl *stratergy.DataLine) {
	logPrefix := "history"
	if history == nil {
		logger.LogPanic(logPrefix, "history not inited")
	} else {
		history.GetSpotPrice(baseCcy, quoteCcy, t0, t1, dl)
	}
}

func InitHistoryInflux(cfg influxdb.ConnConfig) {
	h := new(HistoryFromInflux)
	h.Init(cfg)
	history = h
}

type HistoryFromInflux struct {
	logPrefix string
	walker    influxdb.Walker
	walkerOk  bool
}

func (h *HistoryFromInflux) Init(cfg influxdb.ConnConfig) {
	h.logPrefix = "history-influx"

	if h.walker.Init(cfg) {
		h.walkerOk = true
	} else {
		logger.LogImportant(h.logPrefix, "create walker failed")
		h.walkerOk = false
	}
}

func (h *HistoryFromInflux) GetFuturePrice(ccy, contractType string, t0, t1 time.Time, dl *stratergy.DataLine) {
	if !h.walkerOk {
		return
	}

	dbName := "marketinfo"
	rp := "ticker"
	mm := "okx-ticker"
	fields := make([]string, 0)
	fields = append(fields, "price")
	tags := make(map[string]string)
	tags["id"] = fmt.Sprintf("%s_%s", ccy, contractType)

	h.walker.Walk(dbName, rp, mm, fields, tags, t0, t1, time.Hour*24, func(resp *client.Response, progress float64, finished bool) {
		if resp != nil && len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
			vs := resp.Results[0].Series[0].Values
			for _, v := range vs {
				t, _ := time.Parse(time.RFC3339, v[0].(string))
				p, _ := v[1].(json.Number).Float64()
				dl.Update(t.UnixMilli(), p)
			}
		}
	})
}

func (h *HistoryFromInflux) GetSpotPrice(baseCcy, quoteCcy string, t0, t1 time.Time, dl *stratergy.DataLine) {
	logger.LogImportant(h.logPrefix, "'GetSpotPrice' not implemented")
}
