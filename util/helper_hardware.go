/*
- @Author: aztec
- @Date: 2024-02-02 19:00:11
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package util

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// 获取cpu占用率
func GetCpuUsage(d time.Duration) float64 {
	if res, err := cpu.Percent(d, false); err == nil {
		return res[0]
	} else {
		return 0
	}
}

// 获取内存信息
func GetMemInfo() (totalInG, pct float64) {
	if res, err := mem.VirtualMemory(); err == nil {
		totalInG = float64(res.Total) / 1e9
		pct = res.UsedPercent
	}
	return
}

var netIoBytesRecv = uint64(0)
var netIoBytesSend = uint64(0)
var netIoLastStatTime = time.Time{}

// 获取网络io速度，返回发送、接受速率。单位是MB/s
func GetNetIoSpeed() (sendSpeedMbps, recvSpeedMbps float64) {
	info, _ := net.IOCounters(true)
	send := uint64(0)
	recv := uint64(0)
	for _, v := range info {
		send += v.BytesSent
		recv += v.BytesRecv
	}

	if !netIoLastStatTime.IsZero() {
		timeInSec := time.Now().Sub(netIoLastStatTime).Seconds()
		deltaSend := float64(send-netIoBytesSend) / 1e6
		deltaRecv := float64(recv-netIoBytesRecv) / 1e6
		sendSpeedMbps = deltaSend / timeInSec
		recvSpeedMbps = deltaRecv / timeInSec
	}

	netIoBytesRecv = recv
	netIoBytesSend = send
	netIoLastStatTime = time.Now()
	return
}

// 获取网路io速度，直接返回字符串格式
func GetnetIoSpeedStr() (sendSpeedStr, recvSpeedStr string) {
	s, r := GetNetIoSpeed()
	sendSpeedStr = mbpsToString(s)
	recvSpeedStr = mbpsToString(r)
	return
}

func mbpsToString(mbps float64) string {
	if mbps > 1 {
		return fmt.Sprintf("%.1fMB/s", mbps)
	} else {
		return fmt.Sprintf("%.1fKB/s", mbps*1000)
	}
}
