/*
 * @Author: aztec
 * @Date: 2022-11-21 16:59:24
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package twitterapi

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/network"
)

const logPrefix = "twitter_api"

// OAuth 1.0
// headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})
// OAuth 2.0 app only
// headers["Authorization"] = fmt.Sprintf("Bearer %s", bearerToken)

func Init() {
	id2userInfo = make(map[string]UserInfoCore)
	un2userInfo = make(map[string]UserInfoCore)
}

// 查询tweet
func TweetLookup(id string) {
	ep := fmt.Sprintf("https://api.twitter.com/2/tweets/%s", id)
	method := "GET"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	network.HttpCall(ep, method, "", headers, func(r *http.Response, err error) {
		if err != nil {
			logger.LogImportant(logPrefix, err.Error())
		}
	})
}

// 返回自己的推特时间线（自己的推+关注者的推）
func TimeLines_MyReverseChronoLogical(limit int, sinceId int64) []RefinedTweet {
	tweets := make([]RefinedTweet, 0)

	selfUI, ok := MyUserInfo()
	if !ok {
		logger.LogImportant(logPrefix, "invalid user id")
		return tweets
	}

	ep := fmt.Sprintf("https://api.twitter.com/2/users/%s/timelines/reverse_chronological", selfUI.ID)
	qparam := url.Values{}
	qparam.Add("max_results", fmt.Sprintf("%d", limit))
	qparam.Add("tweet.fields", "author_id,conversation_id,created_at,referenced_tweets,lang")
	if sinceId > 0 {
		qparam.Add("since_id", fmt.Sprintf("%d", sinceId))
	}
	epWithQuery := ep + "?" + qparam.Encode()

	method := "GET"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, qparam)

	// 获取原始推文
	resp, err := network.ParseHttpResult[RawTweetList](logPrefix, "TimeLines_ReverseChronoLogical", epWithQuery, method, "", headers, nil, nil)

	// 提炼推文
	if err == nil {
		for _, rt := range resp.Data {
			if t := refineTweet(rt); t != nil {
				tweets = append(tweets, *t)
			}
		}
	}
	return tweets
}

func refineTweet(rt RawTweet) *RefinedTweet {
	t := RefinedTweet{}
	logPrefix := logPrefix + "_refineTweet"

	ui, uiok := GetUserInfoById(rt.AuthorId)
	if !uiok {
		logger.LogImportant(logPrefix, "get user_info of %s failed", rt.AuthorId)
		return nil
	} else {
		t.AuthorUserName = ui.UserName
		t.Author = ui.Name
	}

	tid, tidok := util.String2Int64(rt.TweetId)
	if !tidok {
		logger.LogImportant(logPrefix, "convert tweetid(%s) to int failed", rt.TweetId)
		return nil
	}
	t.TweetId = tid

	cid, cidok := util.String2Int64(rt.ConversationId)
	if !cidok {
		logger.LogImportant(logPrefix, "convert conversationId(%s) to int failed", rt.ConversationId)
		return nil
	}
	t.ConversationId = cid

	t.Lang = rt.Lang
	t.URL = fmt.Sprintf("https://twitter.com/%s/status/%s", ui.UserName, rt.TweetId)

	// 正文中提取转发者
	if strings.Index(rt.Text, "RT") == 0 {
		i0 := strings.Index(rt.Text, "@")
		i1 := strings.Index(rt.Text, ":")
		t.RetweetFrom = rt.Text[i0+1 : i1]
		t.Text = rt.Text[i1+2:]
	} else {
		t.Text = rt.Text
	}

	// 去掉正文中的链接
	exp := regexp.MustCompile(`(http|ftp|https):\/\/[\w\-_]+(\.[\w\-_]+)+([\w\-\.,@?^=%&:/~\+#]*[\w\-\@?^=%&/~\+#])?`)
	t.Text = string(exp.ReplaceAll([]byte(t.Text), []byte("")))

	// 解析时间
	if tm, err := time.Parse("2006-01-02T15:04:05.000Z", rt.CreatedAt); err == nil {
		t.CreatedAt = tm
	} else {
		logger.LogImportant(logPrefix, "parse time(%s) failed", rt.CreatedAt)
		return nil
	}

	// 引用的推文
	for _, v := range rt.ReferencedTweets {
		var rt struct {
			Type string `json:"type"`
			Id   int64  `json:"id"`
		}

		rt.Type = v.Type
		rt.Id, _ = util.String2Int64(v.Id)

		t.ReferencedTweets = append(t.ReferencedTweets, rt)
	}

	return &t
}

// 注册AccountActivity的Webhook
func RegisterAccountActivityWebhook(envname, whurl string) {
	ep := fmt.Sprintf("https://api.twitter.com/1.1/account_activity/all/%s/webhooks.json", envname)
	method := "POST"
	headers := make(map[string]string)

	params := url.Values{}
	params.Set("url", whurl)
	epWithParam := ep + "?" + params.Encode()

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, params)
	network.HttpCall(epWithParam, method, "", headers, func(r *http.Response, err error) {
		if err != nil {
			logger.LogImportant(logPrefix, err.Error())
		}
	})
}

// OAuth1.0的签名过程
func OAuth1HeaderStr(method, baseURL, consumerKey, userAccessToken string, params url.Values) string {
	// auth内部参数
	params.Add("oauth_consumer_key", consumerKey)
	params.Add("oauth_nonce", fmt.Sprintf("%d", time.Now().UnixMilli())) // 用时间戳作为nonce
	params.Add("oauth_signature_method", "HMAC-SHA1")
	params.Add("oauth_timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	params.Add("oauth_token", userAccessToken)
	params.Add("oauth_version", "1.0")
	pstr := params.Encode()
	signBaseStr := fmt.Sprintf("%s&%s&%s",
		method,
		url.QueryEscape(baseURL),
		url.QueryEscape(pstr))

	signKey := fmt.Sprintf("%s&%s", url.QueryEscape(consumerSecret), url.QueryEscape(userAccessTokenSecret))
	params.Add("oauth_signature", hmacshar1(signBaseStr, signKey))

	sb := strings.Builder{}
	sb.WriteString("OAuth ")
	for k, v := range params {
		if strings.Index(k, "oauth_") == 0 {
			sb.WriteString(fmt.Sprintf(`%s="%s", `, url.QueryEscape(k), url.QueryEscape(strings.Join(v, ","))))
		}
	}
	headerStr := sb.String() //[:sb.Len()-2]
	return headerStr
}
