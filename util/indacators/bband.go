/*
 * @Author: aztec
 * @Date: 2022-12-29 15:22:28
 * @Description: 布林带
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import "github.com/aztecqt/dagger/framework"

type BBand struct {
	orign      *framework.DataLine
	middle     *framework.DataLine
	upper      *framework.DataLine
	lower      *framework.DataLine
	sma        *SMA
	stddev     *StdDev
	stddevma   *SMA
	nMA        int
	nStdDev    float64
	rebuilding bool
}

func NewBBand(orign *framework.DataLine, nMA int, nStdDev float64) *BBand {
	bband := new(BBand)
	bband.orign = orign
	bband.middle = new(framework.DataLine)
	bband.upper = new(framework.DataLine)
	bband.lower = new(framework.DataLine)
	bband.middle.Init("middle", orign.MaxLength(), orign.IntervalMS(), 0)
	bband.upper.Init("upper", orign.MaxLength(), orign.IntervalMS(), 0)
	bband.lower.Init("lower", orign.MaxLength(), orign.IntervalMS(), 0)
	bband.nMA = nMA
	bband.nStdDev = nStdDev
	bband.sma = NewSMA(orign, nMA)
	bband.stddev = NewStdDev(orign, nMA)
	bband.stddevma = NewSMA(bband.stddev.value, 5)

	return bband
}

func (s *BBand) Middle() *framework.DataLine {
	return s.middle
}

func (s *BBand) Upper() *framework.DataLine {
	return s.upper
}

func (s *BBand) Lower() *framework.DataLine {
	return s.lower
}

// ts, m, u, l, ok
func (s *BBand) calculate(index int) (int64, float64, float64, float64, bool) {
	if ts, m, ok := s.sma.calculate(index); ok {
		if _, stddev, ok := s.stddev.calculate(index); ok {
			if _, _, ok := s.stddevma.calculate(index); ok {
				u := m + stddev*s.nStdDev
				l := m - stddev*s.nStdDev
				return ts, m, u, l, true
			}
		}
	}

	return 0, 0, 0, 0, false
}

func (s *BBand) Update() {
	if s.rebuilding {
		return
	}

	s.sma.Update()
	s.stddev.Update()
	s.stddevma.Update()

	if ts, m, u, l, ok := s.calculate(s.orign.Length() - 1); ok {
		s.middle.Update(ts, m)
		s.upper.Update(ts, u)
		s.lower.Update(ts, l)
	}
}

func (s *BBand) Rebuild() {
	s.rebuilding = true
	s.sma.Rebuild()
	s.stddev.Rebuild()
	s.stddevma.Rebuild()
	s.middle.Clear()
	s.upper.Clear()
	s.lower.Clear()
	for i := 0; i < s.orign.Length(); i++ {
		if ts, m, u, l, ok := s.calculate(i); ok {
			s.middle.Update(ts, m)
			s.upper.Update(ts, u)
			s.lower.Update(ts, l)
		}
	}
	s.rebuilding = false
}
