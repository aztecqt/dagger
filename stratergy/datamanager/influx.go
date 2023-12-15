/*
 * @Author: aztec
 * @Date: 2022-12-27 17:42:30
 * @Description: 策略的流式数据存储器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package datamanager

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/stratergy"
	"github.com/aztecqt/dagger/util/influxdb"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/influxdata/influxdb/client/v2"
)

// 空间中一个时间切片上的一组数据点
type dataPoint struct {
	timeStampMicro int64
	points         map[string]float64
}

// 一个策略数据组，包含一系列数据点，以及一组成交
// 简单的策略只包含一个数据组即可
// 复合策略可能有多个数据组
type dataGroup struct {
	deals  []stratergy.Deal
	points []dataPoint
}

func (d *dataGroup) clearAll() {
	d.deals = d.deals[:0]
	d.points = d.points[:0]
}

func (d *dataGroup) clearDeals() {
	d.deals = d.deals[:0]
}

func (d *dataGroup) clearPoints() {
	d.points = d.points[:0]
}

func NewInfluxContext(gtype, gname string) *InfluxContext {
	if influxIns == nil {
		logger.LogPanic("", "must call InitInflux() before create InfluxContext")
	}
	dc := new(InfluxContext)
	dc.inf = influxIns
	dc.gtype = gtype
	dc.gname = gname
	return dc
}

// 应用层使用这个接口来保存数据
type InfluxContext struct {
	inf   *Influx
	gtype string
	gname string
}

func (d *InfluxContext) AddDeal(deal common.Deal, dealType string) {
	d.inf.AddDeal(d.gtype, d.gname, dealType, deal)
}

func (d *InfluxContext) AddDealRaw(deal stratergy.Deal) {
	d.inf.AddDealRaw(d.gtype, d.gname, deal)
}

func (d *InfluxContext) AddDataPoints(points map[string]float64, t time.Time) {
	d.inf.AddDataPoints(d.gtype, d.gname, points, t)
}

func (d *InfluxContext) ClearDataPoints() {
	d.inf.ClearDataPoints(d.gtype, d.gname)
}

type Influx struct {
	name                  string // 策略名
	logPrefix             string
	measurementNamePoints string // 这个表记录曲线
	measurementNameDeals  string // 这个表记录成交
	cfg                   influxdb.ConnConfig
	ifdbConn              client.Client
	chStop                chan int

	// 将要保存的数据
	// key的格式为 gtype.gname
	// 前端通过gtype就可以确定数据的展现方式
	// gtype可以是策略的class，也可以是交易器的名称，总之能唯一描述数据格式就可以
	// gname用于显示和区分
	dgToSave        map[string]*dataGroup
	dgToClearPoints *hashset.Set
	dgMu            sync.Mutex
}

var influxIns *Influx = nil

func InitInflux(name string, cfg influxdb.ConnConfig) *Influx {
	inf := new(Influx)
	inf.name = name
	inf.logPrefix = "DataManager.Influx"
	inf.measurementNameDeals = fmt.Sprintf("%s_deals", inf.name)
	inf.measurementNamePoints = fmt.Sprintf("%s_points", inf.name)
	inf.chStop = make(chan int)
	inf.dgToSave = make(map[string]*dataGroup)
	inf.dgToClearPoints = hashset.New()
	inf.cfg = cfg

	inf.Start()
	influxIns = inf
	return inf
}

func (m *Influx) Start() {
	go m.update()
}

func (m *Influx) Stop() {
	m.chStop <- 0
}

func (m *Influx) AddDeal(gtype, gname, dealType string, deal common.Deal) {
	m.dgMu.Lock()
	defer m.dgMu.Unlock()

	d := stratergy.Deal{DealType: dealType}
	d.From(deal)
	g := m.getDataGroup(gtype, gname)

	// 时间戳不能重复，不然加入influx时会覆盖旧值
	if len(g.deals) > 0 {
		lastdeal := g.deals[len(g.deals)-1]
		if d.TimeStampMicro <= lastdeal.TimeStampMicro {
			d.TimeStampMicro = lastdeal.TimeStampMicro + 1
		}
	}

	g.deals = append(g.deals, d)
}

func (i *Influx) AddDealRaw(gtype, gname string, d stratergy.Deal) {
	i.dgMu.Lock()
	defer i.dgMu.Unlock()

	g := i.getDataGroup(gtype, gname)

	// 时间戳不能重复，不然加入influx时会覆盖旧值
	if len(g.deals) > 0 {
		lastdeal := g.deals[len(g.deals)-1]
		if d.TimeStampMicro <= lastdeal.TimeStampMicro {
			d.TimeStampMicro = lastdeal.TimeStampMicro + 1
		}
	}

	g.deals = append(g.deals, d)
}

func (i *Influx) AddDataPoints(gtype, gname string, points map[string]float64, t time.Time) {
	i.dgMu.Lock()
	defer i.dgMu.Unlock()

	g := i.getDataGroup(gtype, gname)

	// ponit的时间戳可以重复（一般也不会重复）
	dp := dataPoint{points: points, timeStampMicro: t.UnixMicro()}
	g.points = append(g.points, dp)
}

func (i *Influx) ClearDataPoints(gtype, gname string) {
	i.dgMu.Lock()
	defer i.dgMu.Unlock()

	g := i.getDataGroup(gtype, gname)
	g.clearPoints()
	i.dgToClearPoints.Add(i.getDataGroupKey(gtype, gname))
}

// 查询/创建dataGroup
func (i *Influx) getDataGroup(gtype, gname string) *dataGroup {
	key := i.getDataGroupKey(gtype, gname)
	if len(gname) == 0 {
		key = fmt.Sprintf("%s", gtype)
	}

	if _, ok := i.dgToSave[key]; !ok {
		g := new(dataGroup)
		i.dgToSave[key] = g
	}

	v, _ := i.dgToSave[key]
	return v
}

func (*Influx) getDataGroupKey(gtype, gname string) string {
	return fmt.Sprintf("%s.%s", gtype, gname)
}

func (i *Influx) update() {
	ticker := time.NewTicker(time.Millisecond * 500)

	// 调试一下端口泄露问题，定期断开连接试试
	writeCount := 0

	for {
		select {
		case <-ticker.C:
			i.writeInflux()
			writeCount++
			if writeCount >= 1000 {
				i.ifdbConn.Close()
				i.ifdbConn = nil
				writeCount = 0
			}
		case <-i.chStop:
			i.writeInflux()
		}
	}
}

func (i *Influx) checkConnection() bool {
	if i.ifdbConn == nil {
		if conn, err := client.NewHTTPClient(client.HTTPConfig{
			Addr:     i.cfg.Addr,
			Username: i.cfg.UserName,
			Password: i.cfg.Password,
		}); err == nil {
			i.ifdbConn = conn
			logger.LogImportant(i.logPrefix, "connected to influxdb %s", i.cfg.Addr)
		} else {
			logger.LogImportant(i.logPrefix, "connected to influxdb %s failed, err=%s", i.cfg.Addr, err.Error())
		}
	}

	return i.ifdbConn != nil
}

// 将缓存的成交数据，和point数据，都写入influx中
func (i *Influx) writeInflux() {
	if !i.checkConnection() {
		return
	}

	ifdbName := "stratergy"
	rp := "autogen"

	// 需要删除的数据
	if i.dgToClearPoints.Size() > 0 {
		for _, dgkey := range i.dgToClearPoints.Values() {
			influxdb.DeleteMeasurementWithTag(
				i.ifdbConn, ifdbName, rp, i.measurementNamePoints, "group", dgkey.(string))
		}
		i.dgToClearPoints.Clear()
	}

	// 写入所有数据
	batchPoint, err := client.NewBatchPoints(client.BatchPointsConfig{
		Precision:       "ns",
		Database:        ifdbName,
		RetentionPolicy: rp,
	})

	if err != nil {
		logger.LogImportant(i.logPrefix, err.Error())
		return
	}

	func() {
		i.dgMu.Lock()
		defer i.dgMu.Unlock()

		for k, dg := range i.dgToSave {
			// 以datagroup的组作为tag
			tags := make(map[string]string)
			tags["group"] = k

			for _, deal := range dg.deals {
				pt := i.dealToPoint(deal, tags)
				if pt != nil {
					batchPoint.AddPoint(pt)
				}
			}

			for _, dp := range dg.points {
				pt := i.dataPoint2Point(dp, tags)
				if pt != nil {
					batchPoint.AddPoint(pt)
				}
			}
		}
	}()

	if len(batchPoint.Points()) > 0 {
		if err := i.ifdbConn.Write(batchPoint); err != nil {
			logger.LogImportant(i.logPrefix, err.Error())
		}

		// 清除缓存
		func() {
			i.dgMu.Lock()
			defer i.dgMu.Unlock()
			for _, dg := range i.dgToSave {
				dg.clearAll()
			}
		}()
	}
}

func (i *Influx) dealToPoint(d stratergy.Deal, tags map[string]string) *client.Point {
	fields := make(map[string]interface{})
	b, _ := json.Marshal(d)
	fields[d.DealType] = string(b)

	pt, err := client.NewPoint(
		i.measurementNameDeals,
		tags,
		fields,
		time.UnixMicro(d.TimeStampMicro),
	)

	if err != nil {
		logger.LogImportant(i.logPrefix, err.Error())
		return nil
	} else {
		return pt
	}
}

func (i *Influx) dataPoint2Point(dp dataPoint, tags map[string]string) *client.Point {
	fields := make(map[string]interface{})
	for k, v := range dp.points {
		fields[k] = v
	}

	pt, err := client.NewPoint(
		i.measurementNamePoints,
		tags,
		fields,
		time.UnixMicro(dp.timeStampMicro),
	)

	if err != nil {
		logger.LogImportant(i.logPrefix, err.Error())
		return nil
	} else {
		return pt
	}
}
