/*
- @Author: aztec
- @Date: 2024-01-31 16:54:50
- @Description: go-pretty的常用功能封装
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package terminal

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// 创建默认table
func GenTableWriter(autoIndex bool) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetAutoIndex(autoIndex)
	return t
}

var TrackerStyleColorDefault = progress.StyleColors{
	Message: text.Colors{text.FgGreen, text.BgBlack},
	Percent: text.Colors{text.FgCyan, text.BgBlack},
	Stats:   text.Colors{text.FgYellow, text.BgBlack},
	Time:    text.Colors{text.FgGreen, text.BgBlack},
	Tracker: text.Colors{text.FgYellow, text.BgBlack},
	Value:   text.Colors{text.FgCyan, text.BgBlack},
}

// 创建默认进度条
func GenTracker(msg string, totalValue float64, len int, showTime, showValue bool) *TrackerF {
	pw := progress.NewWriter()
	pw.SetTrackerLength(len)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Options.TimeDonePrecision = time.Millisecond
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Colors = TrackerStyleColorDefault
	pw.Style().Visibility.Time = showTime
	pw.Style().Visibility.Value = showValue
	tracker := newTrackerF(msg, totalValue)
	pw.AppendTracker(&tracker.Tracker)
	go pw.Render()
	return tracker
}

// 创建带当前机器硬件状态的进度条
func GenTrackerWithHardwareInfo(msg string, totalValue float64, len int, showTime, showValue bool, cpu, mem, netio bool) *TrackerF {
	pw := progress.NewWriter()
	pw.SetTrackerLength(len)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Options.TimeDonePrecision = time.Millisecond
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Colors = TrackerStyleColorDefault
	pw.Style().Visibility.Time = showTime
	pw.Style().Visibility.Value = showValue
	tracker := newTrackerF(msg, totalValue)
	pw.AppendTracker(&tracker.Tracker)

	go func() {
		for !tracker.IsDone() {
			msgCpu := ""
			msgMem := ""
			msgNet := ""

			if cpu {
				msgCpu = fmt.Sprintf("[cpu:%4.1f%%]", util.GetCpuUsage(time.Second))
			}

			if mem {
				_, pct := util.GetMemInfo()
				msgMem = fmt.Sprintf("[mem:%4.1f%%]", pct)
			}

			if netio {
				send, recv := util.GetnetIoSpeedStr()
				msgNet = fmt.Sprintf("[send:%s][recv:%s]", send, recv)
			}

			tracker.Message = fmt.Sprintf("%s%s%s%s", msg, msgCpu, msgMem, msgNet)
		}
	}()

	go pw.Render()
	return tracker
}

type TrackerF struct {
	progress.Tracker
	value float64
	total float64
}

func newTrackerF(msg string, total float64) *TrackerF {
	t := &TrackerF{Tracker: progress.Tracker{Message: msg, Total: int64(total * 10000)}, total: total}
	return t
}

func (t *TrackerF) SetValue(val float64) {
	t.value = val
	nValue := int64(val * 10000)
	t.Tracker.SetValue(nValue)
}

func (t *TrackerF) Increment(val float64) {
	t.value += val

	if t.value < 0 {
		t.value = 0
	}

	if t.value > t.total {
		t.value = t.total
	}

	nValue := int64(t.value * 10000)
	t.Tracker.SetValue(nValue)
}
