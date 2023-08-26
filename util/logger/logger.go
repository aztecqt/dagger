/*
 * @Author: aztec
 * @Date: 2022-03-24 15:16:59
 * @LastEditors: aztec
 * @LastEditTime: 2023-08-18 10:41:42
 * @FilePath: \stratergy_antc:\work\svn\quant\go\src\dagger\util\logger\logger.go
 * @Description:
 * 日志管理器。基于go自带的log包。以hour/day为文件夹存储个个日志文件
 * 可以设置需要保留的日志文件夹个数以防止占用过多磁盘空间
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package logger

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// 日志分组模式
const (
	SplitMode_ByDays = iota
	SplitMode_ByHours
	SplitMode_ByMinutes
)

var splitMode = SplitMode_ByDays

// 日志等级
const (
	LogLevel_Debug = iota
	LogLevel_Info
	LogLevel_Important
	LogLevel_None
)

var FileLogLevel = LogLevel_Debug
var ConsleLogLevel = LogLevel_Important

func MinLogLevel() int {
	if FileLogLevel < ConsleLogLevel {
		return FileLogLevel
	} else {
		return ConsleLogLevel
	}
}

// 最大日志文件数量
var maxFileCount = 0

// 当前日志文件名
var fileDir string = "log/"
var filePath string

// 上次输出日志的时间
var lastTime = time.Now()

// go日志对象
var fileLogger *log.Logger
var consoleLogger *log.Logger

// 初始化需要手动调用
var inited bool

func Init(sMode int, maxCount int) {
	splitMode = sMode
	maxFileCount = maxCount
	switch splitMode {
	case SplitMode_ByDays:
		fmt.Printf("log system initializing...set to keep %d days\n", maxCount)
	case SplitMode_ByHours:
		fmt.Printf("log system initializing...set to keep %d hours\n", maxCount)
	case SplitMode_ByMinutes:
		fmt.Printf("log system initializing...set to keep %d minutes\n", maxCount)
	}

	createLogDir()
	createInnerLogger()
	checkLogFileCount()
	inited = true
}

// 当前时间对应的日志文件路径
func nowFilePath() string {
	path := ""
	now := time.Now()
	switch splitMode {
	case SplitMode_ByDays:
		path = fileDir + now.Format("2006-01-02.log")
	case SplitMode_ByHours:
		path = fileDir + now.Format("2006-01-02_15.log")
	case SplitMode_ByMinutes:
		path = fileDir + now.Format("2006-01-02_15-04.log")
	}

	return path
}

// 创建日志根目录
func createLogDir() {
	_, err := os.Stat(fileDir)
	if os.IsNotExist(err) {
		os.Mkdir(fileDir, os.ModePerm)
	}
}

// 创建新的innerLogger
func createInnerLogger() {
	filePath = nowFilePath()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Panicln("failed to create log file, path:", filePath)
	} else {
		fileLogger = log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	consoleLogger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)
}

// 当日志目录下的日志文件过多时，删除一部分日志文件
func checkLogFileCount() {
	fileInfos, err := ioutil.ReadDir(fileDir)
	if err != nil {
		log.Panicln("failed to get log file list")
	} else {
		if len(fileInfos) > maxFileCount && maxFileCount > 0 {
			for i := 0; i < len(fileInfos)-maxFileCount; i++ {
				fullPath := fileDir + fileInfos[i].Name()
				os.Remove(fullPath)
			}
		}
	}
}

// 判断是否跨越到下一个时间片
func needNewInnerLogger() bool {
	now := time.Now()
	switch splitMode {
	case SplitMode_ByDays:
		return lastTime.YearDay() != now.YearDay()
	case SplitMode_ByHours:
		return lastTime.Hour() != now.Hour() || (now.Unix()-lastTime.Unix()) > 3600
	case SplitMode_ByMinutes:
		return lastTime.Minute() != time.Now().Minute() || (now.Unix()-lastTime.Unix()) > 60
	}

	return false
}

func doFileLog(msg *string) {
	if !inited {
		panic("call logger.Init first!")
	}

	if needNewInnerLogger() {
		createInnerLogger()
		checkLogFileCount()
	}

	fileLogger.Println(*msg)
}

func LogDebug(prefix string, format string, a ...interface{}) {
	if FileLogLevel <= LogLevel_Debug || ConsleLogLevel <= LogLevel_Debug {
		msg := ""
		if len(a) > 0 {
			msg = fmt.Sprintf(format, a...)
			msg = fmt.Sprintf("[%s] %s", prefix, msg)
		} else {
			msg = fmt.Sprintf("[%s] %s", prefix, format)
		}

		if FileLogLevel <= LogLevel_Debug {
			doFileLog(&msg)
		}

		if ConsleLogLevel <= LogLevel_Debug {
			consoleLogger.Println(msg)
		}
	}
}

func LogInfo(prefix string, format string, a ...interface{}) {
	if FileLogLevel <= LogLevel_Info || ConsleLogLevel <= LogLevel_Info {
		msg := ""
		if len(a) > 0 {
			msg = fmt.Sprintf(format, a...)
			msg = fmt.Sprintf("[%s] %s", prefix, msg)
		} else {
			msg = fmt.Sprintf("[%s] %s", prefix, format)
		}

		if FileLogLevel <= LogLevel_Info {
			doFileLog(&msg)
		}

		if ConsleLogLevel <= LogLevel_Info {
			consoleLogger.Println(msg)
		}
	}
}

func LogImportant(prefix string, format string, a ...interface{}) {
	if FileLogLevel <= LogLevel_Important || ConsleLogLevel <= LogLevel_Important {
		msg := ""
		if len(a) > 0 {
			msg = fmt.Sprintf(format, a...)
			msg = fmt.Sprintf("[%s] %s", prefix, msg)
		} else {
			msg = fmt.Sprintf("[%s] %s", prefix, format)
		}

		if FileLogLevel <= LogLevel_Important {
			doFileLog(&msg)
		}

		if ConsleLogLevel <= LogLevel_Important {
			consoleLogger.Println(msg)
		}
	}
}

func LogPanic(prefix string, format string, a ...interface{}) {
	if FileLogLevel <= LogLevel_Important || ConsleLogLevel <= LogLevel_Important {
		msg := ""
		if len(a) > 0 {
			msg = fmt.Sprintf(format, a...)
			msg = fmt.Sprintf("[%s] %s", prefix, msg)
		} else {
			msg = fmt.Sprintf("[%s] %s", prefix, format)
		}

		if FileLogLevel <= LogLevel_Important {
			doFileLog(&msg)
		}

		if ConsleLogLevel <= LogLevel_Important {
			consoleLogger.Println(msg)
		}

		panic(msg)
	}
}
