/*
- @Author: aztec
- @Date: 2024-05-11 10:51:45
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package pprofutil

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"time"
)

var cpuProfileFiles = map[string]*os.File{}

func StartCPUProfile(id string) {
	path := fmt.Sprintf("cpu.%s.profile", id)
	f, _ := os.OpenFile(path, os.O_CREATE, os.ModePerm)
	pprof.StartCPUProfile(f)
	cpuProfileFiles[id] = f
}

func StopCPUProfile(id string) {
	isWin := runtime.GOOS == "windows"

	if f, ok := cpuProfileFiles[id]; ok {
		pprof.StopCPUProfile()
		f.Close()
		path := fmt.Sprintf("cpu.%s.profile", id)
		cmd := exec.Command("go", "tool", "pprof", "-svg", path)
		if b, err := cmd.CombinedOutput(); err == nil {
			svgPath := fmt.Sprintf("%s.svg", path)
			svg, _ := os.OpenFile(svgPath, os.O_CREATE, os.ModePerm)
			svg.Write(b)
			svg.Close()

			// windows下直接打开并删除，其他os保留svg文件
			if isWin {
				cmdSvg := exec.Command("explorer.exe", svgPath)
				cmdSvg.Run()
				time.Sleep(time.Second)
				os.Remove(svgPath)
			}
		} else {
			fmt.Println(string(b))
			fmt.Println(err.Error())
		}
		delete(cpuProfileFiles, id)
		os.Remove(path)
	}
}
