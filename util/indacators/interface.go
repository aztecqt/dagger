/*
 * @Author: aztec
 * @Date: 2022-12-30 20:32:47
 * @LastEditors: aztec
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package indacators

import "aztecqt/dagger/stratergy"

type Indicator interface {
	Update()
	Rebuild()
}

type Band interface {
	Indicator
	Upper() *stratergy.DataLine
	Lower() *stratergy.DataLine
	Middle() *stratergy.DataLine
}
