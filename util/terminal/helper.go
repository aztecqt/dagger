/*
 * @Author: aztec
 * @Date: 2023-12-17 10:57:44
 * @Description: 命令行帮助函数
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package terminal

import (
	"bufio"
	"fmt"
	"os"

	"github.com/aztecqt/dagger/util"
	"github.com/jedib0t/go-pretty/v6/table"
)

func ReadIndex(min, max int) (int, bool) {
	fmt.Print(">")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	line := input.Text()
	if n, ok := util.String2Int(line); ok {
		if n >= min && n <= max {
			return n, true
		} else {
			fmt.Printf("index out of range[%d-%d]\n", min, max)
			return 0, false
		}
	} else {
		fmt.Printf("can't convert %s to int for index\n", line)
		return 0, false
	}
}

func ReadIndexTillSuccess(min, max int) int {
	input := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">")
		input.Scan()
		line := input.Text()
		if n, ok := util.String2Int(line); ok {
			if n >= min && n <= max {
				return n
			} else {
				fmt.Printf("index out of range[%d-%d]\n", min, max)
			}
		} else {
			fmt.Printf("can't convert %s to int for index\n", line)
		}
	}
}

func SelectItem(tips string, items []string) (string, bool) {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetAutoIndex(true)
	t.ResetHeaders()
	for _, v := range items {
		t.AppendRow(table.Row{v})
	}
	fmt.Println(t.Render())
	fmt.Println(tips)
	if index, ok := ReadIndex(1, len(items)); ok {
		return items[index-1], true
	} else {
		return "", false
	}
}

// 判断参数数量是否达标。ss[0]为操作类型
func CheckParamCount(ss []string, minCount int) bool {
	if len(ss) < minCount {
		fmt.Printf("not enough param for op '%s'\n", ss[0])
		return false
	}

	return true
}

// 解析参数
// -a 1 -b 2 -> map{-a:1, -b:2}
func ParseParamsToMap(ss []string, startIndex int) map[string]string {
	params := make(map[string]string)
	for i := startIndex; i < len(ss); {
		s0 := ss[i]
		s1 := ""
		if i+1 < len(ss) {
			s1 = ss[i+1]
		}

		if s0[0] == '-' {
			if len(s1) > 0 && s1[0] == '-' {
				params[s0] = "" // -xx后面跟下一个-yy，则认为-xx为空参数
				i++
			} else {
				params[s0] = s1 // 有效参数
				i += 2
			}
		} else {
			i++ // 跳过
		}
	}

	return params
}
