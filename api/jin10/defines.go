/*
- @Author: aztec
- @Date: 2023-12-02 17:29:45
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package jin10

import "time"

var logPrefix = "jin10"

// 经济数据
type Economics struct {
	Star           int       `json:"star"`
	Id             int       `json:"id"`
	Name           string    `json:"name"`
	Country        string    `json:"country"`
	TimePeriod     string    `json:"time_period"`
	Unit           string    `json:"unit"`
	PreviousValue  string    `json:"previous"`
	ConsensusValue string    `json:"consensus"`
	ActualValue    string    `json:"actual"`
	PublishTime    time.Time `json:"pub_time"`
}

func (e *Economics) parse() {
	if e.Unit == "口" { // 一个奇怪的单位
		e.Unit = ""
	}
}

// 经济事件
type Event struct {
	Star    int       `json:"star"`
	Id      int       `json:"id"`
	Name    string    `json:"event_content"`
	Country string    `json:"country"`
	Time    time.Time `json:"event_time"`
	People  string    `json:"people"`
}
