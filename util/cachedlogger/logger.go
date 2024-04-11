/*
- @Author: aztec
- @Date: 2024-01-04 10:17:50
- @Description:
- @循环日志，用于调试某个特定bug，且日志数量特别多的情况
- @这种日志，平时不会进行磁盘IO，而是维持一个循环队列，最大保存n条日志
- @当外部条件触发时，才将缓存的日志写入文件并存档（可以多次存档，每个存档是一个独立的文件）
- @这样可以只用极少的磁盘空间，来调试原本需要大量日志文件才能定位的bug
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package cachedlogger

import (
	"fmt"
	"os"
	"time"

	"github.com/aztecqt/dagger/util"
)

type Logger struct {
	name     string
	cache    [][]string // 双缓存队列
	cacheLen int
	i0       int // 当前是哪个cache
	i1       int // cache内部index
}

func NewLogger(cacheLen int, name string) *Logger {
	l := &Logger{}
	l.name = name
	l.cache = make([][]string, 2)
	l.cache[0] = make([]string, cacheLen)
	l.cache[1] = make([]string, cacheLen)
	l.cacheLen = cacheLen
	return l
}

// 保存一条日志。日志内容被写入循环队列，但不写入磁盘
func (l *Logger) Log(format string, a ...interface{}) {
	msg := ""
	if len(a) > 0 {
		msg = fmt.Sprintf(format, a...)
	} else {
		msg = format
	}
	now := time.Now()
	msg = fmt.Sprintf("[%s.%03d] %s\n", now.Format("2006-01-02T15:04:05"), now.UnixMilli()%1000, msg)

	if l.i1 < l.cacheLen {
		l.cache[l.i0][l.i1] = msg
	}

	l.i1++
	if l.i1 >= l.cacheLen {
		l.i1 = 0
		l.i0 = (l.i0 + 1) % 2
	}
}

// 将循环队列保存到磁盘
func (l *Logger) Save() {
	path := fmt.Sprintf("./log/cached/%s/%s.log", l.name, time.Now().Format("2006-01-02-15-04-05"))
	util.MakeSureDirForFile(path)
	i0 := (l.i0 + 1) % 2
	i1 := l.i0
	if file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm); err == nil {
		// 上一段
		for i := 0; i < l.cacheLen; i++ {
			if len(l.cache[i0][i]) > 0 {
				file.WriteString(l.cache[i0][i]) // 自带换行
			}
		}

		// 这一段
		for i := 0; i < l.i1; i++ {
			if len(l.cache[i1][i]) > 0 {
				file.WriteString(l.cache[i1][i]) // 自带换行
			}
		}
		file.Close()
	}
}
