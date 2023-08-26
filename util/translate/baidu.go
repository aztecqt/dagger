/*
* @Author: aztec
* @Date: 2022-10-03
* @Description: 百度翻译api
 */
package translate

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"time"

	"aztecqt/dagger/util/logger"
	"aztecqt/dagger/util/network"
)

const (
	api    = "https://fanyi-api.baidu.com/api/trans/vip/translate"
	appid  = "20221013001389349"
	secret = "Z2LqoEPrLWX8kISrKkH3"
)

type baiduTransResult struct {
	From        string `json:"from"`
	To          string `json:"to"`
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func BaiduTranslate(src string) (string, bool) {
	salt := fmt.Sprintf("%d", time.Now().UnixMilli())
	toSign := appid + src + salt + secret
	sign := fmt.Sprintf("%x", md5.Sum([]byte(toSign)))

	param := url.Values{}
	param.Set("q", src)
	param.Set("from", "auto")
	param.Set("to", "zh")
	param.Set("appid", appid)
	param.Set("salt", salt)
	param.Set("sign", sign)

	url := api + "?" + param.Encode()
	resultstr := ""
	ok := false

	logger.LogImportant("translate-baidu", "calling, len=%d", len(src))
	rst, err := network.ParseHttpResult[baiduTransResult]("baidu-translate", "BaiduTranslate", url, "GET", "", nil, nil, nil)
	if err != nil {
		resultstr = fmt.Sprintf("call baidu translate error: %s", err.Error())
		logger.LogImportant("translate-baidu", resultstr)
		ok = false
	} else {
		for _, r := range rst.TransResult {
			resultstr += r.Dst
		}
		ok = true
	}

	return resultstr, ok
}
