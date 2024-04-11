/*
- @Author: aztec
- @Date: 2024-03-08 10:12:12
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package ibkrtws

import (
	"fmt"
	"strings"
)

// 盈透的股票可以看作现货，其也有股票名和交易货币种类的概念，对应baseCcy和quoteCcy
// 如IBIT-USD代表IBIT这支股票的USD交易对
// 盈透的TWSAPI本身并没有InstrumentId的要求，而是用一个数字编码（Contract Id）来唯一表示一个品种
// 这里转换为InstrumentId，仅仅是为了兼容我们自己的某些接口
// ibit,usd -> IBIT-USD
func SpotTypeToInstId(baseCcy, quoteCcy string) string {
	return fmt.Sprintf("%s-%s", strings.ToUpper(baseCcy), strings.ToUpper(quoteCcy))
}

// IBIT-USD -> ibit,usd
func InstIdToSpotType(instId string) (baseCcy, quoteCcy string) {
	ss := strings.Split(instId, "-")
	return strings.ToLower(ss[0]), strings.ToLower(ss[1])
}
