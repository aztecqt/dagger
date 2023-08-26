/*
 * @Author: aztec
 * @Date: 2022-05-19 16:10:00
 * @LastEditors: aztec
 * @LastEditTime: 2022-12-27 18:27:50
 * @FilePath: \stratergyc:\svn\quant\go\src\dagger\stratergy\terminal.go
 * @Description: terminal controller of stratergy
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package stratergy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"aztecqt/dagger/api"
	"aztecqt/dagger/util"
)

type Terminal struct {
	s Stratergy
}

func (t *Terminal) Run(s Stratergy) {
	defer util.DefaultRecover()
	t.s = s
	go func() {
		input := bufio.NewScanner(os.Stdin)
		for input.Scan() {
			line := input.Text()
			t.OnCommand(line, func(resp string) {
				fmt.Println(resp)
			})
		}
	}()

	// 监控两个信号
	// TERM信号（kill + 进程号 触发）
	// 中断信号（ctrl + c 触发）
	osc := make(chan os.Signal, 1)
	signal.Notify(osc, syscall.SIGTERM, syscall.SIGINT)
	<-osc
	quit(s, func(resp string) {
		fmt.Println(resp)
	})
}

// terminal.Terminal
func (t *Terminal) OnCommand(cmdLine string, onResp func(string)) {
	switch cmdLine {
	case "help":
		sb := strings.Builder{}
		sb.WriteString("cs:             print put all call stack\n")
		sb.WriteString("exit/quit:      stop stratergy and quit\n")
		sb.WriteString("wslog:          switch websocket log on/off\n")
		sb.WriteString("status:         show stratergy's current status\n")
		sb.WriteString("param:          show stratergy's current param\n")
		t.s.OnCommand("help", func(resp string) {
			sb.WriteString("===========stratergy cmd help===========\n")
			sb.WriteString(resp)
		})
		onResp(sb.String())
	case "cls":
		sb := strings.Builder{}
		for i := 0; i < 64; i++ {
			sb.WriteString("\n")
		}
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
	case "status":
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("stratergy: %s, class: %s\n", t.s.Name(), t.s.Class()))
		sb.WriteString("status:\n")
		b, e := json.MarshalIndent(t.s.Status(), "", "  ")
		status := string(b)
		if e == nil {
			sb.WriteString(string(b))
			sb.WriteString("\n")
		} else {
			sb.WriteString("get status failed\n")
		}

		file, err := os.Create("log/status.log")
		if err == nil {
			file.WriteString(status)
			file.Close()
			sb.WriteString(fmt.Sprintf("status saved to %s", file.Name()))
		} else {
			sb.WriteString("status save to file failed")
		}
		onResp(sb.String())
	case "param":
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("stratergy: %s, class: %s\n", t.s.Name(), t.s.Class()))
		sb.WriteString("param:\n")
		b, e := json.MarshalIndent(t.s.Params(), "", "  ")
		param := string(b)
		if e == nil {
			sb.WriteString(param)
			sb.WriteString("\n")
		} else {
			sb.WriteString("get param failed\n")
			onResp(sb.String())
			return
		}

		file, err := os.Create("log/param.log")
		if err == nil {
			file.WriteString(param)
			file.Close()
			sb.WriteString(fmt.Sprintf("param saved to %s", file.Name()))
		} else {
			sb.WriteString("param save to file failed")
		}
		onResp(sb.String())
	case "quit":
		quit(t.s, onResp)
	case "exit":
		quit(t.s, onResp)
	case "wslog":
		api.LogWebsocketDetail = !api.LogWebsocketDetail
		if api.LogWebsocketDetail {
			onResp("websocket log switch to on")
		} else {
			onResp("websocket log switch to off")
		}
	default:
		t.s.OnCommand(cmdLine, onResp)
	}
}

func quit(s Stratergy, onResp func(string)) {
	onResp("stratergy quiting...")
	s.Quit()
	onResp("stratergy quit ok")
	time.Sleep(time.Second)
}
