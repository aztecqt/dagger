/*
 * @Author: aztec
 * @Date: 2022-06-15 11:43
 * @LastEditors: aztec
 * @FilePath: \dagger\util\udpsocket\message.go
 * @Description:
 * 消息头定义
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package udpsocket

type Header struct {
	OP string `json:"op"`
}
