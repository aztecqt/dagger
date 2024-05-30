/*
- @Author: aztec
- @Date: 2024-05-03 09:11:59
- @Description: 策略对象的接口定义
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package framework

type Strategy interface {
	Name() string
	Class() string
	OnCommand(cmdLine string, onResp func(string)) // 策略对命令行的处理(同时也是terminal.Terminal接口)
	Quit(onResp func(string))
}
