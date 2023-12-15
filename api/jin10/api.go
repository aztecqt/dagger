/*
- @Author: aztec
- @Date: 2023-12-02 17:50:03
- @Description: 拉取jin10的重要经济数据、事件
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package jin10

import (
	"fmt"
	"time"

	"github.com/aztecqt/dagger/util/network"
)

var rootUrl = "https://cdn-rili.jin10.com/web_data"

// 获取经济数据（日期为东八区日期）
func GetEconomics(date time.Time) (*[]Economics, error) {
	url := fmt.Sprintf("%s/%d/daily/%02d/%02d/economics.json", rootUrl, date.Year(), date.Month(), date.Day())
	resp, err := network.ParseHttpResult[[]Economics](logPrefix, "GetEconomic", url, "GET", "", nil, nil, nil)
	if err == nil {
		for i, _ := range *resp {
			(*resp)[i].parse()
		}
	}
	return resp, err
}

// 获取经济事件（日期为东八区日期）
func GetEvents(date time.Time) (*[]Event, error) {
	url := fmt.Sprintf("%s/%d/daily/%02d/%02d/event.json", rootUrl, date.Year(), date.Month(), date.Day())
	return network.ParseHttpResult[[]Event](logPrefix, "GetEvents", url, "GET", "", nil, nil, nil)
}
