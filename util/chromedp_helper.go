/*
 * @Author: aztec
 * @Date: 2022-9-22
 * @Description: 封装了使用chromedp抓取网页实际内容的操作
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package util

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

var lastChromedpCasheClearTime = time.Time{}

// waitingWhat: 等待的html元素，如`div[class="stats"]`
// sleep：元素出现后再额外等待的时间
func GetHtmlByChrome(url string, waitingWhat string, sleepSec int64, headless bool) string {
	// 每小时清空一次chrome缓存
	if time.Since(lastChromedpCasheClearTime).Hours() > 1 {
		ClearChomeDpTempFiles()
		lastChromedpCasheClearTime = time.Now()
	}

	logPrefix := "chromedp_helper"
	options := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", headless),
		chromedp.Flag("blink-settings", "imageEnable=false"),
		chromedp.Flag("no-default-browser-check", true),
		//chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disk-cache-size", "67108864"), // 最大64M缓存
		chromedp.Flag("lang", "en-US"),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36`),
	}

	chromeCtx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	chromeCtx, cancel := chromedp.NewContext(chromeCtx, chromedp.WithLogf(log.Printf))
	chromeCtx, cancel = context.WithTimeout(chromeCtx, 60*time.Second)
	defer cancel()

	logger.LogInfo(logPrefix, fmt.Sprintf("visiting url: %s", url))
	var htmlContent string
	err := chromedp.Run(chromeCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady(waitingWhat),
		chromedp.Sleep(time.Second*time.Duration(sleepSec)),
		chromedp.OuterHTML(`document.querySelector("body")`, &htmlContent, chromedp.ByJSPath),
	)

	if err != nil {
		logger.LogImportant(logPrefix, "getHtml err: %s", err.Error())
		return ""
	}

	return htmlContent
}

func SaveCookies() chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		// cookies的获取对应是在devTools的network面板中
		// 1. 获取cookies
		cookies, err := network.GetCookies().Do(ctx)
		if err != nil {
			return
		}

		// 2. 序列化
		cookiesData, err := network.GetCookiesReturns{Cookies: cookies}.MarshalJSON()
		if err != nil {
			return
		}

		// 3. 存储到临时文件
		if err = os.WriteFile("cookies.tmp", cookiesData, 0755); err != nil {
			return
		}
		return
	}
}

func LoadCookies() chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		// 如果cookies临时文件不存在则直接跳过
		if _, _err := os.Stat("cookies.tmp"); os.IsNotExist(_err) {
			return
		}

		// 如果存在则读取cookies的数据
		cookiesData, err := os.ReadFile("cookies.tmp")
		if err != nil {
			return
		}

		// 反序列化
		cookiesParams := network.SetCookiesParams{}
		if err = cookiesParams.UnmarshalJSON(cookiesData); err != nil {
			return
		}

		// 设置cookies
		return network.SetCookies(cookiesParams.Cookies).Do(ctx)
	}
}

func FindButtonWithText(ctx context.Context, reg string) cdp.NodeID {
	ctx0, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nodes := make([]*cdp.Node, 0)
	chromedp.Run(ctx0, chromedp.Nodes("span", &nodes))

	exp := regexp.MustCompile(reg)
	for _, n := range nodes {
		if len(n.Children) == 1 && n.Children[0].NodeType == cdp.NodeTypeText {
			// 文本匹配
			if exp.MatchString(n.Children[0].NodeValue) {
				for {
					if n == nil {
						break
					}

					// 如果是button那就是你没错了
					if v, ok := n.Attribute("role"); ok {
						if v == "button" {
							return n.NodeID
						}
					}

					n = n.Parent
				}
			}
		}
	}

	return 0
}

func FindSpanWithText(ctx context.Context, reg string) cdp.NodeID {
	ctx0, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	nodes := make([]*cdp.Node, 0)
	chromedp.Run(ctx0, chromedp.Nodes("span", &nodes))

	exp := regexp.MustCompile(reg)
	for _, n := range nodes {
		if len(n.Children) == 1 && n.Children[0].NodeType == cdp.NodeTypeText {
			// 文本匹配
			text := n.Children[0].NodeValue
			if exp.MatchString(text) {
				return n.NodeID
			}
		}
	}

	return 0
}

func ClearChomeDpTempFiles() {
	logPrefix := "chromedp_helper"
	home := ""
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	} else {
		logger.LogPanic(logPrefix, "not implemented in non-windows os")
	}

	home = fmt.Sprintf("%s\\AppData\\Local", home)
	err := filepath.WalkDir(home, func(path string, d fs.DirEntry, err error) error {
		// 找出所有chromedp开头的文件夹
		if d.IsDir() && strings.Contains(d.Name(), "chromedp") {
			if fi, err := d.Info(); err == nil {
				hours := time.Since(fi.ModTime()).Hours()
				if hours > 1 {
					// 如果修改时间早于1小时则删除它
					if err := os.RemoveAll(path); err != nil {
						logger.LogImportant(logPrefix, "delete temp dir failed: %s", err.Error())
					} else {
						logger.LogInfo(logPrefix, "deleted temp dir: %s", path)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		logger.LogImportant(logPrefix, "ClearChomeDpTempFiles walk failed: %s", err)
	}
}
