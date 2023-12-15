/*
 * @Author: aztec
 * @Date: 2023-08-24 11:31:58
 * @Description: 行情驱动器接口
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package marketdata

import (
	"time"
)

type Ticker struct {
	Symbol string
	Buy1   float64
	Sell1  float64
}

type Driver interface {
	Run(fnUpdate func(now time.Time, tickers []Ticker))
	ShowProgress()
	Clone() Driver
	StartTime() time.Time
	EndTime() time.Time
}
