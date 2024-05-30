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
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/aztecqt/dagger/util/logger"

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
	a := strings.Index(s, substr)
	if a >= 0 {
		b := len(s) - len(substr)
		return a == b
	} else {
		return false
	}
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
	if len(s) == 0 {
		return decimal.Zero, true
	}

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
	} else if s == "false" || s == "False" || s == "FALSE" || s == "" {
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
	if len(s) == 0 {
		return 0, true
	}

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
	if len(s) == 0 {
		return 0, true
	}

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
	if len(s) == 0 {
		return 0, true
	}

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
	if len(s) == 0 {
		return new(big.Int)
	}

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
	if len(s) == 0 {
		return 0, true
	}

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

func String2Int32(s string) (int32, bool) {
	if len(s) == 0 {
		return 0, true
	}

	i, err := strconv.ParseInt(s, 10, 32)
	if err == nil {
		return int32(i), true
	} else {
		return 0, false
	}
}

func String2Int32Panic(s string) int32 {
	i, ok := String2Int32(s)
	if ok {
		return i
	} else {
		logger.LogPanic("", "can't parse [%s] to int64", s)
		return 0
	}
}

func String2UInt64(s string) (uint64, bool) {
	if len(s) == 0 {
		return 0, true
	}

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
		return 0, true
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

// 转换成带K、M、G、T、P、E的短文本
// d为保留几位小数点
func Float2ShortString(f float64, d int) string {
	postFix := ""
	multi := 1.0
	fabs := math.Abs(f)
	if fabs > 1e18 {
		postFix = "E"
		multi = 1e18
	} else if fabs > 1e15 {
		postFix = "P"
		multi = 1e15
	} else if fabs > 1e12 {
		postFix = "T"
		multi = 1e12
	} else if fabs > 1e9 {
		postFix = "G"
		multi = 1e9
	} else if fabs > 1e6 {
		postFix = "M"
		multi = 1e6
	} else if fabs > 1e3 {
		postFix = "K"
		multi = 1e3
	}

	format := fmt.Sprintf("%%.%df%%s", d)
	return fmt.Sprintf(format, f/multi, postFix)
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
	if len(s) == 0 {
		return 0, true
	}

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

func Object2String(obj interface{}) string {
	b, _ := json.MarshalIndent(obj, "", "  ")
	return string(b)
}

func Object2StringWithoutIntent(obj interface{}) string {
	b, _ := json.Marshal(obj)
	return string(b)
}

func ObjectFromString(str string, obj interface{}) error {
	return json.Unmarshal([]byte(str), obj)
}

func FormatJsonStr(str string) string {
	obj := make(map[string]interface{})
	json.Unmarshal([]byte(str), &obj)
	b, _ := json.MarshalIndent(obj, "", "  ")
	return string(b)
}

// 从str中，找出after之后的第一个json对象（还没测过）
func ExtractJsonAfter(str, after string) string {
	index := strings.Index(str, after)
	return ExtractJsonAfterIndex(str, index)
}

func ExtractJsonAfterIndex(str string, index int) string {
	sb := strings.Builder{}
	if index >= 0 {
		count := 0
		for i := index; i < len(str); i++ {
			char := str[i]

			if count > 0 {
				if sb.Len() == 0 {
					sb.WriteByte('{')
				}
				sb.WriteByte(char)
			}

			if char == '{' {
				count++
			} else if char == '}' {
				count--
				if count == 0 {
					break
				}
			}
		}
	}

	return sb.String()
}

// 从str中，key的位置，向前后查找，尝试找出包含他的一个json
func ExtractAllJsonContainsKey(str, key string) []string {
	results := []string{}
	for {
		index := strings.Index(str, key)
		if index < 0 {
			break
		}

		// 从index的位置向前查找大括号
		startIndex := -1
		for i := index; i >= 0; i-- {
			if str[i] == '}' {
				// 失败，关键字不是被大括号包裹的
				break
			}

			if str[i] == '{' {
				// 成功
				startIndex = i
				break
			}
		}

		if startIndex < 0 {
			str = str[index+len(key):]
		} else {
			jstr := ExtractJsonAfterIndex(str, startIndex)
			if len(jstr) > 0 {
				results = append(results, jstr)
			}
			str = str[startIndex+len(jstr):]
		}
	}

	return results
}

// 字符串压缩
func CompressString(str string) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(str)); err != nil {
		return []byte{}
	}
	w.Close()
	return buf.Bytes()
}

// 字符串压缩
func CompressString2String(str string) string {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(str)); err != nil {
		return ""
	}
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// 字符串解压
func DecompressString(b []byte) string {
	var bufIn bytes.Buffer
	bufIn.Write(b)
	return DecompressStringFromReader(&bufIn)
}

// 字符串解压
func DecompressStringFromReader(reader io.Reader) string {
	if r, err := zlib.NewReader(reader); err == nil {
		bytes := make([]byte, 1024*16)
		var bytesTotal []byte
		for {
			if n, err := r.Read(bytes); err == io.EOF {
				bytesTotal = append(bytesTotal, bytes[:n]...)
				return string(bytesTotal)
			} else if err != nil {
				fmt.Println(err.Error())
				return ""
			} else {
				bytesTotal = append(bytesTotal, bytes[:n]...)
			}
		}
	} else {
		logger.LogPanic("create zlib.Reader failed: %s", err.Error())
	}

	return ""
}

// 字符串解压
func DecompressStringFromString(s string) string {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return DecompressString(b)
	} else {
		logger.LogPanic("", "base64 decode failed")
	}

	return ""
}

func SerializeString(w io.Writer, str string) {
	n := uint8(len(str))
	binary.Write(w, binary.LittleEndian, n)
	binary.Write(w, binary.LittleEndian, []byte(str))
}

func DeserializeString(r io.Reader) (string, bool) {
	n := uint8(0)
	if binary.Read(r, binary.LittleEndian, &n) == nil {
		buf := make([]byte, n)
		if binary.Read(r, binary.LittleEndian, buf) == nil {
			return string(buf), true
		}
	}
	return "", false
}

// 是否为可见字符
func IsVisibleByte(b byte) bool {
	return b >= '!' && b <= '~'
}
