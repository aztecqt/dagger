/*
 * @Author: aztec
 * @Date: 2022-12-29
 * @Description: 加权平均计算器
 */

package mathtools

import "gonum.org/v1/gonum/stat"

type Meaner struct {
	values       []float64
	weights      []float64
	totalWeights float64
}

func (m *Meaner) AddValue(v, w float64) {
	m.values = append(m.values, v)
	m.weights = append(m.weights, w)
	m.totalWeights += w
}

func (m *Meaner) Mean() float64 {
	if m.totalWeights == 0 {
		return 0
	} else {
		return stat.Mean(m.values, m.weights)
	}
}
