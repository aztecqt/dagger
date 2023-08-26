/*
 * @Author: aztec
 * @Date: 2022-12-30 18:54:49
 * @Description: 一段时间内的最大最小值
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import (
	"math"

	"aztecqt/dagger/stratergy"
)

type MinMax struct {
	orign      *stratergy.DataLine
	min        *stratergy.DataLine
	max        *stratergy.DataLine
	n          int
	rebuilding bool
}

func NewMinMax(orign *stratergy.DataLine, n int) *MinMax {
	mm := new(MinMax)
	mm.orign = orign
	mm.min = new(stratergy.DataLine)
	mm.min.Init("min", orign.MaxLength(), orign.IntervalMS(), 0)
	mm.max = new(stratergy.DataLine)
	mm.max.Init("max", orign.MaxLength(), orign.IntervalMS(), 0)
	mm.n = n
	return mm
}

func (s *MinMax) Min() *stratergy.DataLine {
	return s.min
}

func (s *MinMax) Max() *stratergy.DataLine {
	return s.max
}

// ts, min, max, ok
func (s *MinMax) calculate(index int) (int64, float64, float64, bool) {
	min := math.MaxFloat32
	max := -math.MaxFloat32
	if du, ok := s.orign.GetData(index); ok {
		ts := du.MS
		s.orign.Traval(index, s.n, func(v, w float64) {
			if v > max {
				max = v
			}

			if v < min {
				min = v
			}
		})

		return ts, min, max, true
	}

	return 0, 0, 0, false
}

func (s *MinMax) Update() {
	if s.rebuilding {
		return
	}

	if ts, min, max, ok := s.calculate(s.orign.Length() - 1); ok {
		s.min.Update(ts, min)
		s.max.Update(ts, max)
	}
}

func (s *MinMax) Rebuild() {
	s.rebuilding = true
	s.min.Clear()
	s.max.Clear()
	for i := 0; i < s.orign.Length(); i++ {
		if ts, min, max, ok := s.calculate(i); ok {
			s.min.Update(ts, min)
			s.max.Update(ts, max)
		}
	}
	s.rebuilding = false
}
