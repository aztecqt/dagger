/*
 * @Author: aztec
 * @Date: 2022-11-25 09:32:45
 * @Description: 动态map映射器。针对流式key输入，用有限的内存为无限的id提供尽量准确的映射
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package mathtools

type DynamicMap struct {
	map0     map[interface{}]interface{}
	map1     map[interface{}]interface{}
	mapindex int
	mapcap   int
}

func NewDynamicMap(cap int) *DynamicMap {
	d := new(DynamicMap)
	d.map0 = make(map[interface{}]interface{})
	d.map1 = make(map[interface{}]interface{})
	d.mapcap = cap
	return d
}

func (d *DynamicMap) Get(k interface{}) interface{} {
	if v, ok := d.map0[k]; ok {
		return v
	}

	if v, ok := d.map1[k]; ok {
		return v
	}

	return nil
}

func (d *DynamicMap) Set(k, v interface{}) {
	if d.mapindex == 0 {
		d.map0[k] = v
		if len(d.map0) > d.mapcap {
			d.map1 = make(map[interface{}]interface{})
			d.mapindex = 1
		}
	} else if d.mapindex == 1 {
		d.map1[k] = v
		if len(d.map1) > d.mapcap {
			d.map0 = make(map[interface{}]interface{})
			d.mapindex = 0
		}
	}
}
