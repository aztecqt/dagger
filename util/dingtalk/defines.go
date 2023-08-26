/*
 * @Author: aztec
 * @Date: 2022-06-05 15:21
 * @FilePath: \dagger\util\dingtalk\defines.go
 * @Description: 封装钉钉的工作消息推送
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package dingtalk

type commonResp struct {
	ErrorCode int    `json:"errcode"`
	ErrorMsg  string `json:"errmsg"`
}

type getTokenResp struct {
	commonResp
	AccessToken string `json:"access_token"`
}

type getByMobileReq struct {
	Mobile                        string `json:"mobile"`
	SupportExclusiveAccountSearch bool   `json:"support_exclusive_account_search"`
}

type getByMoblieResp struct {
	commonResp
	Result struct {
		UserID string `json:"userid"`
	} `json:"result"`
}

type sendMsgReq struct {
	AgentId    int64  `json:"agent_id"`
	UseridList string `json:"userid_list"`
	ToAllUser  bool   `json:"to_all_user"`
}

type sendTextMsgReq struct {
	sendMsgReq
	Msg struct {
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	} `json:"msg"`
}

type sendLinkMsgReq struct {
	sendMsgReq
	Msg struct {
		MsgType string `json:"msgtype"`
		Link    struct {
			MessageUrl string `json:"messageUrl"`
			PicUrl     string `json:"picUrl"`
			Title      string `json:"title"`
			Text       string `json:"text"`
		} `json:"link"`
	} `json:"msg"`
}

type sendMsgResp struct {
	commonResp
	TaskId int64 `json:"task_id"`
}

type sendProgressReq struct {
	AgentId int64 `json:"agent_id"`
	TaskId  int64 `json:"task_id"`
}

type sendProgressResp struct {
	commonResp
	Progress struct {
		Status int `json:"status"` // 0未开始/1处理中/2结束
	} `json:"progress"`
}

type sendResultReq struct {
	AgentId int64 `json:"agent_id"`
	TaskId  int64 `json:"task_id"`
}

type sendResultResp struct {
	commonResp
}

type uploadMediaReq struct {
	Type  string `json:"type"`
	Media string `json:"media"`
}

type uploadMediaResp struct {
	commonResp
	Type    string `json:"type"`
	MediaId string `json:"media_id"`
}
