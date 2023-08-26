/*
 * @Author: aztec
 * @Date: 2022-03-26 10:08:56
 * @Description: 通用帮助函数
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package util

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"aztecqt/dagger/util/logger"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/shopspring/decimal"
)

// #region time相关
// 东八区
var East8 = time.FixedZone("CST", 8*3600)

// 美国东部时间
func AmericaNYZone() *time.Location {
	z, err := time.LoadLocation("America/New_York")
	if err == nil {
		return z
	} else {
		logger.LogPanic("", err.Error())
		return nil
	}
}

func DateOfTime(t time.Time) time.Time {
	y, m, d := t.Date()
	name, _ := t.Zone()
	str := fmt.Sprintf("%04d-%02d-%02d 00:00:00 %s", y, m, d, name)
	date, e := time.Parse("2006-01-02 15:04:05 MST", str)
	if e != nil {
		fmt.Println(e.Error())
	}
	return date
}

func TimeNowUnix13() int64 {
	return time.Now().UnixNano() / 1e6
}

func ConvetUnix13ToTime(u13 int64) time.Time {
	return time.Unix(0, u13*1000000)
}

func ConvetUnix13StrToTime(u13str string) (time.Time, bool) {
	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000), true
	} else {
		return time.Unix(0, 0), false
	}
}

func ConvetUnix13StrToTimePanic(u13str string) time.Time {
	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000)
	} else {
		logger.LogPanic("", "can't parse [%s] to Unix13", u13str)
		return time.Unix(0, 0)
	}
}

func ConvetUnix13StrToTimePanicUnless(u13str string, unless string) time.Time {
	if u13str == unless {
		return time.Unix(0, 0)
	}

	u13, ok := String2Int64(u13str)
	if ok {
		return time.Unix(0, u13*1000000)
	} else {
		logger.LogPanic("", "can't parse [%s] to Unix13", u13str)
		return time.Unix(0, 0)
	}
}

func ConvetUnix13ToIsoTime(u13 int64) string {
	utcTime := ConvetUnix13ToTime(u13).UTC()
	iso := utcTime.String()
	isoBytes := []byte(iso)
	iso = string(isoBytes[:10]) + "T" + string(isoBytes[11:19]) + "Z"
	return iso
}

func Duration2Str(dur time.Duration) string {
	if dur.Hours() > 24 {
		d := int(dur.Hours() / 24)
		h := int(dur.Hours()) - d*24
		return fmt.Sprintf("%dday %dhour", d, h)
	} else {
		h := int(dur.Hours())
		m := int(dur.Minutes()) - h*60
		s := int(dur.Seconds()) - h*3600 - m*60
		return fmt.Sprintf("%d:%d:%d", h, m, s)
	}
}

func Duration2StrCn(dur time.Duration) string {
	if dur.Hours() > 24 {
		d := int(dur.Hours() / 24)
		h := int(dur.Hours()) - d*24
		return fmt.Sprintf("%d天 %d小时", d, h)
	} else {
		h := int(dur.Hours())
		m := int(dur.Minutes()) - h*60
		s := int(dur.Seconds()) - h*3600 - m*60
		return fmt.Sprintf("%d:%d:%d", h, m, s)
	}
}

// 1s=1
// 2m=120
// 3h=3600x3
// 4d....
func DurationStr2Seconds(str string) (sec int64) {
	sec = 0
	defer DefaultRecover()
	if len(str) > 1 {
		unit := str[len(str)-1]
		if num, numok := String2Int64(str[:len(str)-1]); numok {
			switch unit {
			case 's':
				sec = num
			case 'm':
				sec = num * 60
			case 'h':
				sec = num * 3600
			case 'd':
				sec = num * 3600 * 24
			default:
				sec = 0
			}
		}
	}
	return
}

// 1s
// 1m
// 1h
// 1d
// 1w
func String2Duration(s string) time.Duration {
	if len(s) > 0 {
		c := s[len(s)-1]
		n := String2IntPanic(s[0 : len(s)-1])
		switch c {
		case 's':
			return time.Second * time.Duration(n)
		case 'm':
			return time.Minute * time.Duration(n)
		case 'h':
			return time.Hour * time.Duration(n)
		case 'd':
			return time.Hour * time.Duration(n*24)
		case 'w':
			return time.Hour * time.Duration(n*24*7)
		default:
			panic("invalid duration unit: " + s)
		}
	} else {
		return time.Duration(0)
	}
}

// #endregion

// #region math and numbers
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

func LerpDecimal(a, b, r decimal.Decimal) decimal.Decimal {
	delta := b.Sub(a)
	return a.Add(delta.Mul(r))
}

func LerpFloat(a, b, r float64) float64 {
	delta := b - a
	return a + delta*r
}

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

// #endregion

// #region 其他
func GzipDecode(in []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(in))
	defer reader.Close()
	return ioutil.ReadAll(reader)
}

func DefaultRecover() {
	err := recover()

	if err != nil {
		logger.LogImportant("default recover", "panic caught by default recover")
		logger.LogImportant("default recover", "error: %s", reflect.ValueOf(err).String())
		logger.LogImportant("default recover", "call stack: %s", string(debug.Stack()))
	}
}

func DefaultRecoverWithCallback(cb func(err string)) {
	err := recover()

	if err != nil {
		logger.LogImportant("default recover", "panic caught by default recover")
		logger.LogImportant("default recover", "error: %s", reflect.ValueOf(err).String())
		logger.LogImportant("default recover", "call stack: %s", string(debug.Stack()))

		if cb != nil {
			cb(reflect.ValueOf(err).String())
		}
	}
}

func SliceRemove[T comparable](slice []T, item T) []T {
	if len(slice) == 0 {
		return slice
	}

	if slice[len(slice)-1] == item {
		return slice[0 : len(slice)-1]
	}

	length := len(slice) - 1
	for i := 0; i < length; i++ {
		if item == slice[i] {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func SliceRemoveAt[T any](slice []T, index int) []T {
	if index >= 0 && index < len(slice) {
		return append(slice[:index], slice[index+1:]...)
	} else {
		return slice
	}
}

// a = cond ? a0 : a1
func ValueIf[T any](cond bool, vTrue, vFalse T) T {
	if cond {
		return vTrue
	} else {
		return vFalse
	}
}

// #endregion

// #region 文件操作
func ObjectFromFile(filePath string, obj interface{}) bool {
	b, err := os.ReadFile(filePath)
	if err == nil {
		err := json.Unmarshal(b, obj)
		if err != nil {
			fmt.Println(err.Error())
			return false
		} else {
			return true
		}
	} else {
		return false
	}
}

func ObjectToFile(filePath string, obj interface{}) bool {
	filePath = strings.Replace(filePath, "\\", "/", -1)
	if MakeSureDirForFile(filePath) {
		if data, err := json.MarshalIndent(obj, "", "	"); err == nil {
			os.Chmod(filePath, 0222)
			if err := os.WriteFile(filePath, data, fs.FileMode(os.O_CREATE|os.O_RDWR)); err == nil {
				return true
			} else {
				logger.LogImportant("", err.Error())
				return false
			}
		} else {
			return false
		}
	} else {
		return false
	}
}

func MakeSureDirForFile(filePath string) bool {
	i := strings.LastIndex(filePath, "/")
	if i >= 0 {
		fileDir := filePath[:i]
		_, err := os.ReadDir(fileDir)
		if err != nil {
			// 不存在就创建
			err = os.MkdirAll(fileDir, fs.ModePerm)
			if err != nil {
				logger.LogImportant("", err.Error())
				return false
			} else {
				return true
			}
		} else {
			return true
		}
	}

	return true
}

func MakeSureDir(dir string) bool {
	_, err := os.ReadDir(dir)
	if err != nil {
		// 不存在就创建
		err = os.MkdirAll(dir, fs.ModePerm)
		if err != nil {
			logger.LogImportant("", err.Error())
			return false
		} else {
			return true
		}
	} else {
		return true
	}
}

// 判断所给路径文件/文件夹是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	//isnotexist来判断，是不是不存在的错误
	if os.IsNotExist(err) { //如果返回的错误类型使用os.isNotExist()判断为true，说明文件或者文件夹不存在
		return false, nil
	}
	return false, err //如果有错误了，但是不是不存在的错误，所以把这个错误原封不动的返回
}

// 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {

		return false
	}
	return s.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}

// 获取扩展名
func FileExtName(name string) string {
	ss := strings.Split(name, ".")
	if len(ss) > 1 {
		return ss[len(ss)-1]
	} else {
		return ""
	}
}

// #endregion

// #region 操作系统相关
// 杀死进程。目前仅支持windows，且ProcessKiller.exe需要跟本程序放在同一目录下
func KillProcessWithName(pname string) {
	cmd := exec.Command("ProcessKiller.exe", "chrome")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 调用windows默认程序打开一个文件
func OpenFileWithDefaultProgramOnWindows(path string) {
	exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path).Start()
}

// #endregion
