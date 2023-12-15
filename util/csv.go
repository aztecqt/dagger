/*
 * @Author: aztec
 * @Date: 2023-03-24 09:20:29
 * @Description: csv文件读取
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package util

import (
	"bufio"
	"io"
	"os"
	"strings"
)

type CsvDocument struct {
	ColNames []string
	Rows     [][]string
}

func (c *CsvDocument) Load(path string, firstLineAsRow bool) bool {
	fileHanle, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}

	defer fileHanle.Close()

	c.Rows = make([][]string, 0)
	reader := bufio.NewReader(fileHanle)

	// 按行处理txt
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if len(c.ColNames) == 0 {
			c.ColNames = strings.Split(string(line), ",")
			if firstLineAsRow {
				c.Rows = append(c.Rows, strings.Split(string(line), ","))
			}
		} else {
			c.Rows = append(c.Rows, strings.Split(string(line), ","))
		}
	}

	return true
}

func (c *CsvDocument) Save(path string) bool {
	fileHanle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return false
	}

	defer fileHanle.Close()

	sb := strings.Builder{}
	for i, v := range c.ColNames {
		sb.WriteString(v)
		if i != len(c.ColNames)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("\n")

	for i, row := range c.Rows {
		for i2, v := range row {
			sb.WriteString(v)
			if i2 != len(row)-1 {
				sb.WriteString(",")
			}
		}

		if i != len(c.Rows)-1 {
			sb.WriteString("\n")
		}
	}

	fileHanle.WriteString(sb.String())
	return true
}
