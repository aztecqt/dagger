/*
- @Author: aztec
- @Date: 2023-12-04 09:57:54
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package prettyutil

import "fmt"

func TransformToPct(i interface{}) string {
	return fmt.Sprintf("%.2f%%", i.(float64)*100)
}
