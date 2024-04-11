/*
- @Author: aztec
- @Date: 2024-03-05 17:57:45
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import (
	"fmt"
	"strings"
)

func tagValueListToString(tvs []TagValue) string {
	sb := strings.Builder{}
	for _, tv := range tvs {
		sb.WriteString(fmt.Sprintf("%s=%v;", tv.Tag, tv.Value))
	}
	return sb.String()
}
