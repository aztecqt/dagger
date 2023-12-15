/*
 * @Author: aztec
 * @Date: 2023-09-03 15:12:04
 * @Description: 文件相关的帮助函数
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"bytes"
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
			os.Chmod(filePath, 0777)
			if err := os.WriteFile(filePath, data, 0777); err == nil {
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

func StringToFile(filePath string, content string) {
	MakeSureDirForFile(filePath)
	if file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666); err == nil {
		file.WriteString(content)
		file.Close()
	}
}

type Deserializable interface {
	Deserialize(r io.Reader) bool
}

// 从一个二进制文件中反序列化一组对象
func FileDeserializeToObjectList[T Deserializable](
	filePath string,
	fnNewObj func() T,
	fnOnNewObj func(o T) bool) {

	if file, err := os.OpenFile(filePath, os.O_RDONLY, 0666); err == nil {
		if st, err := file.Stat(); err == nil {
			b := make([]byte, st.Size())
			file.Read(b)
			buf := bytes.NewBuffer(b)

			// 把数据都读出来
			for {
				t := fnNewObj()
				if t.Deserialize(buf) {
					goon := fnOnNewObj(t)
					if !goon {
						break
					}
				} else {
					break
				}
			}
		}
	} else {
		fmt.Println(err.Error())
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

// 把一个字符串转换成文件名
func ConverToFileName(key, ext string) string {
	sb := strings.Builder{}
	b := []byte(key)

	isDot := false
	for i, v := range b {
		if v >= 'a' && v <= 'z' || v >= 'A' && v <= 'Z' || v >= '0' && v <= '9' {
			sb.WriteByte(v)
			isDot = false
		} else if (v == '/' || v == '\\') && i > 0 {
			sb.WriteByte('.')
			isDot = true
		}
	}

	if !isDot {
		sb.WriteByte('.')
	}

	sb.WriteString(ext)
	return sb.String()
}

// 去掉不符合文件名规范的字符
func ConverToDirName(str string) string {
	sb := strings.Builder{}
	b := []byte(str)

	for _, v := range b {
		if v >= 'a' && v <= 'z' || v >= 'A' && v <= 'Z' || v >= '0' && v <= '9' || v == '/' || v == '\\' || v == '.' {
			sb.WriteByte(v)
		}
	}

	return sb.String()
}

// 保存缓存文件
func SaveTempBuffer(key string, value string) {
	path := fmt.Sprintf("./temp/%s", ConverToFileName(key, "txt"))
	MakeSureDirForFile(path)
	os.WriteFile(path, []byte(value), 0666)
}

// 加载缓存文件
func LoadTempBuffer(key string) string {
	path := fmt.Sprintf("./temp/%s", ConverToFileName(key, "txt"))
	MakeSureDirForFile(path)
	if b, err := os.ReadFile(path); err == nil {
		return string(b)
	} else {
		return ""
	}
}
