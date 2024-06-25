/*
- @Author: aztec
- @Date: 2024-06-04 09:33:27
- @Description: apikey-secretkey管理器
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/terminal"
	"github.com/aztecqt/dagger/util/webservice/keymgr/keymgr"
	"github.com/jedib0t/go-pretty/v6/table"
)

type LaunchConfig struct {
	RedisCfg util.RedisConfig `json:"redis"`
}

func main() {
	fmt.Println("===webservice key management===")
	logger.Init(logger.SplitMode_ByDays, 7)
	input := bufio.NewScanner(os.Stdin)

	// 连接数据库
	lc := LaunchConfig{}
	rc := &util.RedisClient{}
	if util.ObjectFromFile("launch.json", &lc) {
		rc.InitFromConfig(lc.RedisCfg)
		fmt.Printf("redis %s@db%d connected\n", lc.RedisCfg.Addr, lc.RedisCfg.DB)
	} else {
		fmt.Printf("redis %s@db%d connect failed\n", lc.RedisCfg.Addr, lc.RedisCfg.DB)
		time.Sleep(time.Second)
		return
	}

	for {
		fmt.Print(">")
		input.Scan()
		line := input.Text()
		ss := strings.Split(line, " ")
		switch ss[0] {
		case "help":
			fmt.Println("ls")
			fmt.Println("gen tag")
			fmt.Println("del tag")
			fmt.Println("save tag")
		case "gen":
			if len(ss) < 2 {
				fmt.Println("missing tag")
				continue
			}
			tag := ss[1]
			// 检查tag是否存在
			if keymgr.Exist(rc, tag) {
				fmt.Printf("tag %s already exist\n", tag)
				continue
			}

			if apiKey, err := generateRandomBytes(32); err == nil {
				if secretKey, err := generateRandomBytes(32); err == nil {
					s := keymgr.SecretKey{
						ApiKey:    string(apiKey),
						SecretKey: string(secretKey),
					}
					if keymgr.Save(rc, tag, s) {
						fmt.Printf("apikey: %s\n", apiKey)
						fmt.Printf("secretKey: %s\n", secretKey)
					} else {
						fmt.Println("write to redis failed")
					}
				} else {
					fmt.Println("gen secretkey failed")
				}
			} else {
				fmt.Println("gen apikey failed")
			}
		case "ls":
			keys := keymgr.GetAll(rc)
			t := terminal.GenTableWriter(true)
			t.AppendHeader(table.Row{"tag", "api-key", "secret-key"})
			for tag, k := range keys {
				t.AppendRow(table.Row{tag, k.ApiKey, k.SecretKey})
			}
			fmt.Println(t.Render())
		case "save":
			if len(ss) < 2 {
				fmt.Println("missing tag")
				continue
			}

			tag := ss[1]
			if k, ok := keymgr.GetByTag(rc, tag); ok {
				os.WriteFile(fmt.Sprintf("api-key.%s.txt", tag), []byte(fmt.Sprintf("api-key: %s\nsecret-key: %s", k.ApiKey, k.SecretKey)), os.ModePerm)
				fmt.Println("saved")
				cmd := exec.Command("explorer.exe", ".\\")
				go cmd.Run()
			} else {
				fmt.Println("load secret-key failed")
			}
		case "del":
			if len(ss) < 2 {
				fmt.Println("missing tag")
				continue
			}

			tag := ss[1]
			if keymgr.DeleteByTag(rc, tag) {
				fmt.Printf("deleted %s\n", tag)
			} else {
				fmt.Printf("delete %s failed\n", tag)
			}
		}
	}
}

// 生成随机字节
func generateRandomBytes(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
