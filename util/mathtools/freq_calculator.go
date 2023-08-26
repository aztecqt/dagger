/*
 * @Author: aztec
 * @Date: 2022-10-28 10:40:36
 * @Description:
 * 频率计算器
 * 第一种：计算一段时间内的平均频率（平均次/秒）
 * 第二种：计算一个时间窗口内的事件发生次数（实际次/时间窗口）
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package mathtools

import "time"

type FreqCalculatorAvg struct {
	data []dataRecord
}

type dataRecord struct {
	tm    time.Time
	count int
}

// 以总数量（递增）的形式喂入数据，返回数据频率。数据单位为次/秒，外部可以自己再换算成其他单位
func (f *FreqCalculatorAvg) FeedTotalCount(count int, dur time.Duration) float64 {
	if f.data == nil {
		f.data = make([]dataRecord, 0)
	}

	now := time.Now()
	f.data = append(f.data, dataRecord{tm: now, count: count})

	// 反向遍历，取出指定时间内的所有数据
	total := 0
	var realDur time.Duration
	for i := len(f.data) - 1; i >= 0; i-- {
		total = count - f.data[i].count
		realDur = now.Sub(f.data[i].tm)
		if realDur >= dur {
			f.data = f.data[i:]
			break
		}
	}

	if dur.Seconds() > 0 {
		return float64(total) / realDur.Seconds()
	} else {
		return 0
	}
}

type FreqCalculatorTimeWindow struct {
	timeRecord []time.Time
	Freq       int
}

func (f *FreqCalculatorTimeWindow) Feed(dur time.Duration) int {
	f.timeRecord = append(f.timeRecord, time.Now())

	// 反向遍历，取出指定时间内的所有数据
	total := 0
	now := time.Now()
	for i := len(f.timeRecord) - 1; i >= 0; i-- {
		durReal := now.Sub(f.timeRecord[i])
		if durReal < dur {
			total++
		} else {
			f.timeRecord = f.timeRecord[i:]
			break
		}
	}
	f.Freq = total
	return f.Freq
}
