/*
 * @Author: aztec
 * @Date: 2022-10-20
 * @Description: 包内帮助函数
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package api

import (
	"encoding/json"
	"net/url"
	"strings"
)

func ToPostData(param url.Values) string {
	m := make(map[string]string)
	for k, v := range param {
		m[k] = strings.Join(v, ",")
	}

	b, _ := json.Marshal(m)
	return string(b)
}
