/*
- @Author: aztec
- @Date: 2023-11-22 18:51:06
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package network

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/russross/blackfriday/v2"
)

func JsonHeaders() map[string]string {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json; charset=utf-8"
	return headers
}

type PuppeteerCookie struct {
	Name       string
	Value      string
	Domain     string
	Path       string
	ExpiresRaw float64
	HttpOnly   bool
	Secure     bool
	Session    bool
}

func (p PuppeteerCookie) ToCookie() http.Cookie {
	c := http.Cookie{
		Name:     p.Name,
		Value:    url.QueryEscape(p.Value),
		Domain:   p.Domain,
		Path:     p.Path,
		Expires:  time.Unix(int64(p.ExpiresRaw), 0),
		HttpOnly: p.HttpOnly,
		Secure:   p.Secure,
	}
	return c
}

func LoadCookiesPuppeteerStyle(path string) []http.Cookie {
	pCookies := make([]PuppeteerCookie, 0)
	cookies := make([]http.Cookie, 0)
	if util.ObjectFromFile(path, &pCookies) {
		for _, pc := range pCookies {
			cookies = append(cookies, pc.ToCookie())
		}
	}
	return cookies
}

func LoadCookiesPuppeteerStyleStr(str string) []http.Cookie {
	pCookies := make([]PuppeteerCookie, 0)
	cookies := make([]http.Cookie, 0)
	if err := util.ObjectFromString(str, &pCookies); err == nil {
		for _, pc := range pCookies {
			cookies = append(cookies, pc.ToCookie())
		}
	}
	return cookies
}

// 将一个.md文件转换成网页发送给客户端
func SendMarkdownAsPage(mdPath string, w http.ResponseWriter) {
	if b, err := os.ReadFile(mdPath); err == nil {
		html := blackfriday.Run(b, blackfriday.WithExtensions(blackfriday.HardLineBreak))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
	} else {
		io.WriteString(w, "open readme.md failed")
	}
}
