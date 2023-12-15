/*
 * @Author: aztec
 * @Date: 2022-04-26 22:21:16
 * @LastEditors: aztec
 * @LastEditTime: 2022-04-27 09:20:51
 * @FilePath: \dagger\util\signals\push_n_hold.go
 * @Description:
 * 一种信号，当输入信号持续为true且持续一段时间后，输出为true；输入为false或输入时间不足，输出为false
 * 就好像一个按钮，只有按下去一段时间后才会亮灯，一松手就灭了
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package signals

import "time"

type PushAndHold struct {
	startTime  time.Time
	intervalMs int64
	cond       bool
}

func (p *PushAndHold) Init(ms int64) {
	p.intervalMs = ms
}

func (p *PushAndHold) Input(cond bool) bool {
	if cond {
		if !p.cond {
			p.cond = true
			p.startTime = time.Now()
		}
		return time.Now().UnixMilli()-p.startTime.UnixMilli() >= p.intervalMs
	} else {
		p.cond = false
		return false
	}
}
