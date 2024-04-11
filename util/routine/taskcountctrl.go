/*
- @Author: aztec
- @Date: 2023-10-09 08:53:36
- @Description: 一个控制多任务并发的小工具，用于启动最大N个协程，来解决一组问题
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package routine

import (
	"sync"
	"time"
)

type TaskCountCtrl struct {
	mu           sync.Mutex
	taskCount    int
	maxTaskCount int
}

func NewTaskCountCtrl(maxCount int) *TaskCountCtrl {
	tc := &TaskCountCtrl{maxTaskCount: maxCount}
	return tc
}

func (t *TaskCountCtrl) WaitNextTask() {
	for {
		if t.taskCount < t.maxTaskCount {
			t.mu.Lock()
			t.taskCount++
			t.mu.Unlock()
			return
		} else {
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (t *TaskCountCtrl) Done() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taskCount--
}

func (t *TaskCountCtrl) Wait() {
	for t.taskCount > 0 {
		time.Sleep(time.Millisecond * 10)
	}
}

func (t *TaskCountCtrl) WaitWithTimeout(ms int64) {
	taskCountLast := t.taskCount
	stuckStartTime := time.Time{}
	for t.taskCount > 0 {
		if taskCountLast == t.taskCount {
			if stuckStartTime.IsZero() {
				stuckStartTime = time.Now()
			}
		} else {
			stuckStartTime = time.Time{}
		}

		if !stuckStartTime.IsZero() && time.Now().Sub(stuckStartTime).Milliseconds() > ms {
			break
		}

		taskCountLast = t.taskCount
		time.Sleep(time.Millisecond * 10)
	}
}
