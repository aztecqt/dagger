/*
 * @Author: aztec
 * @Date: 2022-12-29 15:10:57
 * @LastEditors: aztec
 * @Description: 方差
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import (
	"math"

	"aztecqt/dagger/stratergy"
)

type StdDev struct {
	orign      *stratergy.DataLine
	value      *stratergy.DataLine
	n          int
	rebuilding bool
}

func NewStdDev(orign *stratergy.DataLine, n int) *StdDev {
	stddev := new(StdDev)
	stddev.orign = orign
	stddev.value = new(stratergy.DataLine)
	stddev.value.Init("stddev", orign.MaxLength(), orign.IntervalMS(), 0)
	stddev.n = n
	return stddev
}

func (s *StdDev) Value() *stratergy.DataLine {
	return s.value
}

// ts, value, ok
func (s *StdDev) calculate(index int) (int64, float64, bool) {
	if du, ok := s.orign.GetData(index); ok {
		ts := du.MS
		count, total := s.orign.SumLatest(index, s.n)
		avg := total / float64(count)

		total = 0.0
		totalW := 0.0

		s.orign.Traval(index, s.n, func(v, w float64) {
			delta := v - avg
			total += delta * delta * w
			totalW += w
		})

		temp := total / totalW
		stddev := math.Sqrt(temp)
		return ts, stddev, true
	}

	return 0, 0, false
}

func (s *StdDev) Update() {
	if s.rebuilding {
		return
	}

	if ts, v, ok := s.calculate(s.orign.Length() - 1); ok {
		s.value.Update(ts, v)
	}
}

func (s *StdDev) Rebuild() {
	s.rebuilding = true
	s.value.Clear()
	for i := 0; i < s.orign.Length(); i++ {
		if ts, v, ok := s.calculate(i); ok {
			s.value.Update(ts, v)
		}
	}
	s.rebuilding = false
}
