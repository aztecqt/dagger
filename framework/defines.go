/*
- @Author: aztec
- @Date: 2024-05-03 09:17:36
- @Description: 策略的启动参数
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package framework

type LaunchConfig struct {
	Name           string      `json:"name"`
	Class          string      `json:"class"`
	ExchangeName   string      `json:"ex"`
	Account        string      `json:"acc"`
	ExchangeConfig interface{} `json:"ex_cfg"`

	// 日志设置
	LogConfig struct {
		PeriodConfig    string `json:"period"`     // d7, h12, m120
		ConsoleLogLevel string `json:"console_lv"` // info, debug, important
		FileLogLevel    string `json:"file_lv"`    // info, debug, important
	} `json:"log"`

	// 交易密钥服务
	Key struct {
		ServerAddr string `json:"addr"`
		ServerPort int    `json:"port"`
		Share      bool   `json:"share"`
	} `json:"key"`

	// 中央服务器
	CsConfig struct {
		Addr     string `json:"addr"`      // 服务器地址
		HttpPort int    `json:"http_port"` // http端口
		UdpPort  int    `json:"udp_port"`  // udp端口
	} `json:"cs"`

	// pprof服务的端口。0表示不启动pprof
	PProfPort int `json:"pprof_port"`

	// web服务的端口号。用于搭建策略前端
	WebServerPort int `json:"web_port"`

	// 配置根目录
	ProfileRoot string

	// 参数不再在StratergyParam中配置，而是使用专门的param.json，配合load/save函数，方便参数的在线调整
	ParamPath string
}
