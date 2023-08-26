/*
 * @Author: aztec
 * @Date: 2022-12-30 20:10:18
 * @Description: 根据近期最大最小值，乘以时间加权，获得的一条通道
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import (
	"aztecqt/dagger/stratergy"
)

type MmwBand struct {
	orign      *stratergy.DataLine
	lower      *stratergy.DataLine
	upper      *stratergy.DataLine
	mm         *MinMax
	ma         *SMA
	n          int
	n1         int
	rebuilding bool
}

func NewMmwBand(orign *stratergy.DataLine, n, n1 int) *MmwBand {
	mmwband := new(MmwBand)
	mmwband.orign = orign
	mmwband.lower = new(stratergy.DataLine)
	mmwband.upper = new(stratergy.DataLine)
	mmwband.lower.Init("middle", orign.MaxLength(), orign.IntervalMS(), 0)
	mmwband.upper.Init("upper", orign.MaxLength(), orign.IntervalMS(), 0)
	mmwband.mm = NewMinMax(orign, n1)
	mmwband.ma = NewSMA(orign, n)
	mmwband.n = n
	mmwband.n1 = n1
	return mmwband
}

func (s *MmwBand) Upper() *stratergy.DataLine {
	return s.upper
}

func (s *MmwBand) Lower() *stratergy.DataLine {
	return s.lower
}

func (s *MmwBand) Middle() *stratergy.DataLine {
	return s.ma.value
}

// ts, u, l, ok
func (s *MmwBand) calculate(index int) (int64, float64, float64, bool) {
	s.mm.calculate(index)
	s.ma.calculate(index)

	if du, ok := s.orign.GetData(index); ok {
		ts := du.MS
		totalMin := 0.0
		totalMax := 0.0
		totalW := 0.0

		sampleCount := index
		if sampleCount > s.n {
			sampleCount = s.n
		}
		seg := 1 + sampleCount/s.n1
		for i := 0; i < seg; i++ {
			realIndex := index - seg*i
			duMin, _ := s.mm.min.GetData(realIndex)
			duMax, _ := s.mm.max.GetData(realIndex)
			w := float64(seg-i) / float64(seg)
			totalMin += duMin.V * w
			totalMax += duMax.V * w
			totalW += w
		}

		u := totalMax / totalW
		l := totalMin / totalW
		return ts, u, l, true
	}

	return 0, 0, 0, false
}

func (s *MmwBand) Update() {
	if s.rebuilding {
		return
	}

	s.mm.Update()
	s.ma.Update()

	if ts, u, l, ok := s.calculate(s.orign.Length() - 1); ok {
		s.upper.Update(ts, u)
		s.lower.Update(ts, l)
	}
}

func (s *MmwBand) Rebuild() {
	s.rebuilding = true
	s.mm.Rebuild()
	s.ma.Rebuild()
	s.lower.Clear()
	s.upper.Clear()
	for i := 0; i < s.orign.Length(); i++ {
		if ts, u, l, ok := s.calculate(i); ok {
			s.upper.Update(ts, u)
			s.lower.Update(ts, l)
		}
	}
	s.rebuilding = false
}
