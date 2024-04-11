/*
 * @Author: aztec
 * @Date: 2022-10-20 17:30:40
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package binanceapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
)

var ApiStatus map[string]interface{}
var MuApiStatus sync.Mutex

// 处理rest请求的头
func ProcessResponse(resp *http.Response, body []byte, apiType string) *ErrorMessage {
	if resp != nil {
		// 超频判断
		for keystr, value := range resp.Header {
			if strings.Contains(keystr, "X-Mbx-Used-Weight-") {
				periodstr := keystr[len("X-Mbx-Used-Weight-"):]
				weight := util.String2IntPanic(string(value[0]))

				if ApiStatus == nil {
					ApiStatus = make(map[string]interface{})
				}

				MuApiStatus.Lock()
				key := fmt.Sprintf("bn-api-weight-%s", apiType)
				ApiStatus[key] = weight
				MuApiStatus.Unlock()
				logger.LogImportant("binance_rest", "bn(%s) weight usage: %s: %d", key, periodstr, weight)
			}
		}

		// 尝试解析错误码
		bodystr := string(body[:20])
		if strings.Contains(bodystr, `"code"`) {
			errmsg := new(ErrorMessage)
			json.Unmarshal(body, errmsg)

			// 简单的超频保护
			if errmsg.Code == -1003 {
				time.Sleep(time.Second * 3)
			}
			return errmsg
		}
	}

	return nil
}
