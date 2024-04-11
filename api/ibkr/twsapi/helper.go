/*
- @Author: aztec
- @Date: 2024-02-27 17:03:28
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

import (
	"bytes"
	"math"
	"strconv"

	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

func readInt(buf *bytes.Buffer) int {
	str := readString(buf)
	return util.String2IntPanic(str)
}

func readIntMax(buf *bytes.Buffer) int {
	str := readString(buf)
	if str == "" {
		return math.MaxInt32
	} else {
		return util.String2IntPanic(str)
	}
}

func readBool(buf *bytes.Buffer) bool {
	str := readString(buf)
	if n, ok := util.String2Int(str); ok {
		return n != 0
	} else {
		return false
	}
}

func readFloat64(buf *bytes.Buffer) float64 {
	str := readString(buf)
	return util.String2Float64Panic(str)
}

func readFloat64Max(buf *bytes.Buffer) float64 {
	str := readString(buf)
	if str == "" {
		return math.MaxFloat64
	} else if str == "Infinity" {
		return math.Inf(1)
	} else {
		return util.String2Float64Panic(str)
	}
}

func readDecimal(buf *bytes.Buffer) decimal.Decimal {
	str := readString(buf)
	return util.String2DecimalPanic(str)
}

func readDecimalMax(buf *bytes.Buffer) decimal.Decimal {
	return decimal.NewFromFloat(readFloat64Max(buf))
}

func readBitMask(buf *bytes.Buffer) twsmodel.BitMask {
	return twsmodel.NewBitMask(readInt(buf))
}

func readString(buf *bytes.Buffer) string {
	if str, err := buf.ReadString(0); err == nil {
		if len(str) > 0 {
			str = str[:len(str)-1]
		}

		return str
	}

	return ""
}

func readStringUnquote(buf *bytes.Buffer) string {
	str := readString(buf)
	if str, err := strconv.Unquote(`"` + str + `"`); err == nil {
		return str
	}

	return ""
}

func visualizeBuffer(buf *bytes.Buffer) string {
	temp := []byte{}
	for _, b := range buf.Bytes() {
		if util.IsVisibleByte(b) {
			temp = append(temp, b)
		} else {
			temp = append(temp, ',')
		}
	}
	return string(temp)
}

func deployParamList(src []interface{}, dst *[]interface{}) {
	for _, v := range src {
		if v != nil {
			if array, ok := v.([]interface{}); ok {
				deployParamList(array, dst)
			} else {
				(*dst) = append(*dst, v)
			}
		}
	}
}
