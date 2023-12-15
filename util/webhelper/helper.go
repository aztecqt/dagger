/*
 * @Author: aztec
 * @Date: 2023-3-16
 * @LastEditors: aztec
 * @Description: web爬虫的帮助函数
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package webhelper

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

func ClickAll(sel interface{}, opts ...chromedp.QueryOption) chromedp.QueryAction {
	return chromedp.QueryAfter(sel, func(ctx context.Context, execCtx runtime.ExecutionContextID, nodes ...*cdp.Node) error {
		if len(nodes) < 1 {
			return fmt.Errorf("selector %q did not return any nodes", sel)
		}

		var err error
		for _, n := range nodes {
			err = chromedp.MouseClickNode(n).Do(ctx)
		}
		return err
	}, append(opts, chromedp.NodeVisible)...)
}

func GetTexts(n *html.Node, texts *[]string, url *[]string) {
	if n.Type == html.TextNode && len(n.Data) > 0 {
		*texts = append(*texts, n.Data)
	}

	for _, a := range n.Attr {
		if a.Key == "href" {
			*url = append(*url, a.Val)
		}
	}

	child := n.FirstChild
	for child != nil {
		GetTexts(child, texts, url)
		child = child.NextSibling
	}
}
func SearchAttr(n *html.Node, f func(k, v string)) {
	for _, a := range n.Attr {
		f(a.Key, a.Val)
	}

	child := n.FirstChild
	for child != nil {
		SearchAttr(child, f)
		child = child.NextSibling
	}
}
