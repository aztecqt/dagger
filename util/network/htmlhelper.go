/*
 * @Author: aztec
 * @Date: 2023-01-16 18:05:34
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package network

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/aztecqt/dagger/util"
	"golang.org/x/net/html"
)

func GetTextsFromHtmlNode(n *html.Node) []string {
	texts := make([]string, 0)
	recvGetTextsFromHtmlNode(n, &texts)
	return texts
}

func GetTextsFromHtmlNodeTrim(n *html.Node, trimstr string) []string {
	temp := make([]string, 0)
	recvGetTextsFromHtmlNode(n, &temp)

	texts := make([]string, 0)
	for _, v := range temp {
		str := strings.Trim(v, trimstr)
		if len(str) > 0 {
			texts = append(texts, str)
		}
	}
	return texts
}

func recvGetTextsFromHtmlNode(n *html.Node, texts *[]string) {

	if n.Type == html.TextNode && len(n.Data) > 0 {
		*texts = append(*texts, n.Data)
	}

	child := n.FirstChild
	for child != nil {
		recvGetTextsFromHtmlNode(child, texts)
		child = child.NextSibling
	}
}

func WalkThroughHtmlNode(n *html.Node, fn func(n *html.Node)) {
	fn(n)

	child := n.FirstChild
	for child != nil {
		WalkThroughHtmlNode(child, fn)
		child = child.NextSibling
	}
}

// 从模板生成一个html页面文件
func CreateHtmlFromTemplate(htmlTemlatePath string, title, body string) ([]byte, bool) {
	// 打开并读取模板文件
	if fsrc, err := os.OpenFile(htmlTemlatePath, os.O_RDONLY, os.ModePerm); err == nil {
		// 逐行读取目标文件，找到指定位置了，就写进去
		scanner := bufio.NewScanner(fsrc)
		buf := bytes.NewBuffer(nil)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "<title>") {
				if len(title) > 0 {
					buf.WriteString(fmt.Sprintf("<title>%s</title>", title))
				}
			} else if line == "</body>" {
				buf.WriteString(body)
				buf.WriteString("\n")
				buf.WriteString(line)
				buf.WriteString("\n")
			} else {
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}
		fsrc.Close()
		return buf.Bytes(), true
	} else {
		return nil, false
	}
}

// 从模板生成一个html页面文件
func CreateHtmlFileFromTemplate(htmlTemlatePath, htmlPath, title, body string) bool {
	if b, ok := CreateHtmlFromTemplate(htmlTemlatePath, title, body); ok {
		util.MakeSureDirForFile(htmlPath)
		if fdst, err := os.OpenFile(htmlPath, os.O_CREATE|os.O_WRONLY, os.ModePerm); err == nil {
			fdst.Write(b)
			fdst.Close()
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// 生成一个文本标签
func CreateHtmlText(content, class string) string {
	return fmt.Sprintf("<div class=\"%s\">%s</div>", class, content)
}

// 生成一个链接标签
func CreateHtmlHref(content, url, class string, newPage, newline bool) string {
	if newPage {
		return fmt.Sprintf(`<a class="%s" href="%s" target="_blank">%s</a>%s`, class, url, content, util.ValueIf(newline, "<br>", ""))
	} else {
		return fmt.Sprintf(`<a class="%s" href="%s">%s</a>%s`, class, url, content, util.ValueIf(newline, "<br>", ""))
	}
}
