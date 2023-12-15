/*
 * @Author: aztec
 * @Date: 2022-04-16 09:49:52
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2023-09-07 22:04:49
 * @FilePath: \dagger\stratergy\stratergy.go
 * @Description: 策略通用接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package stratergy

type Stratergy interface {
	Name() string
	Class() string
	Status() interface{}                                    // 返回策略状态
	Params() *Param                                         // 返回策略当前参数
	OnParamChanged(paramData []byte)                        // 检测到策略参数修改时调用
	OnCommand(cmdLine string, onResp func(string))          // 向策略发送指令(同时也是terminal.Terminal接口)
	OnQuantEvent(name string, param map[string]string) bool // 向策略发送事件，返回值代表是否处理
	Quit()
}

type Param struct {
	Version   int         `json:"ver"`
	AutoTrade bool        `json:"auto_trade"`
	Data      interface{} `json:"data"`
}
