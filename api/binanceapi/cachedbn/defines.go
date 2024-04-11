/*
- @Author: aztec
- @Date: 2023-12-20 10:46:01
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedbn

import (
	"time"

	"github.com/aztecqt/dagger/api/binanceapi"
)

var logPrefix = "bn.cached"
var DisableCached = false

type fnKlineRaw func(symbol, interval string, t0, t1 time.Time, limit int) (*binanceapi.KLine, error)
type fnprg func(prg time.Time)
