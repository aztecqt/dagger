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
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/aztecqt/dagger/util/logger"
)

// #region 其他
func GzipDecode(in []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(in))
	defer reader.Close()
	return io.ReadAll(reader)
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

// a = cond0 ? vTrue0 : (cond1 ? vTrue1 : false)
func ValueIf2[T any](cond0, cond1 bool, vTrue0, vTrue1, vFalse T) T {
	if cond0 {
		return vTrue0
	} else if cond1 {
		return vTrue1
	} else {
		return vFalse
	}
}

// 把url中的host替换成p
// 支持如下格式
// www.baidu.com
// https://www.baidu.com/xxx
// http://p1.aztecqt.cn:8888
func ReplaceHost2Ip(url string) string {
	urlReal := url

	if strings.Index(urlReal, "://") > 0 {
		urlReal = strings.Split(urlReal, "://")[1]
	}

	if strings.Index(urlReal, ":") > 0 {
		urlReal = strings.Split(urlReal, ":")[0]
	}

	if strings.Index(urlReal, "/") > 0 {
		urlReal = strings.Split(urlReal, "/")[0]
	}

	addr, err := net.LookupHost(urlReal)
	if err != nil {
		return url
	}

	ip := addr[0]

	url = strings.Replace(url, urlReal, ip, 1)
	return url
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
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
	path, _ = filepath.Abs(path)
	exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path).Start()
}

// 当程序退出时
func OnProgramQuit(fn func()) {
	osc := make(chan os.Signal, 1)
	signal.Notify(osc, syscall.SIGTERM, syscall.SIGINT)
	<-osc
	fn()
}

// #endregion
