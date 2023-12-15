/*
 * @Author: aztec
 * @Date: 2023-08-20
 * @Description: 读取profile.txt里的名称，生成profile目录
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package util

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func GetProfileDir() (string, bool) {
	// 读取profile名
	bpf, err := os.ReadFile("profile.txt")
	if err != nil {
		fmt.Println("read profile.txt failed")
		time.Sleep(time.Second * 3)
		return "", false
	} else {
		text := string(bpf)
		splited := strings.Split(text, "\r\n")
		if len(splited) == 0 {
			splited = strings.Split(text, "\r")
		}
		if len(splited) == 0 {
			splited = strings.Split(text, "\n")
		}
		if len(splited) == 0 {
			return "", false
		}

		profileName := splited[0]
		profileDir := fmt.Sprintf("profiles/%s", profileName)
		return profileDir, true
	}
}
