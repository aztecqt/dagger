/*
 * @Author: aztec
 * @Date: 2023-07-20 17:02:49
 * @Description: 一个数据集合，包含一组以名称为索引的时间序列，以及一组非均匀分布的点
 * 所有时间序列的间隔、起止时间都相同
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package datavisual

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"

	"github.com/aztecqt/dagger/stratergy"
	"github.com/aztecqt/dagger/util/logger"
)

var logPrefix = "datavisual"

type PointTag int

// 对应C#里的同名枚举
const (
	PointTag_Buy PointTag = iota
	PointTag_Sell
	PointTag_BuyMod1
	PointTag_SellMod1
	PointTag_BuyMod2
	PointTag_SellMod2
)

// 一个点
type Point struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"val"`
	Tag   PointTag  `json:"tag"`
}

// 一个时间事件（仅标记时间，不带y值）
type TimeEvent struct {
	Time  time.Time `json:"time"`
	Color Color     `json:"color"`
}

// 数据集合
type DataGroup struct {
	Interval int64 `json:"interval"` // 所包含的时间序列间隔

	lines      map[string]*stratergy.DataLine // 时间序列
	points     map[string][]Point             // 非均匀分布的点
	timeEvents map[string][]TimeEvent         // 时间事件
	extraInfo  strings.Builder
}

func NewDataGroup(interval int64) *DataGroup {
	dg := new(DataGroup)
	dg.Interval = interval
	dg.lines = make(map[string]*stratergy.DataLine)
	dg.points = make(map[string][]Point)
	dg.timeEvents = make(map[string][]TimeEvent)
	return dg
}

// 查找一个Line，没有则创建
func (d *DataGroup) FindOrAddDataLine(name string) *stratergy.DataLine {
	if dl, ok := d.lines[name]; ok {
		return dl
	} else {
		dl := new(stratergy.DataLine)
		dl.Init(name, math.MaxInt, d.Interval, 0)
		d.lines[name] = dl
		return dl
	}
}

// 查找一个Point数组，没有则创建
func (d *DataGroup) FindOrAddPointList(name string) []Point {
	if pts, ok := d.points[name]; ok {
		return pts
	} else {
		pts := make([]Point, 0)
		d.points[name] = pts
		return pts
	}
}

// 查找时间事件数组，没有则创建
func (d *DataGroup) FindOrAddTimeEventList(name string) []TimeEvent {
	if tes, ok := d.timeEvents[name]; ok {
		return tes
	} else {
		tes := make([]TimeEvent, 0)
		d.timeEvents[name] = tes
		return tes
	}
}

// 记录一个点到Line里
func (d *DataGroup) RecordLine(key string, val float64, t time.Time) {
	dl := d.FindOrAddDataLine(key)
	dl.Update(t.UnixMilli(), val)
}

func (d *DataGroup) CopyDataLine(dl *stratergy.DataLine) {
	for i := 0; i < dl.Length(); i++ {
		d.RecordLine(dl.Name(), dl.Values[i], time.UnixMilli(dl.Times[i]))
	}
}

func (d *DataGroup) CopyDataLineAsPoint(dl *stratergy.DataLine, tag PointTag) {
	for i := 0; i < dl.Length(); i++ {
		d.RecordPoint(dl.Name(), Point{Time: time.UnixMilli(dl.Times[i]), Value: dl.Values[i], Tag: tag})
	}
}

func (d *DataGroup) GetLine(key string) *stratergy.DataLine {
	return d.FindOrAddDataLine(key)
}

// 记录一个Point
func (d *DataGroup) RecordPoint(key string, pt Point) {
	pts := d.FindOrAddPointList(key)
	if len(pts) > 0 && pts[len(pts)-1].Time.UnixMilli() == pt.Time.UnixMilli() {
		pt.Time = pts[len(pts)-1].Time.Add(time.Nanosecond)
	}
	d.points[key] = append(pts, pt)
}

// 记录一个TimeEvent
func (d *DataGroup) RecordTimeEvent(key string, te TimeEvent) {
	tes := d.FindOrAddTimeEventList(key)
	d.timeEvents[key] = append(tes, te)
}

// 记录额外信息
func (d *DataGroup) SaveExtraInfo(info string) {
	d.extraInfo.WriteString(info)
}

// 保存到目录中
func (d *DataGroup) SaveToDir(root string) {
	util.MakeSureDir(root)

	// 本体保存到datagroup.json文件
	path := fmt.Sprintf("%s/datagroup.json", root)
	util.ObjectToFile(path, d)

	// Line保存成.dataline文件
	for name, dl := range d.lines {
		n := name
		d := dl
		path := fmt.Sprintf("%s/%s.dataline", root, n)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		sb := strings.Builder{}
		if err == nil {
			for i := 0; i < d.Length(); i++ {
				d, ok := d.GetData(i)
				if ok {
					sb.WriteString(fmt.Sprintf("%s,%v\n", time.UnixMilli(d.MS).Format(time.DateTime), d.V))
					// sb.WriteString(fmt.Sprintf("%d,%v\n", d.MS, d.V))
				}
			}
			file.WriteString(sb.String())
			file.Close()
		} else {
			logger.LogImportant(logPrefix, "error while save datagroup: %s", err.Error())
		}
	}

	// Points保存成.points文件
	for name, pts := range d.points {
		n := name
		p := pts
		path := fmt.Sprintf("%s/%s.points", root, n)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		sb := strings.Builder{}
		if err == nil {
			for i := 0; i < len(p); i++ {
				pt := p[i]
				sb.WriteString(fmt.Sprintf("%s,%v,%d\n", pt.Time.Format(time.DateTime), pt.Value, pt.Tag))
				// sb.WriteString(fmt.Sprintf("%d,%v,%d\n", pt.Time.UnixMilli(), pt.Value, pt.Tag))
			}
			file.WriteString(sb.String())
			file.Close()
		} else {
			logger.LogImportant(logPrefix, "error while save datagroup: %s", err.Error())
		}
	}

	// TimeEvents保存成.events文件
	for name, evts := range d.timeEvents {
		n := name
		e := evts
		path := fmt.Sprintf("%s/%s.events", root, n)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		sb := strings.Builder{}
		if err == nil {
			for i := 0; i < len(e); i++ {
				evt := e[i]
				sb.WriteString(fmt.Sprintf("%s,%d,%d,%d\n", evt.Time.Format(time.DateTime), evt.Color.R, evt.Color.G, evt.Color.B))
			}
			file.WriteString(sb.String())
			file.Close()
		} else {
			logger.LogImportant(logPrefix, "error while save datagroup: %s", err.Error())
		}
	}

	// ExtraInfo保存为info.txt
	path = fmt.Sprintf("%s/info.txt", root)
	util.StringToFile(path, d.extraInfo.String())

	time.Sleep(time.Second)
}
