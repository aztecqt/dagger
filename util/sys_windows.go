/*
 * @Author: aztec
 * @Date: 2022-12-20 09:09:03
 * @Description: 重定向系统错误输出，以便捕获fatal error
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package util

import (
	"os"
	"syscall"
)

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	var kernel32 = syscall.MustLoadDLL("kernel32.dll")
	var procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

// RedirectStderr to the file passed in
func RedirectStderr() (err error) {
	logFile, err := os.OpenFile("./std-error.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	err = setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(logFile.Fd()))
	if err != nil {
		return
	}
	// SetStdHandle does not affect prior references to stderr
	os.Stderr = logFile
	return
}
