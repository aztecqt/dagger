/*
 * @Author: aztec
 * @Date: 2022-11-28 17:31:21
 * @Description: userinfo相关接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package twitterapi

import (
	"fmt"
	"net/url"

	"aztecqt/dagger/api"
	"aztecqt/dagger/util/network"
)

// 自己的userinfo
var selfUserInfo UserInfoCore

// id-userinfo的映射
// username-userinfo的映射
var id2userInfo map[string]UserInfoCore
var un2userInfo map[string]UserInfoCore

func bufferUserInfo(ui UserInfoCore) {
	id2userInfo[ui.ID] = ui
	un2userInfo[ui.UserName] = ui
}

func MyUserInfo() (UserInfoCore, bool) {
	if selfUserInfo.valid() {
		return selfUserInfo, true
	}

	ep := "https://api.twitter.com/2/users/me"
	method := "GET"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	ui, err := network.ParseHttpResult[UserInfo](logPrefix, "MyUserId", ep, method, "", headers, nil, nil)
	if err == nil {
		selfUserInfo = ui.Data
		bufferUserInfo(selfUserInfo)
		return ui.Data, true
	} else {
		return UserInfoCore{}, false
	}
}

func GetUserInfoById(id string) (UserInfoCore, bool) {
	if ui, ok := id2userInfo[id]; ok {
		return ui, true
	}

	ep := fmt.Sprintf("https://api.twitter.com/2/users/%s", id)
	method := "GET"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	ui, err := network.ParseHttpResult[UserInfo](logPrefix, "GetUserInfoById", ep, method, "", headers, nil, nil)

	if err == nil {
		bufferUserInfo(ui.Data)
		return ui.Data, true
	} else {
		return UserInfoCore{}, false
	}
}

func GetUserInfoByUserName(un string) (UserInfoCore, bool) {
	if ui, ok := un2userInfo[un]; ok {
		return ui, true
	}

	ep := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", un)
	method := "GET"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	ui, err := network.ParseHttpResult[UserInfo](logPrefix, "GetUserInfoByUserName", ep, method, "", headers, nil, nil)

	if err == nil {
		bufferUserInfo(ui.Data)
		return ui.Data, true
	} else {
		return UserInfoCore{}, false
	}
}

func commonGetUserList(ep, funcName string, maxResult string) *UserInfoList {
	method := "GET"
	headers := make(map[string]string)

	nextToken := ""
	uilist := UserInfoList{}
	for {
		param := url.Values{}
		param.Set("max_results", maxResult)
		if len(nextToken) > 0 {
			param.Set("pagination_token", nextToken)
		}
		epwithquery := ep + "?" + param.Encode()

		// OAuth 1.0
		headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, param)

		uilistpage, err := network.ParseHttpResult[UserInfoList](logPrefix, funcName, epwithquery, method, "", headers, nil, nil)
		if err == nil {
			for _, ui := range uilistpage.Data {
				id2userInfo[ui.ID] = ui
				uilist.Data = append(uilist.Data, ui)
			}

			if len(uilistpage.Meta.NextToken) == 0 {
				break
			} else {
				nextToken = uilistpage.Meta.NextToken
			}
		} else {
			return nil
		}
	}
	return &uilist
}

func ListMembers(listId string) *UserInfoList {
	return commonGetUserList(fmt.Sprintf("https://api.twitter.com/2/lists/%s/members", listId), "ListMembers", "100")
}

func FollowingMembers(id string) *UserInfoList {
	return commonGetUserList(fmt.Sprintf("https://api.twitter.com/2/users/%s/following", id), "FollowingMembers", "1000")
}

// 关注一个用户（仅对本人生效）
func Follow(id string) bool {
	selfUI, ok := MyUserInfo()
	if !ok {
		return false
	}

	ep := fmt.Sprintf("https://api.twitter.com/2/users/%s/following", selfUI.ID)
	method := "POST"
	headers := make(map[string]string)
	param := url.Values{}
	param.Set("target_user_id", id)
	postData := api.ToPostData(param)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	resp, err := network.ParseHttpResult[FollowResponse](logPrefix, "Follow", ep, method, postData, headers, nil, nil)
	if err == nil {
		return resp.Data.Following
	} else {
		return false
	}
}

// 取消关注一个用户（仅对本人生效）
func Unfollow(id string) bool {
	selfUI, ok := MyUserInfo()
	if !ok {
		return false
	}

	ep := fmt.Sprintf("https://api.twitter.com/2/users/%s/following/%s", selfUI.ID, id)
	method := "DELETE"
	headers := make(map[string]string)

	// OAuth 1.0
	headers["Authorization"] = OAuth1HeaderStr(method, ep, consumerKey, userAccessToken, url.Values{})

	resp, err := network.ParseHttpResult[UnfollowResponse](logPrefix, "Unfollow", ep, method, "", headers, nil, nil)
	if err == nil {
		return !resp.Data.Following
	} else {
		return false
	}
}
