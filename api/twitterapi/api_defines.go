/*
 * @Author: aztec
 * @Date: 2022-11-27 17:56:48
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package twitterapi

import "time"

// 用户信息
type UserInfoCore struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserName string `json:"username"`
}

func (u *UserInfoCore) valid() bool {
	return len(u.ID) > 0 && len(u.Name) > 0 && len(u.UserName) > 0
}

// 用户信息
type UserInfo struct {
	Data UserInfoCore `json:"data"`
}

// 用户信息列表
type UserInfoList struct {
	Data []UserInfoCore `json:"data"`
	Meta struct {
		ResultCount   int    `json:"result_count"`
		NextToken     string `json:"next_token"`
		PreviousToken string `json:"previous_token"`
	}
}

func (u *UserInfoList) ContainsId(id string) bool {
	for _, uic := range u.Data {
		if uic.ID == id {
			return true
		}
	}

	return false
}

// api返回的原始推文
type RawTweet struct {
	AuthorId         string `json:"author_id"`
	ConversationId   string `json:"conversation_id"`
	TweetId          string `json:"id"`
	Lang             string `json:"lang"`
	Text             string `json:"text"`
	CreatedAt        string `json:"created_at"`
	ReferencedTweets []struct {
		Type string `json:"type"`
		Id   string `json:"id"`
	} `json:"referenced_tweets"`
}

// 原始推文列表
type RawTweetList struct {
	Data []RawTweet `json:"data"`
}

// 加工后的推文
type RefinedTweet struct {
	AuthorUserName   string    `json:"author_username"`
	Author           string    `json:"author"`
	RetweetFrom      string    `json:"retweet_from"`
	ConversationId   int64     `json:"conversation_id"`
	TweetId          int64     `json:"id"`
	Lang             string    `json:"lang"`
	Text             string    `json:"text"`
	CreatedAt        time.Time `json:"created_at"`
	URL              string    `json:"url"`
	ReferencedTweets []struct {
		Type string `json:"type"`
		Id   int64  `json:"id"`
	} `json:"referenced_tweets"`
}

// 关注结果
type FollowResponse struct {
	Data struct {
		Following     bool `json:"following"`
		PendingFollow bool `json:"pending_follow"`
	} `json:"data"`
}

// 取消关注结果
type UnfollowResponse struct {
	Data struct {
		Following bool `json:"following"`
	} `json:"data"`
}
