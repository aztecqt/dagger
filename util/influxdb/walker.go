/*
 * @Author: aztec
 * @Date: 2022-10-31 09:18:43
 * @Description: influxdb数据遍历器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package influxdb

import (
	"time"

	"aztecqt/dagger/util/logger"
	"github.com/influxdata/influxdb/client/v2"
)

type Walker struct {
	logPrefix string
	conn      client.Client
}

func (w *Walker) Init(cfg ConnConfig) bool {
	w.logPrefix = "influx-walker"
	conn, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.UserName,
		Password: cfg.Password,
	})

	if err == nil {
		w.conn = conn
		logger.LogImportant(w.logPrefix, "connected to influxDb(%s) success", cfg.Addr)
		return true
	} else {
		logger.LogImportant(w.logPrefix, "connected to influxDb(%s) failed: %s", cfg.Addr, err.Error())
		return false
	}
}

func (w *Walker) Walk(
	dbName,
	rp,
	measurement string,
	fields []string,
	tags map[string]string,
	tStart,
	tEnd time.Time,
	d time.Duration,
	callback func(resp *client.Response, progress float64, finished bool)) {
	if len(rp) == 0 {
		rp = "autogen"
	}

	// 拼command
	q := MakeQuery(fields, dbName, rp, measurement, tags, tStart, tEnd)
	logger.LogImportant(w.logPrefix, q.Command)

	// 查询
	cresp, err := w.conn.QueryAsChunk(q)
	if err != nil {
		logger.LogImportant(w.logPrefix, "query failed: %s", err.Error())
		return
	}

	for {
		if resp, err := cresp.NextResponse(); err == nil {
			for _, r := range resp.Results {
				if len(r.Err) == 0 {
					maxTime := time.Time{}
					for _, r2 := range r.Series {
						if len(r2.Values) > 0 && len(r2.Values[len(r2.Values)-1]) > 0 {
							if tdata, err := time.Parse(time.RFC3339Nano, r2.Values[len(r2.Values)-1][0].(string)); err == nil {
								if tdata.After(maxTime) {
									maxTime = tdata
								}
							}
						}
					}

					progress := float64(maxTime.Unix()-tStart.Unix()) / float64(tEnd.Unix()-tStart.Unix())
					callback(resp, progress, false)
				} else {
					logger.LogImportant(w.logPrefix, "get next response failed, err=%s", r.Err)
					time.Sleep(time.Second * 10)
				}
			}
		} else {
			if resp == nil && err.Error() == "EOF" {
				callback(nil, 1, true)
				logger.LogImportant(w.logPrefix, "finished")
				break
			} else {
				logger.LogImportant(w.logPrefix, "get next response failed, err=%s", err.Error())
				time.Sleep(time.Second * 10)
			}
		}
	}
}
