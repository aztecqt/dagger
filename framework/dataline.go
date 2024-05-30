/*
 * @Author: aztec
 * @Date: 2022-04-09 17:34:59
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2024-05-05 10:55:02
 * @FilePath: \dagger\stratergy\dataline.go
 * @Description: 按固定时间间隔排列的数据队列
 * 对于外部输入时间-数据对，将其时间对齐后再记录
 * 有最大长度，超出最大长度则舍弃多余的数据
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package framework

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/markcheno/go-talib"
	"github.com/shopspring/decimal"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

type DataLine struct {
	name          string
	logPrefix     string
	filePath      string // 保存文件路径。正确设定后会自动存储
	maxLength     int
	intervalMs    int64
	lastAlignedMs int64
	ver           int
	Values        []float64
	Times         []int64
}

// #region DataLineUnit
type DatalineUnit struct {
	MS int64
	V  float64
}

func (du *DatalineUnit) String() string {
	return fmt.Sprintf(`%d:%v`, du.MS, du.V)
}

func (du *DatalineUnit) Parse(str string) {
	ss := strings.Split(str, ":")
	if len(ss) != 2 {
		logger.LogPanic("DataLineUnit", "parse bad format:%s", str)
	}

	du.MS = util.String2Int64Panic(ss[0])
	du.V = util.String2Float64Panic(ss[1])
}

// #endregion

// #region core
func (d *DataLine) Init(name string, maxLength int, intervalMs int64, ver int) *DataLine {
	d.name = name
	d.logPrefix = "dataline_" + name
	d.maxLength = maxLength
	d.intervalMs = intervalMs
	d.Values = make([]float64, 0)
	d.Times = make([]int64, 0)
	d.ver = ver
	return d
}

func (d *DataLine) Name() string {
	return d.name
}

func (d *DataLine) WithFilePath(path string) *DataLine {
	if len(path) > 0 {
		d.filePath = path
		d.load()
	}

	return d
}

func (d *DataLine) WithFileDirAndPath(dir, path string) *DataLine {
	if len(dir) > 0 && len(path) > 0 {
		if dir[len(dir)-1] != '\\' && dir[len(dir)-1] != '/' {
			dir = dir + "/"
		}
		d.filePath = dir + path
		d.load()
	}

	return d
}

func (d *DataLine) MaxLength() int {
	return d.maxLength
}

func (d *DataLine) IntervalMS() int64 {
	return d.intervalMs
}

func (d *DataLine) Clear() {
	d.Values = d.Values[:0]
	d.Times = d.Times[:0]
	d.ver++
}

// 从一个float序列，创建一个时间同步的新的line
func (d *DataLine) CreateParallel(name string, vals []float64) *DataLine {
	if len(vals) != len(d.Values) {
		return nil
	}

	dl := &DataLine{}
	dl.Init(name, d.maxLength, d.intervalMs, 0)
	dl.Times = slices.Clone(d.Times)
	dl.Values = slices.Clone(vals)
	return dl
}

func (d *DataLine) Update(ms int64, v float64) {
	d.updateInner(ms, v)
}

func (d *DataLine) UpdateDecimal(ms int64, v decimal.Decimal) {
	d.updateInner(ms, v.InexactFloat64())
}

func (d *DataLine) updateInner(ms int64, v float64) {
	if len(d.Values) == 0 {
		// 首个元素直接插入
		msAligned := ms / d.intervalMs * d.intervalMs
		d.Times = append(d.Times, msAligned)
		d.Times = append(d.Times, ms)
		d.Values = append(d.Values, v)
		d.Values = append(d.Values, v)
		d.lastAlignedMs = msAligned
	} else {
		for {
			l := len(d.Values)
			deltaMs := ms - d.lastAlignedMs
			if deltaMs >= 0 {
				if deltaMs < d.intervalMs {
					// 更新最后一个元素
					d.Values[l-1] = v
					d.Times[l-1] = ms
					break
				} else {
					// 补齐后面的元素
					d.Times[l-1] = d.Times[l-2] + d.intervalMs
					d.Values[l-1] = v
					d.lastAlignedMs = d.Times[l-1]
					d.Values = append(d.Values, v)
					d.Times = append(d.Times, d.Times[l-1]+1)
				}
			} else {
				break
			}
		}

		// 数据过长则舍弃一半元素
		if d.maxLength > 0 {
			if len(d.Values) >= d.maxLength {
				d.Values = append(d.Values[:0], d.Values[len(d.Values)-d.maxLength/2:]...)
				d.Times = append(d.Times[:0], d.Times[len(d.Times)-d.maxLength/2:]...)
			}
		}
	}
}

func (d *DataLine) Length() int {
	return len(d.Values)
}

func (d *DataLine) GetData(index int) (DatalineUnit, bool) {
	if index >= 0 && index < len(d.Values) {
		return DatalineUnit{V: d.Values[index], MS: d.Times[index]}, true
	} else {
		return DatalineUnit{}, false
	}
}

// index： 0, 1, 2, 3
// index：-4,-3,-2,-1
func (d *DataLine) GetValue(index int) (float64, bool) {
	if index < 0 {
		index = d.Length() + index
	}

	if index >= 0 && index < len(d.Values) {
		return d.Values[index], true
	} else {
		return 0, false
	}
}

func (d *DataLine) GetTime(index int) (int64, bool) {
	if index < 0 {
		index = d.Length() + index
	}

	if index >= 0 && index < len(d.Times) {
		return d.Times[index], true
	} else {
		// 越界时，返回推测出来的时间，但是ok为false
		if index < 0 {
			return d.Times[0] - d.intervalMs*(-int64(index)), false
		} else {
			return d.Times[0] + d.intervalMs*(int64(index)), false
		}
	}
}

func (d *DataLine) FirstValue() (float64, bool) {
	if len(d.Values) > 0 {
		return d.Values[0], true
	} else {
		return 0, false
	}
}

func (d *DataLine) FirstMS() (int64, bool) {
	if len(d.Times) > 0 {
		return d.Times[0], true
	} else {
		return 0, false
	}
}

func (d *DataLine) LastValue() (float64, bool) {
	if len(d.Values) > 0 {
		return d.Values[len(d.Values)-1], true
	} else {
		return 0, false
	}
}

func (d *DataLine) LastMS() (int64, bool) {
	if len(d.Times) > 0 {
		return d.Times[len(d.Times)-1], true
	} else {
		return 0, false
	}
}

func (d *DataLine) Save() {
	if len(d.filePath) == 0 {
		return
	}

	os.Remove(d.filePath)
	util.MakeSureDirForFile(d.filePath)
	if file, err := os.OpenFile(d.filePath, os.O_WRONLY|os.O_CREATE, 0666); err == nil {
		// 序列化
		buf := new(bytes.Buffer)
		l := d.Length()
		for i := 0; i < l; i++ {
			binary.Write(buf, binary.LittleEndian, d.Times[i])
			binary.Write(buf, binary.LittleEndian, d.Values[i])
		}

		// 保存
		file.Write(buf.Bytes())
		file.Close()
	}
}

func (d *DataLine) load() {
	if len(d.filePath) == 0 {
		return
	}

	if file, err := os.OpenFile(d.filePath, os.O_RDONLY, 0666); err == nil {
		// 读取
		fi, _ := file.Stat()
		sz := fi.Size()
		b := make([]byte, sz)
		file.Read(b)
		buf := bytes.NewBuffer(b)

		// 解析
		for {
			ms := int64(0)
			v := float64(0)
			err0 := binary.Read(buf, binary.LittleEndian, &ms)
			err1 := binary.Read(buf, binary.LittleEndian, &v)
			if err0 == nil && err1 == nil {
				d.Update(ms, v)
			} else {
				break
			}
		}
	}
}

// #endregion

// #region helper
// 计算最多N个数据的和
// 返回实际数量和实际总和
func (d *DataLine) SumLatest(index, maxCount int) (int, float64) {
	total := 0.0
	count := 0.0
	d.Traval(index, maxCount, func(v, w float64) {
		total += v * w
		count += w
	})

	return int(count + 0.1), total
}

// 从第index个元素开始向前遍历
// v：值，w：权重
func (d *DataLine) Traval(index, maxCount int, cb func(v, w float64)) {
	if index < 0 || index >= d.Length() {
		return
	}

	if d.Length() == 1 {
		v, _ := d.LastValue()
		cb(v, 1)
	} else {
		if index != d.Length()-1 {
			for i := 0; i < maxCount; i++ {
				j := index - i
				if j >= 0 {
					cb(d.Values[j], 1)
				} else {
					break
				}
			}
		} else {
			r := float64(d.Times[index]-d.Times[index-1]) / float64(d.intervalMs)
			for i := 0; i < maxCount; i++ {
				j := index - i
				if j >= 0 {
					if i == 0 {
						cb(d.Values[j], r)
					} else if i == maxCount-1 || j == 0 {
						cb(d.Values[j], 1-r)
					} else {
						cb(d.Values[j], 1)
					}
				} else {
					break
				}
			}
		}
	}
}

// 取尾部一段数据，最大长度l
func (d *DataLine) Tail(l int) ([]float64, []int64) {
	i := 0
	if l >= d.Length() {
		l = d.Length()
	} else {
		i = d.Length() - l
	}

	return d.Values[i:], d.Times[i:]
}

// 取尾部的平均值，最大长度l
func (d *DataLine) Avg(l int) float64 {
	vals, _ := d.Tail(l)
	if len(vals) == 0 {
		return 0
	} else {
		return floats.Sum(vals) / float64(len(vals))
	}
}

func (d *DataLine) SMA(timePeriod int) float64 {
	l := d.Length()
	if l >= timePeriod {
		marst := talib.Ma(d.Values[l-timePeriod:], timePeriod, talib.SMA)
		return marst[len(marst)-1]
	} else {
		marst := talib.Ma(d.Values, l, talib.SMA)
		return marst[len(marst)-1]
	}
}

func (d *DataLine) Boll(timePeriod int, k float64) (upper, middle, lower float64) {
	// 取尾部一段数据
	vals, _ := d.Tail(timePeriod)
	l := len(vals)
	if l == 0 {
		return 0, 0, 0
	}

	// 计算ma
	maval := stat.Mean(vals, nil)
	std := stat.StdDev(vals, nil)
	return maval + std*k, maval, maval - std*k
}

func (d *DataLine) Std(timePeriod int) float64 {
	// 取尾部一段数据
	vals, _ := d.Tail(timePeriod)
	l := len(vals)
	if l == 0 {
		return 0
	}

	// 计算std
	return stat.StdDev(vals, nil)
}

func (d *DataLine) String() string {
	return fmt.Sprintf("dataline len=%d, cap=%d, data=%v", len(d.Values), cap(d.Values), d.Values)
}

// #endregion
