/*
 * @Author: aztec
 * @Date: 2022-04-21 15:43:12
 * @LastEditors: aztec
 * @LastEditTime: 2022-05-01 15:48:02
 * @FilePath: \dagger\util\code_path_recorder.go
 * @Description: 用来记录代码调用路径的一个小工具
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package util

import "github.com/emirpasic/gods/queues/linkedlistqueue"

type CodePathRecorder struct {
	UpdateHistoryStr string                 `json:"u_history"`
	UpdateHistory    *linkedlistqueue.Queue `json:"-"`
}

func (ss *CodePathRecorder) RecordUpdatePosition(pos int) {
	if ss.UpdateHistory == nil {
		ss.UpdateHistory = linkedlistqueue.New()
	}

	ss.UpdateHistory.Enqueue(pos)
	if ss.UpdateHistory.Size() > 50 {
		ss.UpdateHistory.Dequeue()
	}
	ss.UpdateHistoryStr = ss.UpdateHistory.String()
}
