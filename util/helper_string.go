/*
 * @Author: aztec
 * @Date: 2023-08-20 15:55:50
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"aztecqt/dagger/util/logger"

	"github.com/emirpasic/gods/sets/treeset"
	"github.com/shopspring/decimal"
)

// 一个排序的字符串集合
func NewStringTreeSet() *treeset.Set {
	m := treeset.NewWith(func(a, b interface{}) int {
		sa := a.(string)
		sb := b.(string)
		return strings.Compare(sa, sb)
	})

	return m
}

// 判断字符串是否包含前缀
func StringStartWith(s, substr string) bool {
	return strings.Index(s, substr) == 0
}

// 判断字符串是否包含后缀
func StringEndWith(s, substr string) bool {
	return strings.Index(s, substr) == len(s)-len(substr)
}

// 获取s中pre之后、下个post之前的字符串
func FetchMiddleString(s *string, pre string, post string) string {
	i := strings.Index(*s, pre)
	if i >= 0 {
		i += len(pre)
		offset := strings.Index((*s)[i:], post)
		if offset >= 0 {
			return (*s)[i : i+offset]
		} else {
			return ""
		}
	} else {
		return ""
	}
}

// 字符串仅保留字母、数字
func ToLetterNumberOnly(orign string, limit int) string {
	converted := ""
	if len(orign) > 0 {
		bb := bytes.Buffer{}
		for _, c := range orign {
			if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
				bb.WriteRune(c)
			}
		}
		converted = bb.String()
		if limit > 0 && len(converted) > limit {
			converted = converted[0:limit]
		}
	}
	return converted
}

func IsIntNumber(s string) bool {
	_, ok := String2Int64(s)
	return ok
}

func String2Decimal(s string) (decimal.Decimal, bool) {
	if len(s) == 0 {
		return decimal.Zero, true
	}

	multi := DecimalOne
	lastChar := s[len(s)-1:]
	if lastChar == "K" || lastChar == "k" {
		multi = decimal.NewFromInt(1000)
	} else if lastChar == "M" || lastChar == "m" {
		multi = decimal.NewFromInt(1000000)
	} else if lastChar == "G" || lastChar == "g" {
		multi = decimal.NewFromInt(1000000000)
	} else if lastChar == "T" || lastChar == "t" {
		multi = decimal.NewFromInt(1000000000000)
	} else if lastChar == "P" || lastChar == "p" {
		multi = decimal.NewFromInt(1000000000000000)
	} else if lastChar == "E" || lastChar == "e" {
		multi = decimal.NewFromInt(1000000000000000000)
	} else if lastChar == "%" {
		multi = DecimalOne.Div(decimal.NewFromInt(100))
	}

	if multi != DecimalOne {
		s = s[:len(s)-1]
	}

	d, err := decimal.NewFromString(s)
	if err == nil {
		return d.Mul(multi), true
	} else {
		return decimal.Zero, false
	}
}

func String2DecimalPanic(s string) decimal.Decimal {
	d, ok := String2Decimal(s)
	if ok {
		return d
	} else {
		logger.LogPanic("", "can't parse [%s] to decimal", s)
		return decimal.Zero
	}
}

func String2DecimalPanicUnless(s, unless string) decimal.Decimal {
	if s == unless {
		return decimal.Zero
	}

	return String2DecimalPanic(s)
}

func HexString2Decimal(s string, dcm int32) (decimal.Decimal, bool) {
	bi := HexString2BigInt(s)
	if bi != nil {
		d := decimal.NewFromBigInt(bi, -dcm)
		return d, true
	} else {
		return decimal.Zero, false
	}
}

func String2Bool(s string) (bool, bool) {
	if s == "true" || s == "True" || s == "TRUE" {
		return true, true
	} else if s == "false" || s == "False" || s == "FALSE" {
		return false, true
	} else {
		return false, false
	}
}

func String2BoolPanic(s string) bool {
	b, ok := String2Bool(s)
	if ok {
		return b
	} else {
		logger.LogPanic("", "can't parse [%s] to bool", s)
		return false
	}
}

func String2Int(s string) (int, bool) {
	i, err := strconv.Atoi(s)
	if err == nil {
		return i, true
	} else {
		return 0, false
	}
}

func String2IntPanic(s string) int {
	i, ok := String2Int(s)
	if ok {
		return i
	} else {
		logger.LogPanic("", "can't parse [%s] to int", s)
		return 0
	}
}

func HexString2UInt64(s string) (uint64, bool) {
	numberStr := strings.Replace(s, "0x", "", -1)
	numberStr = strings.Replace(numberStr, "0X", "", -1)
	n, err := strconv.ParseUint(numberStr, 16, 64)
	if err != nil {
		return 0, false
	} else {
		return n, true
	}
}

func HexString2Int64(s string) (int64, bool) {
	numberStr := strings.Replace(s, "0x", "", -1)
	numberStr = strings.Replace(numberStr, "0X", "", -1)
	n, err := strconv.ParseInt(numberStr, 16, 64)
	if err != nil {
		return 0, false
	} else {
		return n, true
	}
}

func HexString2BigInt(s string) *big.Int {
	numberStr := strings.Replace(s, "0x", "", -1)
	numberStr = strings.Replace(numberStr, "0X", "", -1)
	if len(numberStr)%2 != 0 {
		numberStr = "0" + numberStr
	}
	byteValue, err := hex.DecodeString(numberStr)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return new(big.Int).SetBytes(byteValue)
}

func String2Int64(s string) (int64, bool) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return i, true
	} else {
		return 0, false
	}
}

func String2Int64Panic(s string) int64 {
	i, ok := String2Int64(s)
	if ok {
		return i
	} else {
		logger.LogPanic("", "can't parse [%s] to int64", s)
		return 0
	}
}

func String2UInt64(s string) (uint64, bool) {
	i, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		return i, true
	} else {
		return 0, false
	}
}

func String2UInt64Panic(s string) uint64 {
	i, ok := String2UInt64(s)
	if ok {
		return i
	} else {
		logger.LogPanic("", "can't parse [%s] to uint64", s)
		return 0
	}
}

func String2Float64(s string) (float64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	multi := 1.0
	lastChar := s[len(s)-1:]
	if lastChar == "K" || lastChar == "k" {
		multi = 1000
	} else if lastChar == "M" || lastChar == "m" {
		multi = 1000000
	} else if lastChar == "G" || lastChar == "g" {
		multi = 1000000000
	} else if lastChar == "T" || lastChar == "t" {
		multi = 1000000000000
	} else if lastChar == "P" || lastChar == "p" {
		multi = 1000000000000000
	} else if lastChar == "E" || lastChar == "e" {
		multi = 1000000000000000000
	} else if lastChar == "%" {
		multi = 0.01
	}

	if multi != 1 {
		s = s[:len(s)-1]
	}

	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f * multi, true
	} else {
		return 0, false
	}
}

func String2Float64Panic(s string) float64 {
	f, ok := String2Float64(s)
	if ok {
		return f
	} else {
		logger.LogPanic("", "can't parse [%s] to float64", s)
		return 0
	}
}

func PctString2Float64(s string) (float64, bool) {
	s = strings.Trim(s, " ")
	i := strings.LastIndexByte(s, '%')
	if i == len(s)-1 {
		s = s[:i]
		f, ok := String2Float64(s)
		return f / 100, ok
	} else {
		return 0, false
	}
}

func SplitString2IntSlice(s, sep string) ([]int, bool) {
	splited := strings.Split(s, sep)
	rst := make([]int, 0)

	for _, v := range splited {
		i, err := strconv.Atoi(v)
		if err == nil {
			rst = append(rst, i)
		} else {
			return nil, false
		}
	}

	return rst, true
}

func SplitString2DecimalSlice(s, sep string) ([]decimal.Decimal, bool) {
	splited := strings.Split(s, sep)
	rst := make([]decimal.Decimal, 0)

	for _, v := range splited {
		d, err := decimal.NewFromString(v)
		if err == nil {
			rst = append(rst, d)
		} else {
			return nil, false
		}
	}

	return rst, true
}
