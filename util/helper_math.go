/*
 * @Author: aztec
 * @Date: 2023-09-03 15:14:16
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"math"
	"math/rand"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/shopspring/decimal"
)

var DecimalOne = decimal.NewFromInt(1)
var DecimalNegOne = decimal.NewFromInt(-1)

type Relation int

const (
	Relation_Equal = iota
	Relation_GreaterThan
	Relation_GreaterThanOrEqual
	Relation_LessThan
	Relation_LessThanOrEqual
)

func Relation2Str(r Relation) string {
	switch r {
	case Relation_Equal:
		return "=="
	case Relation_GreaterThan:
		return ">"
	case Relation_GreaterThanOrEqual:
		return ">="
	case Relation_LessThan:
		return "<"
	case Relation_LessThanOrEqual:
		return "<="
	default:
		return "??"
	}
}

func NewDecimalTreeMap() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		da := a.(decimal.Decimal)
		db := b.(decimal.Decimal)
		if da.GreaterThan(db) {
			return 1
		} else if db.GreaterThan(da) {
			return -1
		} else {
			return 0
		}
	})

	return m
}

func NewDecimalTreeMapInverted() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		da := a.(decimal.Decimal)
		db := b.(decimal.Decimal)
		if da.GreaterThan(db) {
			return -1
		} else if db.GreaterThan(da) {
			return 1
		} else {
			return 0
		}
	})

	return m
}

func NewInt64TreeMap() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		ia := a.(int64)
		ib := b.(int64)
		if ia > ib {
			return 1
		} else if ia < ib {
			return -1
		} else {
			return 0
		}
	})

	return m
}

func NewInt64TreeMapInverted() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		ia := a.(int64)
		ib := b.(int64)
		if ia > ib {
			return -1
		} else if ia < ib {
			return 1
		} else {
			return 0
		}
	})

	return m
}

func NewFloatTreeMap() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		ia := a.(float64)
		ib := b.(float64)
		if ia > ib {
			return 1
		} else if ia < ib {
			return -1
		} else {
			return 0
		}
	})

	return m
}

func NewFloatTreeMapInverted() *treemap.Map {
	m := treemap.NewWith(func(a, b interface{}) int {
		ia := a.(float64)
		ib := b.(float64)
		if ia > ib {
			return -1
		} else if ia < ib {
			return 1
		} else {
			return 0
		}
	})

	return m
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	} else {
		return b
	}
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func MinInt(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func MinDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	} else {
		return b
	}
}

func MaxDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThan(b) {
		return a
	} else {
		return b
	}
}

func RandomIntBetweenFloat(f1, f2 float64) int {
	if f1 == float64(int(f1)) && f2 == float64(int(f2)) {
		return int(f1) + rand.Intn(int(f2)-int(f1)+1)
	} else {
		f := f1 + rand.Float64()*(f2-f1)
		n := int(f)
		fpost := f - float64(n)
		if rand.Float64() < fpost {
			return n
		} else {
			return n + 1
		}
	}
}

func RandomInt(n0, n1 int) int {
	if n1 > n0 {
		return n0 + rand.Intn(n1-n0+1)
	} else {
		return n0 + rand.Intn(n0-n1+1)
	}
}

func RandomInt64(n0, n1 int64) int64 {
	if n1 > n0 {
		return n0 + rand.Int63n(n1-n0+1)
	} else {
		return n0 + rand.Int63n(n0-n1+1)
	}
}

// 计算a相对于b的偏差。如105对100的偏差就是0.05
func DecimalDeviation(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return DecimalOne
	} else {
		return a.Div(b).Sub(decimal.NewFromInt(1))
	}
}

func DecimalDeviationAbs(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return DecimalOne
	} else {
		return a.Div(b).Sub(decimal.NewFromInt(1)).Abs()
	}
}

// 计算a相对于b的偏差。如105对100的偏差就是0.05
func FloatDeviation(a, b float64) float64 {
	if b == 0 {
		return 1
	} else {
		return a/b - 1
	}
}

func FloatDeviationAbs(a, b float64) float64 {
	if b == 0 {
		return 1
	} else {
		return math.Abs(a/b - 1)
	}
}

func ClampDecimal(v, a, b decimal.Decimal) decimal.Decimal {
	if b.GreaterThan(a) {
		if v.LessThan(a) {
			v = a
		} else if v.GreaterThan(b) {
			v = b
		}
	} else {
		if v.LessThan(b) {
			v = b
		} else if v.GreaterThan(a) {
			v = a
		}
	}

	return v
}

func ClampFloat(v, a, b float64) float64 {
	if b > a {
		if v < a {
			v = a
		} else if v > b {
			v = b
		}
	} else {
		if v < b {
			v = b
		} else if v > a {
			v = a
		}
	}

	return v
}

func ClampInt(v, a, b int) int {
	if b > a {
		if v < a {
			v = a
		} else if v > b {
			v = b
		}
	} else {
		if v < b {
			v = b
		} else if v > a {
			v = a
		}
	}

	return v
}

func LerpDecimal(a, b, r decimal.Decimal) decimal.Decimal {
	delta := b.Sub(a)
	return a.Add(delta.Mul(r))
}

func LerpFloat(a, b, r float64) float64 {
	delta := b - a
	return a + delta*r
}

// 最大回撤
func MaxRetracement(slice []float64) float64 {
	l := len(slice)
	max := float64(math.MinInt32)
	maxr := float64(0)
	for i := 0; i < l; i++ {
		v := slice[i]
		if v > max {
			max = slice[i]
		}

		if v < max {
			if max-v > maxr {
				maxr = max - v
			}
		}
	}

	return maxr
}

func TotalRetracement(slice []float64) float64 {
	l := len(slice)
	total := 0.0
	for i := 1; i < l; i++ {
		v0 := slice[i-1]
		v1 := slice[i]
		if v1 < v0 {
			total += v0 - v1
		}
	}

	return total
}
