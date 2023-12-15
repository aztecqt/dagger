/*
 * @Author: aztec
 * @Date: 2022-11-25 09:32:45
 * @Description: id去重器。针对流式id输入，用有限的内存为无限的id提供尽量准确的去重
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package mathtools

import "github.com/emirpasic/gods/sets/hashset"

type Deduplicator struct {
	set0     *hashset.Set
	set1     *hashset.Set
	setindex int
	setcap   int
}

func NewDeduplicator(cap int) *Deduplicator {
	d := new(Deduplicator)
	d.set0 = hashset.New()
	d.set1 = hashset.New()
	d.setcap = cap
	return d
}

func (d *Deduplicator) IsDuplicated(v interface{}) bool {
	duplicated := false
	if d.set0.Contains(v) || d.set1.Contains(v) {
		duplicated = true
	}

	if d.setindex == 0 {
		d.set0.Add(v)
		if d.set0.Size() > d.setcap {
			d.set1 = hashset.New()
			d.setindex = 1
		}
	} else if d.setindex == 1 {
		d.set1.Add(v)
		if d.set1.Size() > d.setcap {
			d.set0 = hashset.New()
			d.setindex = 0
		}
	}

	return duplicated
}

func (d *Deduplicator) Empty() bool {
	return d.set0.Empty() && d.set1.Empty()
}
