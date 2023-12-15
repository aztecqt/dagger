/*
 * @Author: aztec
 * @Date: 2023-01-16 18:05:34
 * @Description:
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package network

import (
	"strings"

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
