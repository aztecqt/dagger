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

// RedirectStderr to the file passed in
func RedirectStderr() (err error) {
	logFile, err := os.OpenFile("./std-error.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	err = syscall.Dup3(int(logFile.Fd()), int(os.Stderr.Fd()), 0)
	if err != nil {
		return
	}
	return
}

// 系统缓存目录
func SystemCachePath() string {
	if b, err := os.ReadFile("/var/cache/dagger/linux_cache_path.txt"); err == nil {
		return string(b)
	}

	return "/var/cache"
}
