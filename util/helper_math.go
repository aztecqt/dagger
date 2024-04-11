/*
 * @Author: aztec
 * @Date: 2023-09-03 15:14:16
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/shopspring/decimal"
)

var DecimalOne = decimal.NewFromInt(1)
var DecimalTwo = decimal.NewFromInt(2)
var DecimalTen = decimal.NewFromInt(10)
var Decimal100 = decimal.NewFromInt(100)
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

func AbsInt64(a int64) int64 {
	if a >= 0 {
		return a
	} else {
		return -a
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

func AbsInt(a int) int {
	if a >= 0 {
		return a
	} else {
		return -a
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

// 获取decimal的最小精度
func DecimalTick(d decimal.Decimal) decimal.Decimal {
	ticker := decimal.New(1, d.Exponent())
	return ticker
}

// 对齐decimal到指定精度
func AlignDecimal(d decimal.Decimal, tick decimal.Decimal, alignUp bool) decimal.Decimal {
	aligned := tick.Mul(decimal.NewFromInt(d.Div(tick).IntPart()))
	if alignUp && !aligned.Equal(d) {
		aligned = aligned.Add(tick)
	}
	return aligned
}

// 获取float的最小精度
func FloatTick(f float64) float64 {
	d := decimal.NewFromFloat(f)
	t := DecimalTick(d)
	return t.InexactFloat64()
}

// 对齐float到指定精度
func AlignFloat(val float64, tick float64, alignUp bool) float64 {
	aligned := 0.0

	if alignUp {
		aligned = math.Ceil((val+tick/10)/tick) * tick
	} else {
		aligned = math.Floor((val+tick/10)/tick)*tick + tick/2
	}

	str := fmt.Sprintf("%v", aligned)
	dot := strings.Index(str, ".")
	d := int(math.Log10(tick))
	pos := dot - d + 2
	l := len(str)
	if l <= pos {
		return aligned
	} else {
		cnext := str[pos]
		str = str[:pos]
		if v, err := strconv.ParseFloat(str, 64); err == nil {
			aligned = v
		}

		if cnext == '9' {
			return aligned + tick
		} else {
			return aligned
		}
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

func LerpInt(a, b int, r float64) int {
	delta := b - a
	return int(a + int(float64(delta)*r))
}

func LerpInt64(a, b int64, r float64) int64 {
	delta := b - a
	return int64(a + int64(float64(delta)*r))
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

// 对于一个单调函数fn，给定初始值y、目标值、判定函数，求出x
// stepx必须为正数
// 返回值：x,y,count
func Approach(fn func(x float64) float64, initX, stepX, minX, maxX, targetY, minDiff, maxDiff float64, maxTry int) (float64, float64, int, bool) {
	if stepX <= 0 {
		panic("stepx must positive")
	}

	if minDiff >= maxDiff {
		panic("minDiff must < maxDiff")
	}

	x := initX
	y := fn(x)
	lessLast := y < targetY

	// 计算第一个step
	yr := fn(initX + stepX)
	if targetY > y && yr < y || targetY < y && yr > y {
		stepX = -stepX
	}

	passCount := 0
	tryCount := 0
	for {
		// 移动
		x = ClampFloat(x+stepX, minX, maxX)
		y = fn(x)
		tryCount++
		if tryCount >= maxTry {
			return x, y, tryCount, false
		}

		diff := y - targetY

		// fmt.Printf("x move to %v with step %v, y=%v, diff=%v\n", x, stepX, y, diff)
		// 检查结果是否达标
		if diff < maxDiff && diff > minDiff {
			return x, y, tryCount, true
		}

		// 未达标
		// 如果移动之后仍然处于同侧，则加倍移动
		// 如果移动后处于不同侧，反向移动一半
		lessNow := y < targetY
		if lessNow == lessLast {
			if passCount == 0 {
				stepX = stepX * 2
			}
		} else {
			passCount++
			stepX = -stepX / 2
		}
		lessLast = lessNow
	}
}

// #region 统计学算子
// Spearman相关系数
func SpearmanCorr(x, y []float64) float64 {
	// 创建一个映射表，用于保存排序后的等级
	rankMapX := make(map[float64]int)
	rankMapY := make(map[float64]int)

	// 将数据排序并保存等级
	sortedX := make([]float64, len(x))
	copy(sortedX, x)
	sort.Float64s(sortedX)
	for i, val := range sortedX {
		rankMapX[val] = i + 1
	}

	sortedY := make([]float64, len(y))
	copy(sortedY, y)
	sort.Float64s(sortedY)
	for i, val := range sortedY {
		rankMapY[val] = i + 1
	}

	// 计算等级差
	var rankDiffSum float64
	for i := range x {
		rankDiff := float64(rankMapX[x[i]] - rankMapY[y[i]])
		rankDiffSum += rankDiff * rankDiff
	}

	// 计算Spearman相关系数
	n := float64(len(x))
	spearman := 1 - (6*rankDiffSum)/(n*(n*n-1))

	return spearman
}

// #endregion
