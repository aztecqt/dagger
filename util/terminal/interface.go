/*
 * @Author: aztec
 * @Date: 2022-06-15 11:43
 * @LastEditors: aztec
 * @FilePath: \center_serverc:\work\svn\go\src\dagger\util\terminal\interface.go
 * @Description:
 * 定义了一个命令行接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package terminal

import (
	"bufio"
	"fmt"
	"os"
)

type Terminal interface {
	OnCommand(cmdLine string, onResp func(string))
}

func Run(t Terminal) {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		line := input.Text()
		t.OnCommand(line, func(resp string) {
			fmt.Println(resp)
		})
	}
}
