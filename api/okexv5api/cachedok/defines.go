/*
- @Author: aztec
- @Date: 2023-12-19 10:48:50
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cachedok

import "time"

var logPrefix = "okexv5api.cached"
var DisableCached = false

type fnprg func(prg time.Time)
