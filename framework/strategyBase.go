/*
- @Author: aztec
- @Date: 2024-05-03 09:36:43
- @Description: 封装了策略程序的公共实现。包括profile解析，读取启动参数，启动、停止必要模块等
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package framework

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/aztecqt/center_server/csclient"
	"github.com/aztecqt/center_server/server/activestatus"
	"github.com/aztecqt/center_server/server/file"
	"github.com/aztecqt/center_server/server/intel"
	"github.com/aztecqt/dagger/api"
	"github.com/aztecqt/dagger/cex/binance"
	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/cex/okexv5"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/apikey"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/webservice"
)

type StrategyBase struct {
	running    bool
	LC         LaunchConfig
	Ex         common.CEx
	LogPrefix  string
	errorCount int
	csAddr     string

	// 消息通知服务
	IntelClient *csclient.IntelClient

	// 活动状态服务
	ActiveStatus *activestatus.Sender

	// 策略管理服务
	csClient *csclient.StratergyClientV2

	// web服务
	WebService *webservice.Service

	// 子类实现
	onCommand func(cmdLine string, onResp func(string))
	onQuit    func()
}

func (s *StrategyBase) Start(onStart func(), onQuit func(), onCmd func(cmdLine string, onResp func(string))) {
	s.onQuit = onQuit
	s.onCommand = onCmd

	// 根据profile加载启动参数
	configPath := ""
	profileDir, profileDirOk := util.GetProfileDir()
	if profileDirOk {
		configPath = fmt.Sprintf("%s/launch.json", profileDir)
	} else {
		fmt.Println("load profile failed")
		return
	}

	// 读取策略配置
	lc := LaunchConfig{}
	lc.ProfileRoot = profileDir
	lc.ParamPath = profileDir + "/param.json" // 这个是运行参数的路径，不是启动参数
	util.ObjectFromFile(configPath, &lc)
	s.LC = lc

	// 初始化Log
	logger.InitByStr(lc.LogConfig.PeriodConfig)
	logger.ConsleLogLevel = logger.String2LogLevel(lc.LogConfig.ConsoleLogLevel)
	logger.FileLogLevel = logger.String2LogLevel(lc.LogConfig.FileLogLevel)
	s.LogPrefix = fmt.Sprintf("%s.%s", lc.Class, lc.Name)

	// 获取apikey
	kreq := apikey.Requester{}
	if len(lc.Account) > 0 {
		logger.LogImportant(s.LogPrefix, "get apikey from server...")
		kreq.Go(lc.ExchangeName, lc.Account, lc.Name, lc.Key.Share, lc.Key.ServerAddr, lc.Key.ServerPort)
		logger.LogImportant(s.LogPrefix, "get apikey success")
	} else {
		logger.LogImportant(s.LogPrefix, "no need to get apikey")
	}

	// 创建交易所对象
	logger.LogImportant(s.LogPrefix, "starting exchange %s ...", lc.ExchangeName)
	if strings.ToLower(lc.ExchangeName) == "okex" {
		okex := new(okexv5.Exchange)
		okexv5.StratergyName = lc.Name // 用于标识订单归属
		var excfg *okexv5.ExchangeConfig
		if lc.ExchangeConfig != nil {
			excfg = &okexv5.ExchangeConfig{}
			b, _ := json.Marshal(lc.ExchangeConfig)
			if json.Unmarshal(b, excfg) != nil {
				excfg = nil
			}
		}
		okex.Init(kreq.Key, kreq.Secret, kreq.Password, excfg, s.errorNotifier)
		s.Ex = okex
	} else if strings.ToLower(lc.ExchangeName) == "binance" {
		binance := new(binance.Exchange)
		binance.Init(kreq.Key, kreq.Secret, s.errorNotifier)
		s.Ex = binance
	} else {
		logger.LogPanic("unknown exchange: %s", lc.ExchangeName)
	}
	logger.LogImportant(s.LogPrefix, "%s started", lc.ExchangeName)

	// 启动命令行
	s.runTerminal()

	// cs相关组件
	s.csAddr = fmt.Sprintf("http://%s:%d", lc.CsConfig.Addr, lc.CsConfig.HttpPort)
	s.IntelClient = csclient.NewIntelClient(s.csAddr)
	s.ActiveStatus = activestatus.NewSender(s.csAddr)
	s.csClient = &csclient.StratergyClientV2{}
	s.csClient.Start(
		lc.CsConfig.Addr,
		lc.CsConfig.UdpPort,
		lc.Name,
		lc.Class,
		s)

	// 启动PProf
	if lc.PProfPort > 0 {
		pprofAddr := fmt.Sprintf("localhost:%d", lc.PProfPort)
		go func(addr string) {
			logger.LogInfo(s.LogPrefix, "starting pprof at port %d", lc.PProfPort)
			if err := http.ListenAndServe(addr, nil); err != http.ErrServerClosed {
				logger.LogImportant(s.LogPrefix, "pprof server ListenAndServe error: %s", err.Error())
			}
		}(pprofAddr)
	}

	// 启动web服务
	if lc.WebServerPort > 0 {
		s.WebService = &webservice.Service{}
		s.WebService.Start(lc.WebServerPort)
		s.WebService.RegisterPath("/ping", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, util.Object2String(
				map[string]interface{}{
					"ts":    time.Now().UnixMilli(),
					"name":  s.Name(),
					"class": s.Class(),
				},
			))
		})
		logger.LogInfo(s.LogPrefix, "web-service started at port %d", lc.WebServerPort)
	}

	// 启动策略
	onStart()

	s.running = true
	for s.running {
		time.Sleep(time.Second)
	}
}

func (s *StrategyBase) StartUploader(path string, intervalSec int) {

	// html目录设置为自动上传
	uploader := file.Uploader{}
	uploader.Init(s.csAddr, path, fmt.Sprintf("%s/%s", s.LC.Class, s.LC.Name), "", intervalSec)
}

// 本地命令行逻辑
func (s *StrategyBase) runTerminal() {
	// 本地命令行输入
	go func() {
		defer util.DefaultRecover()
		input := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print(">")
			input.Scan()
			line := input.Text()
			s.OnCommand(line, func(resp string) {
				fmt.Println(resp)
			})
		}
	}()

	go func() {
		// 监控两个信号
		// TERM信号（kill + 进程号 触发）
		// 中断信号（ctrl + c 触发）
		osc := make(chan os.Signal, 1)
		signal.Notify(osc, syscall.SIGTERM, syscall.SIGINT)
		<-osc
		s.Quit(func(resp string) {
			fmt.Println(resp)
		})
	}()
}

// 错误通知
func (s *StrategyBase) errorNotifier(e error) {
	s.errorCount++
	it := intel.Intel{
		Time:     time.Now(),
		Level:    0,
		Type:     "stratergy",
		SubType:  s.LC.Name,
		DingType: "",
		Title:    fmt.Sprintf("策略异常(n=%d)", s.errorCount),
		Content:  e.Error(),
	}
	s.IntelClient.SendIntel(it)

	if s.errorCount%100 == 0 {
		it := intel.Intel{
			Time:     time.Now(),
			Level:    0,
			Type:     "stratergy",
			SubType:  s.LC.Name,
			DingType: intel.DingType_Text,
			Title:    "策略异常",
			Content:  "累计错误数量已达100条，请注意查看",
		}
		s.IntelClient.SendIntel(it)
	}
}

// #region 实现strategy接口
func (s *StrategyBase) Name() string {
	return s.LC.Name
}

func (s *StrategyBase) Class() string {
	return s.LC.Class
}

// 同时也实现terminal.Terminal接口
func (s *StrategyBase) OnCommand(cmdLine string, onResp func(string)) {
	switch cmdLine {
	case "help":
		sb := strings.Builder{}
		sb.WriteString("cs:             print put all call stack\n")
		sb.WriteString("exit/quit:      stop stratergy and quit\n")
		sb.WriteString("wslog:          switch websocket log on/off\n")
		s.onCommand("help", func(resp string) {
			sb.WriteString("\n")
			sb.WriteString(resp)
		})
		onResp(sb.String())
	case "cs":
		pf := pprof.Lookup("goroutine")
		pf.WriteTo(os.Stdout, 1)

		file, err := os.Create("log/callstack.log")
		if err == nil {
			pf.WriteTo(file, 1)
			file.Close()
			onResp(fmt.Sprintf("call stack saved to %s\n", file.Name()))
		} else {
			onResp("call stack save to file failed")
		}
	case "wslog":
		api.LogWebsocketDetail = !api.LogWebsocketDetail
		if api.LogWebsocketDetail {
			onResp("websocket log switch to on")
		} else {
			onResp("websocket log switch to off")
		}
	case "quit":
		s.Quit(onResp)
	default:
		s.onCommand(cmdLine, onResp)
	}
}

func (s *StrategyBase) Quit(onResp func(string)) {
	onResp("strategy quiting...")
	s.csClient.OnQuit()
	s.onQuit()
	onResp("strategy quited")
	time.Sleep(time.Second)
	s.running = false
}

// #endregion
