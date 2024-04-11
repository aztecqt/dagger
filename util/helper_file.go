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
	"compress/flate"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

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

func StringToFile(filePath string, content string) bool {
	MakeSureDirForFile(filePath)
	if file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666); err == nil {
		file.WriteString(content)
		file.Close()
		return true
	} else {
		return false
	}
}

func BytesToFile(filePath string, b []byte) bool {
	MakeSureDirForFile(filePath)
	if file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666); err == nil {
		file.Write(b)
		file.Close()
		return true
	} else {
		return false
	}
}

type Deserializable interface {
	Deserialize(r io.Reader) bool
}

// 从一个二进制文件中反序列化一组对象
func FileDeserializeToObjects[T Deserializable](
	filePath string,
	fnNewObj func() T,
	fnOnNewObj func(o T) bool) bool {

	if file, err := os.OpenFile(filePath, os.O_RDONLY, 0666); err == nil {
		// 把数据都读出来
		for {
			t := fnNewObj()
			if t.Deserialize(file) {
				goon := fnOnNewObj(t)
				if !goon {
					break
				}
			} else {
				break
			}
		}
		return true
	} else {
		return false
	}
}

// 从一个reader反序列化一组对象
func DeserializeToObjects[T Deserializable](
	reader io.Reader,
	fnNewObj func() T,
	fnOnNewObj func(o T) bool) bool {

	// 把数据都读出来
	for {
		t := fnNewObj()
		if t.Deserialize(reader) {
			goon := fnOnNewObj(t)
			if !goon {
				break
			}
		} else {
			break
		}
	}
	return true
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

// #region compress
const CompressBufferSize = 1024 * 1024 * 16

type CompressedFile struct {
	inputFile        *os.File
	decompressReader io.ReadCloser
}

func (c CompressedFile) Read(p []byte) (n int, err error) {
	return c.decompressReader.Read(p)
}

func (c CompressedFile) Close() error {
	c.inputFile.Close()
	return c.decompressReader.Close()
}

// 压缩文件(flate方式)
// level=1~9
func CompressFile_Flate(srcPath, dstPath string, level int) (bool, int64) {
	t0 := time.Now()
	if f0, err := os.OpenFile(srcPath, os.O_RDONLY, os.ModePerm); err == nil {
		defer f0.Close()
		os.Remove(dstPath)
		if f1, err := os.OpenFile(dstPath, os.O_CREATE|os.O_RDWR, os.ModePerm); err == nil {
			defer f1.Close()
			if wr, err := flate.NewWriter(f1, level); err == nil {
				defer wr.Close()
				buf := make([]byte, CompressBufferSize)
				len := 0
				for {
					if n, _ := f0.Read(buf); n > 0 {
						len += n
						if _, err := wr.Write(buf[:n]); err != nil {
							return false, 0
						}
					} else {
						break
					}
				}
				t1 := time.Now()
				return true, t1.UnixMilli() - t0.UnixMilli()
			} else {
				return false, 0
			}
		} else {
			return false, 0
		}
	} else {
		return false, 0
	}
}

// 解压缩文件（flate方式）
func DecompressFile_Flate(srcPath, dstPath string) bool {
	if b, _ := LoadCompressedFile_Flate(srcPath); b != nil {
		if os.WriteFile(dstPath, b.Bytes(), os.ModePerm) == nil {
			return true
		}
	}

	return false
}

// 打开压缩文件（flate方式）
func OpenCompressedFile_Flate(path string) (*CompressedFile, error) {
	if f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm); err == nil {
		return &CompressedFile{inputFile: f, decompressReader: flate.NewReader(f)}, nil
	} else {
		return nil, err
	}
}

// 加载压缩文件（flate方式）
func LoadCompressedFile_Flate(path string) (*bytes.Buffer, int64) {
	t0 := time.Now()
	if f, err := OpenCompressedFile_Flate(path); err == nil {
		defer f.Close()
		buf := make([]byte, CompressBufferSize)
		b := bytes.NewBuffer(nil)
		for {
			if n, _ := f.Read(buf); n > 0 {
				if _, err := b.Write(buf[:n]); err != nil {
					return nil, 0
				}
			} else {
				break
			}
		}

		t1 := time.Now()
		return b, t1.UnixMilli() - t0.UnixMilli()
	} else {
		return nil, 0
	}
}

// 压缩文件(zlib方式)
func CompressFile_Zlib(srcPath, dstPath string) (bool, int64) {
	t0 := time.Now()
	if f0, err := os.OpenFile(srcPath, os.O_RDONLY, os.ModePerm); err == nil {
		defer f0.Close()
		os.Remove(dstPath)
		if f1, err := os.OpenFile(dstPath, os.O_CREATE|os.O_RDWR, os.ModePerm); err == nil {
			defer f1.Close()
			if wr := zlib.NewWriter(f1); wr != nil {
				defer wr.Close()
				buf := make([]byte, CompressBufferSize)
				len := 0
				for {
					if n, _ := f0.Read(buf); n > 0 {
						len += n
						if _, err := wr.Write(buf[:n]); err != nil {
							return false, 0
						}
					} else {
						break
					}
				}
				t1 := time.Now()
				return true, t1.UnixMilli() - t0.UnixMilli()
			} else {
				return false, 0
			}
		} else {
			return false, 0
		}
	} else {
		return false, 0
	}
}

// 解压缩文件（zlib方式）
func DecompressFile_Zlib(srcPath, dstPath string) bool {
	if b, _ := LoadCompressedFile_Zlib(srcPath); b != nil {
		if os.WriteFile(dstPath, b.Bytes(), os.ModePerm) == nil {
			return true
		}
	}

	return false
}

// 打开压缩文件（zlib方式）
func OpenCompressedFile_Zlib(path string) (*CompressedFile, error) {
	if f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm); err == nil {
		if r, err := zlib.NewReader(f); err == nil {
			return &CompressedFile{inputFile: f, decompressReader: r}, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// 加载压缩文件（zlib方式）
func LoadCompressedFile_Zlib(path string) (*bytes.Buffer, int64) {
	t0 := time.Now()
	if f, err := OpenCompressedFile_Zlib(path); err == nil {
		defer f.Close()
		buf := make([]byte, CompressBufferSize)
		b := bytes.NewBuffer(nil)
		for {
			if n, _ := f.Read(buf); n > 0 {
				if _, err := b.Write(buf[:n]); err != nil {
					return nil, 0
				}
			} else {
				break
			}
		}

		t1 := time.Now()
		return b, t1.UnixMilli() - t0.UnixMilli()
	} else {
		return nil, 0
	}
}

// #endregion
