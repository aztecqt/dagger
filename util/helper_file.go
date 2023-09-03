/*
 * @Author: aztec
 * @Date: 2023-09-03 15:12:04
 * @Description: 文件相关的帮助函数
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/aztecqt/dagger/util/logger"
)

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
	if i < 0 {
		i = strings.LastIndex(filePath, "\\")
	}
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

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}

	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
