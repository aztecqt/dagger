/*
- @Author: aztec
- @Date: 2024-03-01 15:52:01
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

type BitMask struct {
	mask int
}

func NewBitMask(mask int) BitMask {
	return BitMask{mask: mask}
}

func (b BitMask) Get(index int) bool {
	if index < 0 || index >= 32 {
		panic("index overrange")
	}

	return (b.mask & (1 << index)) != 0
}

func (b *BitMask) Set(index int, val bool) {
	if index < 0 || index >= 32 {
		panic("index overrange")
	}

	if val {
		b.mask |= 1 << index
	} else {
		b.mask &= ^(1 << index)
	}
}
