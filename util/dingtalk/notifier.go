/*
 * @Author: aztec
 * @Date: 2022-06-05 15:21
 * @FilePath: \stratergy_antc:\work\svn\quant\go\src\dagger\util\dingtalk\notifier.go
 * @Description: 封装钉钉的工作消息推送功能
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package dingtalk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/network"
)

const logPrefixRest = "ding-rest"
const rootURL = "https://oapi.dingtalk.com"

type NotifierConfig struct {
	Name    string `json:"name"`
	AgentId int64  `json:"agent_id"`
	Key     string `json:"key"`
	Secret  string `json:"secret"`
}

type Notifier struct {
	logPrefix     string
	logPrefixRest string
	agentId       int64
	key           string
	secret        string
	accessToken   string

	mob2uid map[int64]string // 手机号-userid的映射
}

func (n *Notifier) Init(cfg NotifierConfig) {
	n.logPrefix = "DingNotifier-" + cfg.Name
	n.agentId = cfg.AgentId
	n.key = cfg.Key
	n.secret = cfg.Secret
	n.mob2uid = make(map[int64]string)

	// 获取并维持accessToken
	if !n.refreshAccessToken() {
		logger.LogPanic(n.logPrefix, "get access token failed")
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		for {
			<-ticker.C
			n.refreshAccessToken()
		}
	}()
}

func (n *Notifier) SendTextByMob(text string, mobs ...int64) *Message {
	uids := make([]string, 0)
	for _, mob := range mobs {
		uid, ok := n.mobile2UserId(mob)
		if ok {
			uids = append(uids, uid)
		}
	}

	return n.SendTextByUid(text, uids...)
}

func (n *Notifier) SendTextByUid(text string, uids ...string) *Message {
	if len(uids) > 0 {
		m := new(Message)
		uidstr := strings.Join(uids, ",")
		m.initAsText(n, uidstr, text)
		logger.LogInfo(n.logPrefix, "sending text msg to user, userid=%s, text=%s", uidstr, text)
		go m.send()
		return m
	}

	return nil
}

// rawText: title, markdown, btnText0, btnUrl0...
func (n *Notifier) SendActionCardByMob(rawText []string, mobs ...int64) *Message {
	uids := make([]string, 0)
	for _, mob := range mobs {
		uid, ok := n.mobile2UserId(mob)
		if ok {
			uids = append(uids, uid)
		}
	}

	return n.SendActionCardByUid(rawText, uids...)
}

func (n *Notifier) SendActionCardByUid(rawText []string, uids ...string) *Message {
	if len(uids) > 0 {
		m := new(Message)
		uidstr := strings.Join(uids, ",")
		m.initAsActionCard(n, uidstr, rawText)
		logger.LogInfo(n.logPrefix, "sending action_card msg to user, userid=%s, rawtext=%s", uidstr, rawText)
		go m.send()
		return m
	}

	return nil
}

func (n *Notifier) SendLinkByMob(url, pic, title, text string, mobs ...int64) *Message {
	uids := make([]string, 0)
	for _, mob := range mobs {
		uid, ok := n.mobile2UserId(mob)
		if ok {
			uids = append(uids, uid)
		}
	}

	return n.SendLinkByUid(url, pic, title, text, uids...)
}

func (n *Notifier) SendLinkByUid(url, pic, title, text string, uids ...string) *Message {
	if len(uids) > 0 {
		m := new(Message)
		uidstr := strings.Join(uids, ",")
		m.intiAsLink(n, uidstr, url, pic, title, text)
		logger.LogInfo(n.logPrefix, "sending link msg to user, userid=%s, title=%s, url=%s", uidstr, title, url)
		go m.send()
		return m
	}

	return nil
}

func (n *Notifier) mobile2UserId(mob int64) (string, bool) {
	if uid, ok := n.mob2uid[mob]; ok {
		return uid, true
	} else {
		uid, ok := n.getUserId(mob)
		if ok {
			n.mob2uid[mob] = uid
			return uid, true
		} else {
			return "", false
		}
	}
}

func (n *Notifier) refreshAccessToken() bool {
	defer util.DefaultRecover()
	action := "/gettoken"
	method := "GET"
	params := url.Values{}
	params.Set("appkey", n.key)
	params.Set("appsecret", n.secret)
	action = action + "?" + params.Encode()
	url := rootURL + action
	resp, err := network.ParseHttpResult[getTokenResp](n.logPrefix, "refreshAccessToken", url, method, "", network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			n.accessToken = resp.AccessToken
			logger.LogInfo(n.logPrefix, "access token refreshed!")
			return true
		} else {
			logger.LogImportant(n.logPrefix, "refresh access token failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			return false
		}
	} else {
		return false
	}
}

func (n *Notifier) getUserId(mob int64) (uid string, ok bool) {
	defer util.DefaultRecover()
	action := "/topapi/v2/user/getbymobile"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := getByMobileReq{
		Mobile:                        fmt.Sprintf("%d", mob),
		SupportExclusiveAccountSearch: true,
	}
	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[getByMoblieResp](n.logPrefix, "getUserId", url, method, postStr, network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			uid = resp.Result.UserID
			ok = true
			logger.LogInfo(n.logPrefix, "userid for %d is %s", mob, uid)
		} else {
			ok = false
			logger.LogImportant(n.logPrefix, "get user id failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
		}
	}

	return
}

func (n *Notifier) sendTextMessage(uid string, text string) int64 {
	defer util.DefaultRecover()
	action := "/topapi/message/corpconversation/asyncsend_v2"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := sendTextMsgReq{}
	req.AgentId = n.agentId
	req.UseridList = uid
	req.ToAllUser = false
	req.Msg.MsgType = "text"
	req.Msg.Text.Content = text

	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[sendMsgResp](n.logPrefix, "sendTextMessage", url, method, postStr, network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			return resp.TaskId
		} else {
			logger.LogImportant(n.logPrefix, "send text msg failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			return 0
		}
	} else {
		return 0
	}
}

func (n *Notifier) sendLinkMessage(uid string, messageUrl, picUrl, title, text string) int64 {
	defer util.DefaultRecover()
	action := "/topapi/message/corpconversation/asyncsend_v2"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := sendLinkMsgReq{}
	req.AgentId = n.agentId
	req.UseridList = uid
	req.ToAllUser = false
	req.Msg.MsgType = "link"
	req.Msg.Link.MessageUrl = messageUrl
	req.Msg.Link.PicUrl = picUrl
	req.Msg.Link.Title = title
	req.Msg.Link.Text = text

	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[sendMsgResp](n.logPrefix, "sendLinkMessage", url, method, postStr, network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			return resp.TaskId
		} else {
			logger.LogImportant(n.logPrefix, "send link msg failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			return 0
		}
	} else {
		return 0
	}
}

func (n *Notifier) sendActionCardMessage(uid string, title, markdown string, btnTextsAndUrls []string) int64 {
	// 最大1000个字符，过长需要拆分
	lTitle := len(title) + len(markdown)
	btnTextsAndUrlsArray := [][]string{}
	temp := []string{}
	lTotal := lTitle
	for i := 0; i < len(btnTextsAndUrls); i += 2 {
		text := btnTextsAndUrls[i]
		url := btnTextsAndUrls[i+1]
		lTotal = lTotal + len(text) + len(url)
		if lTotal < 800 {
			temp = append(temp, text)
			temp = append(temp, url)
		} else {
			btnTextsAndUrlsArray = append(btnTextsAndUrlsArray, temp)
			temp = []string{}
			lTotal = lTitle
			lTotal = lTotal + len(text) + len(url)
			temp = append(temp, text)
			temp = append(temp, url)
		}
	}

	if len(temp) > 0 {
		btnTextsAndUrlsArray = append(btnTextsAndUrlsArray, temp)
	}

	id := int64(0)
	for _, v := range btnTextsAndUrlsArray {
		id = n.doSendActionCardMessage(uid, title, markdown, v)
	}
	return id
}

// title是卡片标题
// btnTextsAndUrls长度应为偶数，为 按钮文本、按钮url...
func (n *Notifier) doSendActionCardMessage(uid string, title, markdown string, btnTextsAndUrls []string) int64 {
	defer util.DefaultRecover()
	action := "/topapi/message/corpconversation/asyncsend_v2"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := sendActionCardMsgReq{}
	req.AgentId = n.agentId
	req.UseridList = uid
	req.ToAllUser = false
	req.Msg.MsgType = "action_card"
	req.Msg.ActionCard.Title = title
	req.Msg.ActionCard.Markdown = markdown
	req.Msg.ActionCard.BtnOrientation = "0" // 仅支持纵向
	if len(btnTextsAndUrls)%2 == 0 {
		for i := 0; i < len(btnTextsAndUrls)/2; i++ {
			title := btnTextsAndUrls[i*2]
			url := btnTextsAndUrls[i*2+1]
			// 转换为钉钉格式的url，禁用pc侧边栏
			url = fmt.Sprintf("dingtalk://dingtalkclient/page/link?url=%s&pc_slide=false", url)
			req.Msg.ActionCard.Btns = append(req.Msg.ActionCard.Btns, actionCardButton{Title: title, Url: url})
		}
	}

	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[sendMsgResp](n.logPrefix, "sendTextMessage", url, method, postStr, network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			return resp.TaskId
		} else {
			logger.LogImportant(n.logPrefix, "send text msg failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			return 0
		}
	} else {
		return 0
	}
}

func (n *Notifier) sendFinished(taskId int64) (finished, ok bool) {
	defer util.DefaultRecover()
	action := "/topapi/message/corpconversation/getsendprogress"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := sendProgressReq{
		AgentId: n.agentId,
		TaskId:  taskId,
	}
	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[sendProgressResp](n.logPrefix, "sendFinished", url, method, postStr, network.JsonHeaders(), nil, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			return resp.Progress.Status == 2, true
		} else {
			logger.LogImportant(n.logPrefix, "getsendprogress failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			return false, false
		}
	} else {
		return false, false
	}
}

func (n *Notifier) sendResult(taskId int64) (result string, ok bool) {
	defer util.DefaultRecover()
	action := "/topapi/message/corpconversation/getsendresult"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := sendResultReq{
		AgentId: n.agentId,
		TaskId:  taskId,
	}
	b, _ := json.Marshal(req)
	postStr := string(b)

	resp, err := network.ParseHttpResult[sendResultResp](n.logPrefix, "sendResult", url, method, postStr, network.JsonHeaders(), func(resp *http.Response, body []byte) {
		result = string(body)
	}, nil)

	if err == nil {
		if resp.ErrorCode == 0 {
			ok = true
		} else {
			logger.LogImportant(n.logPrefix, "getsendresult failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			ok = false
		}
	} else {
		ok = false
	}

	return
}

func (n *Notifier) UploadMediaFile(filePath, fileType string) (result string, ok bool) {
	defer util.DefaultRecover()
	action := "/media/upload"
	method := "POST"
	params := url.Values{}
	params.Set("access_token", n.accessToken)
	params.Set("type", fileType)
	action = action + "?" + params.Encode()
	url := rootURL + action

	req := uploadMediaReq{
		Type:  fileType,
		Media: filePath,
	}
	b, _ := json.Marshal(req)
	postStr := string(b)

	headers := make(map[string]string)
	boundary := "BOUNDARY"
	headers["Content-Type"] = fmt.Sprintf("multipart/from-data; boundary=%s\r\n--%s\r\nContent-Disposition: form-data; name=\"media\"; filename=\"%s\"\r\n--%s--\r\n", boundary, boundary, filePath, boundary)
	fmt.Println(headers["Content-Type"])
	resp, err := network.ParseHttpResult[uploadMediaResp](n.logPrefix, "uploadMediaFile", url, method, postStr, headers, func(resp *http.Response, body []byte) {
		result = string(body)
	}, nil)
	if err == nil {
		if resp.ErrorCode == 0 {
			logger.LogImportant(n.logPrefix, "media upload success, media_id=%s", resp.MediaId)
			result = resp.MediaId
			ok = true
		} else {
			logger.LogImportant(n.logPrefix, "media upload failed, errCode=%d, errMsg=%s", resp.ErrorCode, resp.ErrorMsg)
			ok = false
		}
	} else {
		ok = false
	}

	return
}
