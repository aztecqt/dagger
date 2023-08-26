/*
 * @Author: aztec
 * @Date: 2022-12-28 17:20:29
 * @Description: 简单的MA指标
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import (
	"aztecqt/dagger/stratergy"
)

type SMA struct {
	orign      *stratergy.DataLine
	value      *stratergy.DataLine
	n          int
	rebuilding bool
}

func NewSMA(orign *stratergy.DataLine, n int) *SMA {
	sma := new(SMA)
	sma.orign = orign
	sma.value = new(stratergy.DataLine)
	sma.value.Init("sma", orign.MaxLength(), orign.IntervalMS(), 0)
	sma.n = n
	return sma
}

func (s *SMA) Value() *stratergy.DataLine {
	return s.value
}

// ts, value, ok
func (s *SMA) calculate(index int) (int64, float64, bool) {
	if du, ok := s.orign.GetData(index); ok {
		ts := du.MS
		count, total := s.orign.SumLatest(index, s.n)
		return ts, total / float64(count), true
	}

	return 0, 0, false
}

func (s *SMA) Update() {
	if s.rebuilding {
		return
	}

	if ts, v, ok := s.calculate(s.orign.Length() - 1); ok {
		s.value.Update(ts, v)
	}
}

func (s *SMA) Rebuild() {
	s.rebuilding = true
	s.value.Clear()
	for i := 0; i < s.orign.Length(); i++ {
		if ts, v, ok := s.calculate(i); ok {
			s.value.Update(ts, v)
		}
	}
	s.rebuilding = false
}
