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
				ApiStatus[fmt.Sprintf("bn-api-weight-%s", apiType)] = weight
				MuApiStatus.Unlock()
				logger.LogInfo("binance_rest", "bn(%s) weight usage: %s: %d", apiType, periodstr, weight)
			}
		}

		// 尝试解析错误码
		bodystr := string(body[:20])
		if strings.Contains(bodystr, `"code"`) {
			errmsg := new(ErrorMessage)
			json.Unmarshal(body, errmsg)
			return errmsg
		}
	}

	return nil
}
