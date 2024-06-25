/*
 * @Author: aztec
 * @Date: 2024-05-05 10:59:51
 * @Description: 一组数据线
 *
 * Copyright (c) 2024 by aztec, All Rights Reserved.
 */
package framework

type DataLines struct {
	Lines []*DataLine
}

func (d *DataLines) Init(names []string, dir string, maxLength int, intervalMs int64, autoSave bool) {
	d.Lines = make([]*DataLine, len(names))
	for i := range d.Lines {
		d.Lines[i] = &DataLine{}
		d.Lines[i].Init(names[i], maxLength, intervalMs, 0).WithFileDirAndPath(dir, names[i]+".dl", true)
	}
}

func (d *DataLines) Update(ms int64, vals []float64) {
	if len(vals) != len(d.Lines) {
		panic("vals and lines not match")
	}

	for i, dl := range d.Lines {
		dl.Update(ms, vals[i])
	}
}

func (d *DataLines) Save() {
	for _, dl := range d.Lines {
		dl.Save()
	}
}
