/*
 * @Author: aztec
 * @Date: 2023-09-03 15:13:35
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"
)

// 东八区
var East8 = time.FixedZone("CCT", 8*3600)

// 美国东部时间
var UsEastern = time.FixedZone("EST", -5*3600)

func MinTime(tms ...time.Time) time.Time {
	if len(tms) == 0 {
		return time.Time{}
	}

	min := tms[0]
	for i := 1; i < len(tms); i++ {
		if tms[i].Before(min) {
			min = tms[i]
		}
	}
	return min
}

func MaxTime(tms ...time.Time) time.Time {
	if len(tms) == 0 {
		return time.Time{}
	}

	max := tms[0]
	for i := 1; i < len(tms); i++ {
		if tms[i].After(max) {
			max = tms[i]
		}
	}
	return max
}

func HourOfTime(t time.Time) time.Time {
	y, m, d := t.Date()
	h := t.Hour()
	name, _ := t.Zone()
	str := fmt.Sprintf("%04d-%02d-%02d %02d:00:00 %s", y, m, d, h, name)
	hour, e := time.Parse("2006-01-02 15:04:05 MST", str)
	if e != nil {
		fmt.Println(e.Error())
	}
	return hour
}

func TimeOfDay(t time.Time, h, m, s int) time.Time {
	y, mo, d := t.Date()
	name, _ := t.Zone()
	str := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d %s", y, mo, d, h, m, s, name)
	tRst, e := time.Parse("2006-01-02 15:04:05 MST", str)
	if e != nil {
		fmt.Println(e.Error())
	}
	return tRst
}

// 是否是同一小时（忽略时区）
func IsSameHour(t0, t1 time.Time) bool {
	y0, m0, d0 := t0.Date()
	y1, m1, d1 := t1.Date()
	h0 := t0.Hour()
	h1 := t1.Hour()
	return y0 == y1 && m0 == m1 && d0 == d1 && h0 == h1
}

func DateOfTime(t time.Time) time.Time {
	str := t.String()
	i0 := 11
	i1 := strings.Index(str[i0:], " ") + i0
	str = strings.Replace(str, str[i0:i1], "00:00:00", 1)
	if i := strings.Index(str, " m="); i > 0 {
		str = str[:i]
	}
	date, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", str)
	return date
}

// 是否同一天（忽略时区）
func IsSameDate(t0, t1 time.Time) bool {
	y0, m0, d0 := t0.Date()
	y1, m1, d1 := t1.Date()
	return y0 == y1 && m0 == m1 && d0 == d1
}

func MonthOfTime(t time.Time) time.Time {
	y, m, _ := t.Date()
	name, _ := t.Zone()
	str := fmt.Sprintf("%04d-%02d-%02d 00:00:00 %s", y, m, 1, name)
	date, e := time.Parse("2006-01-02 15:04:05 MST", str)
	if e != nil {
		fmt.Println(e.Error())
	}
	return date
}

func TimeNowUnix13() int64 {
	return time.Now().UnixNano() / 1e6
}

func ConvetUnix13ToTime(u13 int64) time.Time {
	return time.Unix(0, u13*1000000)
}

func ConvetUnix13StrToTime(u13str string) (time.Time, bool) {
	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000), true
	} else {
		return time.Unix(0, 0), false
	}
}

func ConvetUnix13StrToTimePanic(u13str string) time.Time {
	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000)
	} else {
		logger.LogPanic("", "can't parse [%s] to Unix13", u13str)
		return time.Unix(0, 0)
	}
}

func ConvetUnix13StrToTimePanicUnless(u13str string, unless string) time.Time {
	if u13str == unless {
		return time.Unix(0, 0)
	}

	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000)
	} else {
		logger.LogPanic("", "can't parse [%s] to Unix13", u13str)
		return time.Unix(0, 0)
	}
}

func ConvetUnix13ToIsoTime(u13 int64) string {
	utcTime := ConvetUnix13ToTime(u13).UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:19]) + "Z"
	return iso
}

func Duration2Str(dur time.Duration) string {
	if dur.Hours() > 24 {
		d := int(dur.Hours() / 24)
		h := int(dur.Hours()) - d*24
		return fmt.Sprintf("%dday %dhour", d, h)
	} else {
		h := int(dur.Hours())
		m := int(dur.Minutes()) - h*60
		s := int(dur.Seconds()) - h*3600 - m*60
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
}

func Duration2StrCn(dur time.Duration) string {
	if dur.Hours() > 24 {
		d := int(dur.Hours() / 24)
		h := int(dur.Hours()) - d*24
		return fmt.Sprintf("%d天 %d小时", d, h)
	} else {
		h := int(dur.Hours())
		m := int(dur.Minutes()) - h*60
		s := int(dur.Seconds()) - h*3600 - m*60
		return fmt.Sprintf("%d时 %d分 %d秒", h, m, s)
	}
}

// 1s=1
// 2m=120
// 3h=3600x3
// 4d....
func DurationStr2Seconds(str string) (sec int64) {
	sec = 0
	defer DefaultRecover()
	if len(str) > 1 {
		unit := str[len(str)-1]
		if num, numok := String2Int64(str[:len(str)-1]); numok {
			switch unit {
			case 's':
				sec = num
			case 'm':
				sec = num * 60
			case 'h':
				sec = num * 3600
			case 'd':
				sec = num * 3600 * 24
			default:
				sec = 0
			}
		}
	}
	return
}

// 1s
// 1m
// 1h
// 1d
// 1w
func String2Duration(s string) (time.Duration, error) {
	if len(s) > 0 {
		c := s[len(s)-1]
		n, nok := String2Int(s[0 : len(s)-1])
		if !nok {
			return 0, errors.New("invalid number format")
		}
		switch c {
		case 's':
			return time.Second * time.Duration(n), nil
		case 'm':
			return time.Minute * time.Duration(n), nil
		case 'h':
			return time.Hour * time.Duration(n), nil
		case 'd':
			return time.Hour * time.Duration(n*24), nil
		case 'w':
			return time.Hour * time.Duration(n*24*7), nil
		default:
			return time.Duration(0), errors.New("invalid format")
		}
	} else {
		return time.Duration(0), errors.New("empty string")
	}
}

// 对齐时间
// 将当前时间向下对齐到最近的整数倍interval
func AlignTime(t time.Time, intervalMs int64) time.Time {
	return time.UnixMilli((t.UnixMilli() / intervalMs) * intervalMs)
}

// 计算时间开销
type TimeConsumor struct {
	record map[string]time.Time
	fnLog  func(str string)
}

func NewTimeConsumor(fnLog func(str string)) *TimeConsumor {
	t := &TimeConsumor{
		record: make(map[string]time.Time),
		fnLog:  fnLog,
	}

	if fnLog == nil {
		panic("mush have fnLog")
	}

	return t
}

func (t *TimeConsumor) Record0(key string) {
	t.record[key] = time.Now()
}

func (t *TimeConsumor) Record1(key string) {
	if t0, ok := t.record[key]; ok {
		ns := time.Now().Sub(t0).Nanoseconds()
		t.fnLog(fmt.Sprintf("[TimeComsumor] [%s] cost %.4fms", key, float64(ns)/1e6))
	}
}
