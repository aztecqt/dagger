/*
 * @Author: aztec
 * @Date: 2022-06-05 17:03
 * @FilePath: \stratergy_antc:\work\svn\quant\go\src\dagger\util\dingtalk\message.go
 * @Description: 一条推送消息
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package dingtalk

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/util/logger"
)

var msgIndex int

type MessageStatus int

const (
	MessageStatus_Sending MessageStatus = iota
	MessageStatus_Sended
	MessageStatus_Finished
	MessageStatus_Failed
)

type Message struct {
	ntf               *Notifier
	index             int
	logPrefix         string
	taskId            int64
	userid            string
	msgType           string
	text              string
	actionCardRawText []string
	urlUrl            string
	urlPic            string
	urlTitle          string
	status            MessageStatus
	errCount          int
}

func (m *Message) Status() MessageStatus {
	return m.status
}

func (m *Message) ErrorCount() int {
	return m.errCount
}

func (m *Message) initAsText(ntf *Notifier, userid string, text string) {
	m.index = msgIndex
	m.logPrefix = fmt.Sprintf("dingtalk-text-msg-%d", m.index)
	msgIndex++
	m.ntf = ntf
	m.userid = userid
	m.msgType = "text"
	m.text = fmt.Sprintf("%s\n\n%s", text, time.Now().Format("2006-01-02 15:04:05"))
	logger.LogInfo(m.logPrefix, "init as text, text=%s", text)
}

// text格式：cardTitle,cardMarkdown,btnTitle0,btcUrl0,btnTitle1,btnUrl1...
func (m *Message) initAsActionCard(ntf *Notifier, userid string, rawtext []string) {
	m.index = msgIndex
	m.logPrefix = fmt.Sprintf("dingtalk-actionCard-msg-%d", m.index)
	msgIndex++
	m.ntf = ntf
	m.userid = userid
	m.msgType = "action_card"
	m.actionCardRawText = rawtext
	logger.LogInfo(m.logPrefix, "init as action_card, raw_text=%s", rawtext)
}

func (m *Message) intiAsLink(ntf *Notifier, userid string, url, pic, title, text string) {
	m.index = msgIndex
	m.logPrefix = fmt.Sprintf("dingtalk-link-msg-%d", m.index)
	msgIndex++
	m.ntf = ntf
	m.userid = userid
	m.msgType = "link"
	m.urlUrl = url
	m.urlPic = pic
	m.urlTitle = title
	m.text = fmt.Sprintf("%s\n\n%s", text, time.Now().Format("2006-01-02 15:04:05"))
	logger.LogInfo(m.logPrefix, "init as url, title=%s, url=%s", title, url)
}

// 发送消息
// 如果发送失败，停一会儿再发
// 如果发送成功，追踪发送结果并输出日志
func (m *Message) send() {
	ticker := time.NewTicker(time.Second * 3)
	m.status = MessageStatus_Sending

	// 首次发送
	logger.LogInfo(m.logPrefix, "sending...")
	sendSuccess := m.dosend()
	if !sendSuccess {
		logger.LogInfo(m.logPrefix, "sending failed")
		m.errCount++
	} else {
		logger.LogInfo(m.logPrefix, "sending success")
		m.status = MessageStatus_Sended
	}

	// 发送进度是否结束，查询获得
	sendFinished := false

	for {
		<-ticker.C
		if m.errCount >= 10 {
			logger.LogImportant(m.logPrefix, "too many fail, abort")
			m.status = MessageStatus_Failed
			break
		} else {
			// 再次发送
			if !sendSuccess {
				logger.LogInfo(m.logPrefix, "resending...")
				sendSuccess = m.dosend()
				if !sendSuccess {
					logger.LogInfo(m.logPrefix, "sending failed")
					m.errCount++
				} else {
					logger.LogInfo(m.logPrefix, "sending success")
					m.status = MessageStatus_Sended
				}
			} else if !sendFinished {
				// 追踪接收状态
				logger.LogInfo(m.logPrefix, "cheding send progress")
				finished, ok := m.sendFinished()
				if ok {
					if finished {
						logger.LogInfo(m.logPrefix, "cheding send result")
						result, ok := m.sendResult()
						if ok {
							logger.LogInfo(m.logPrefix, "send finished, result: %s", result)
							m.status = MessageStatus_Finished
							break
						} else {
							m.errCount++
						}
					} else {
						logger.LogInfo(m.logPrefix, "send not finished....keep waiting")
					}
				} else {
					m.errCount++
				}
			}
		}
	}
}

func (m *Message) dosend() bool {
	taskId := int64(0)
	if m.msgType == "text" {
		taskId = m.ntf.sendTextMessage(m.userid, m.text)
	} else if m.msgType == "link" {
		taskId = m.ntf.sendLinkMessage(m.userid, m.urlUrl, m.urlPic, m.urlTitle, m.text)
	} else if m.msgType == "action_card" {
		ss := m.actionCardRawText
		if len(ss) >= 4 && len(ss)%2 == 0 {
			title := ss[0]
			markdown := ss[1]
			btnTextAndUrls := ss[2:]
			taskId = m.ntf.sendActionCardMessage(m.userid, title, markdown, btnTextAndUrls)
		} else {
			logger.LogImportant(m.logPrefix, "invalid msg format for action_cark: %s", m.text)
			return false
		}
	} else {
		logger.LogImportant(m.logPrefix, "unknown msg type: %s", m.msgType)
		return false
	}

	if taskId > 0 {
		m.taskId = taskId
		return true
	} else {
		return false
	}
}

func (m *Message) sendFinished() (finished, ok bool) {
	return m.ntf.sendFinished(m.taskId)
}

func (m *Message) sendResult() (result string, ok bool) {
	return m.ntf.sendResult(m.taskId)
}
