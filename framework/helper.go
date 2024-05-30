/*
 * @Author: aztec
 * @Date: 2024-05-05 11:59:55
 * @Description:
 *
 * Copyright (c) 2024 by aztec, All Rights Reserved.
 */
package framework

import (
	"fmt"
	"net/http"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/webservice"
)

// from+limit的组合太常用了，提取成公共函数
func ReadFromAndLimit(w http.ResponseWriter, r *http.Request, limitDefault, limitMax int) (from int64, limit int, ok bool) {
	// 读取参数
	q := r.URL.Query()
	fromStr := q.Get("from")
	limitStr := q.Get("limit")
	from = 0
	limit = limitDefault
	ok = false

	if len(fromStr) > 0 {
		if v, vok := util.String2Int64(fromStr); vok {
			from = v
		} else {
			webservice.WriteError(w, "format error: from")
			return
		}
	}

	if len(limitStr) > 0 {
		if v, vok := util.String2Int(limitStr); vok {
			limit = v
		} else {
			webservice.WriteError(w, "format error: limit")
			return
		}
	}

	if limit > limitMax {
		webservice.WriteError(w, fmt.Sprintf("limit must <= %d", limitMax))
		return
	}

	ok = true
	return
}
