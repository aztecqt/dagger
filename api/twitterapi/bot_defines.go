/*
 * @Author: aztec
 * @Date: 2022-11-21 17:01:08
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package twitterapi

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aztecqt/dagger/util"
	"golang.org/x/net/html"
)

const bearerToken = "AAAAAAAAAAAAAAAAAAAAAFrUjQEAAAAAXmQlN6L5iPnDcgPlGjyVv7tDfiA%3DgBHPXC4MnEsCdHoEWdw0dEsnn8Lr6x42NDGBTd5aEztczuZjkW"
const consumerKey = "byPK4GHoeCjmrlB9hoRr3HXQp"
const consumerSecret = "5nZ4ZvtFAZzGSvQSmOXm24U97YeCxJWmptyPBmLM9TRdgyqDgr"
const userAccessToken = "1421059738368892932-8JhqwCnIfrDqUP30LW6ZOMoxUoPLct"
const userAccessTokenSecret = "ZzIkDwBcVKPxMSUmILMO0rIoby2x0kvOyJW7lxVuMd56X"

func hmacshar1(base, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(base))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// =======================bot=======================
// 一篇推文
type Tweet struct {
	Text          string    `json:"text"`           // 正文
	Language      string    `json:"lang"`           // 语言
	Author        string    `json:"owner"`          // 发布者
	Time          time.Time `json:"time"`           // 发布时间（UTC时间）
	Url           string    `json:"url"`            // 推文链接
	RetweetBy     string    `json:"retweet_by"`     // 被何人转推
	TweetId       int64     `json:"tweet_id"`       // 推文数字id
	HasImage      bool      `json:"has_img"`        // 是否包含图片
	HasVideo      bool      `json:"has_video"`      // 是否包含视频
	HasOutterLink bool      `json:"has_outterlink"` // 是否包含外链
	HasQuoted     bool      `json:"has_quoted"`     // 是否引用了其他推文
	Quoted        struct {
		Text     string    `json:"text"`
		Language string    `json:"lang"`
		Author   string    `json:"owner"`
		Time     time.Time `json:"time"`
	} `json:"quoted"` // 被提及的推文
}

func (t *Tweet) String() string {
	str := fmt.Sprintf("id:%d\nowner:%s\nretweet-by:%s\ntime:%v\nurl:%s\nlang:%s\ntext:%s\n",
		t.TweetId,
		t.Author,
		t.RetweetBy,
		t.Time.Local(),
		t.Url,
		t.Language,
		t.Text)

	if len(t.Quoted.Text) > 0 && len(t.Quoted.Author) > 0 {
		str += fmt.Sprintf("quoted:%s\nquoted-time:%v\nquoted-text:%s\n",
			t.Quoted.Author,
			t.Quoted.Time.Local(),
			t.Quoted.Text)
	}

	return str
}

// 个人主页
type HomePage struct {
	Tweets []*Tweet
}

func (h *HomePage) String() string {
	sb := strings.Builder{}
	for _, t := range h.Tweets {
		sb.WriteString(t.String())
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func directText(s *goquery.Selection) string {
	var buf bytes.Buffer

	// Slightly optimized vs calling Each: no single selection object created
	var f func(*html.Node) bool
	f = func(n *html.Node) bool {
		if n.Type == html.TextNode {
			// Keep newlines and spaces, like jQuery
			buf.WriteString(n.Data)
			return true
		}
		if n.FirstChild != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if f(c) {
					break
				}
			}
		}
		return false
	}
	for _, n := range s.Nodes {
		if f(n) {
			break
		}
	}

	return buf.String()
}

type tweetTextWrap struct {
	author          string
	authorToConfirm string
	textbuffer      bytes.Buffer
	lang            string
	time            time.Time
}

func (t *tweetTextWrap) empty() bool {
	return len(t.author) == 0 && t.time.IsZero()
}

type tweetAnalysisContext struct {
	twt             *Tweet
	ttws            []tweetTextWrap
	inTweetTextNode bool
	ttwLatest       tweetTextWrap
	isAD            bool
}

func (t *tweetAnalysisContext) ttwNotEmpty() bool {
	if len(t.ttws) > 0 || !t.ttwLatest.empty() {
		return true
	} else {
		return false
	}
}

// 将最后一段推文合并进来
func (t *tweetAnalysisContext) mergeLatestTTW() {
	if !t.ttwLatest.empty() {
		t.ttws = append(t.ttws, t.ttwLatest)
		t.ttwLatest = tweetTextWrap{}
	}
}

func tweetFromSelection(s *goquery.Selection) *Tweet {
	t := new(tweetAnalysisContext)
	t.twt = new(Tweet)
	t.ttws = make([]tweetTextWrap, 0)
	for _, n := range s.Nodes {
		doAnalize(n, t)
	}

	// 最多支持两段推文，一个是正文，一个是提及的推文
	if len(t.ttws) > 0 {
		ttw := t.ttws[0]
		t.twt.Author = ttw.author
		t.twt.Language = ttw.lang
		t.twt.Text = ttw.textbuffer.String()
		t.twt.Time = ttw.time

		if len(t.ttws) > 1 {
			ttw := t.ttws[1]
			t.twt.Quoted.Author = ttw.author
			t.twt.Quoted.Language = ttw.lang
			t.twt.Quoted.Text = ttw.textbuffer.String()
			t.twt.Quoted.Time = ttw.time
			t.twt.HasQuoted = true
		}

		if t.isAD {
			return nil
		} else {
			return t.twt
		}
	} else {
		return nil
	}
}

func doAnalize(n *html.Node, t *tweetAnalysisContext) {
	isTweetTextNode := false
	isArticalNode := false
	if n.Type == html.ElementNode {
		if n.Data == "a" {
			if v, ok := getAttributeValue("href", n); ok {
				if strings.Index(v, "/") == 0 {
					if strings.Count(v, "/") == 1 {
						// 首个满足此条件的，是推文的发送者
						if len(t.twt.Author) == 0 {
							t.twt.Author = v[1:]
						}
					} else if strings.Count(v, "/") == 3 {
						// 这个是推文链接，其中包含真正的作者和推文id
						if t.twt.TweetId == 0 {
							ss := strings.Split(v, "/")
							if ss[2] == "status" {
								author := ss[1]
								tid, tidok := util.String2Int64(ss[3])
								if tidok {
									t.twt.TweetId = tid
									t.twt.Url = v
									if author != t.twt.Author {
										t.twt.RetweetBy = t.twt.Author
										t.twt.Author = author
									}
								}
							}
						}
					}
				}

				if strings.Contains(v, "https://") {
					// 正文之后出现外链标签，说明推文包含外链
					if t.ttwNotEmpty() {
						t.twt.HasOutterLink = true
					}
				}
			}
		} else if n.Data == "time" {
			if v, ok := getAttributeValue("datetime", n); ok {
				tm, err := time.Parse("2006-01-02T15:04:05.000Z", v)
				if err == nil {
					t.ttwLatest.time = tm
				}
			}
		} else if n.Data == "div" {
			datatestid, ok0 := getAttributeValue("data-testid", n)
			lang, ok1 := getAttributeValue("lang", n)
			if ok0 && ok1 && datatestid == "tweetText" {
				// 这个是推文头部
				t.ttwLatest.lang = lang
				t.inTweetTextNode = true
				isTweetTextNode = true
			}
		} else if n.Data == "img" {
			if getAttributeValueStr("draggable", n) == "true" {
				if t.ttwNotEmpty() {
					// 推文之后出现img标签说明有图片（非表情），说明有推文中包含图片
					t.twt.HasImage = true
				}
			}
		} else if n.Data == "video" {
			t.twt.HasVideo = true
		} else if n.Data == "article" {
			isArticalNode = true
		}
	} else if n.Type == html.TextNode {
		if t.inTweetTextNode {
			t.ttwLatest.textbuffer.WriteString(n.Data)
		} else if strings.Index(n.Data, "@") == 0 {
			t.ttwLatest.authorToConfirm = n.Data[1:]
		} else if n.Data == "·" {
			t.ttwLatest.author = t.ttwLatest.authorToConfirm
		} else if n.Data == "Promoted" {
			t.isAD = true
		}
	}

	if n.FirstChild != nil {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			doAnalize(c, t)
		}
	}

	if isTweetTextNode {
		t.mergeLatestTTW()
		t.inTweetTextNode = false
	} else if isArticalNode {
		t.mergeLatestTTW()
	}
}

func getAttributeValue(attrName string, n *html.Node) (string, bool) {
	if n == nil {
		return "", false
	}

	for i, a := range n.Attr {
		if a.Key == attrName {
			return n.Attr[i].Val, true
		}
	}
	return "", false
}

func getAttributeValueStr(attrName string, n *html.Node) string {
	if n == nil {
		return ""
	}

	for i, a := range n.Attr {
		if a.Key == attrName {
			return n.Attr[i].Val
		}
	}
	return ""
}
