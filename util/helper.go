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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"

	"github.com/aztecqt/dagger/util/logger"
)

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
