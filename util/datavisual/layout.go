/*
 * @Author: aztec
 * @Date: 2023-07-20 17:43:44
 * @Description: 描述一个DataGroup的绘制方式
 *
 * Copyright (c) 2023 by aztec, All Rights Reserved.
 */
package datavisual

import (
	"fmt"

	"github.com/aztecqt/dagger/util"
)

type Color struct {
	R int
	G int
	B int
}

func (c *Color) String() string {
	return fmt.Sprintf("%d, %d, %d", c.R, c.G, c.B)
}

func (c Color) RGBA() (r, g, b, a uint32) {
	return uint32(c.R), uint32(c.G), uint32(c.B), 255
}

func NewColorFromRGB(R, G, B int) Color {
	c := Color{R: R, G: G, B: B}
	return c
}

var Color_AliceBlue = Color{R: 240, G: 248, B: 255}
var Color_AntiqueWhite = Color{R: 250, G: 235, B: 215}
var Color_Aqua = Color{R: 0, G: 255, B: 255}
var Color_Aquamarine = Color{R: 127, G: 255, B: 212}
var Color_Azure = Color{R: 240, G: 255, B: 255}
var Color_Beige = Color{R: 245, G: 245, B: 220}
var Color_Bisque = Color{R: 255, G: 228, B: 196}
var Color_Black = Color{R: 0, G: 0, B: 0}
var Color_BlanchedAlmond = Color{R: 255, G: 235, B: 205}
var Color_Blue = Color{R: 0, G: 0, B: 255}
var Color_BlueViolet = Color{R: 138, G: 43, B: 226}
var Color_Brown = Color{R: 165, G: 42, B: 42}
var Color_BurlyWood = Color{R: 222, G: 184, B: 135}
var Color_CadetBlue = Color{R: 95, G: 158, B: 160}
var Color_Chartreuse = Color{R: 127, G: 255, B: 0}
var Color_Chocolate = Color{R: 210, G: 105, B: 30}
var Color_Coral = Color{R: 255, G: 127, B: 80}
var Color_CornflowerBlue = Color{R: 100, G: 149, B: 237}
var Color_Cornsilk = Color{R: 255, G: 248, B: 220}
var Color_Crimson = Color{R: 220, G: 20, B: 60}
var Color_Cyan = Color{R: 0, G: 255, B: 255}
var Color_DarkBlue = Color{R: 0, G: 0, B: 139}
var Color_DarkCyan = Color{R: 0, G: 139, B: 139}
var Color_DarkGoldenrod = Color{R: 184, G: 134, B: 11}
var Color_DarkGray = Color{R: 169, G: 169, B: 169}
var Color_DarkGreen = Color{R: 0, G: 100, B: 0}
var Color_DarkKhaki = Color{R: 189, G: 183, B: 107}
var Color_DarkMagenta = Color{R: 139, G: 0, B: 139}
var Color_DarkOliveGreen = Color{R: 85, G: 107, B: 47}
var Color_DarkOrange = Color{R: 255, G: 140, B: 0}
var Color_DarkOrchid = Color{R: 153, G: 50, B: 204}
var Color_DarkRed = Color{R: 139, G: 0, B: 0}
var Color_DarkSalmon = Color{R: 233, G: 150, B: 122}
var Color_DarkSeaGreen = Color{R: 143, G: 188, B: 139}
var Color_DarkSlateBlue = Color{R: 72, G: 61, B: 139}
var Color_DarkSlateGray = Color{R: 47, G: 79, B: 79}
var Color_DarkTurquoise = Color{R: 0, G: 206, B: 209}
var Color_DarkViolet = Color{R: 148, G: 0, B: 211}
var Color_DeepPink = Color{R: 255, G: 20, B: 147}
var Color_DeepSkyBlue = Color{R: 0, G: 191, B: 255}
var Color_DimGray = Color{R: 105, G: 105, B: 105}
var Color_DodgerBlue = Color{R: 30, G: 144, B: 255}
var Color_Firebrick = Color{R: 178, G: 34, B: 34}
var Color_FloralWhite = Color{R: 255, G: 250, B: 240}
var Color_ForestGreen = Color{R: 34, G: 139, B: 34}
var Color_Fuchsia = Color{R: 255, G: 0, B: 255}
var Color_Gainsboro = Color{R: 220, G: 220, B: 220}
var Color_GhostWhite = Color{R: 248, G: 248, B: 255}
var Color_Gold = Color{R: 255, G: 215, B: 0}
var Color_Goldenrod = Color{R: 218, G: 165, B: 32}
var Color_Gray = Color{R: 128, G: 128, B: 128}
var Color_Green = Color{R: 0, G: 128, B: 0}
var Color_GreenYellow = Color{R: 173, G: 255, B: 47}
var Color_Honeydew = Color{R: 240, G: 255, B: 240}
var Color_HotPink = Color{R: 255, G: 105, B: 180}
var Color_IndianRed = Color{R: 205, G: 92, B: 92}
var Color_Indigo = Color{R: 75, G: 0, B: 130}
var Color_Ivory = Color{R: 255, G: 255, B: 240}
var Color_Khaki = Color{R: 240, G: 230, B: 140}
var Color_Lavender = Color{R: 230, G: 230, B: 250}
var Color_LavenderBlush = Color{R: 255, G: 240, B: 245}
var Color_LawnGreen = Color{R: 124, G: 252, B: 0}
var Color_LemonChiffon = Color{R: 255, G: 250, B: 205}
var Color_LightBlue = Color{R: 173, G: 216, B: 230}
var Color_LightCoral = Color{R: 240, G: 128, B: 128}
var Color_LightCyan = Color{R: 224, G: 255, B: 255}
var Color_LightGoldenrodYellow = Color{R: 250, G: 250, B: 210}
var Color_LightGray = Color{R: 211, G: 211, B: 211}
var Color_LightGreen = Color{R: 144, G: 238, B: 144}
var Color_LightPink = Color{R: 255, G: 182, B: 193}
var Color_LightSalmon = Color{R: 255, G: 160, B: 122}
var Color_LightSeaGreen = Color{R: 32, G: 178, B: 170}
var Color_LightSkyBlue = Color{R: 135, G: 206, B: 250}
var Color_LightSlateGray = Color{R: 119, G: 136, B: 153}
var Color_LightSteelBlue = Color{R: 176, G: 196, B: 222}
var Color_LightYellow = Color{R: 255, G: 255, B: 224}
var Color_Lime = Color{R: 0, G: 255, B: 0}
var Color_LimeGreen = Color{R: 50, G: 205, B: 50}
var Color_Linen = Color{R: 250, G: 240, B: 230}
var Color_Magenta = Color{R: 255, G: 0, B: 255}
var Color_Maroon = Color{R: 128, G: 0, B: 0}
var Color_MediumAquamarine = Color{R: 102, G: 205, B: 170}
var Color_MediumBlue = Color{R: 0, G: 0, B: 205}
var Color_MediumOrchid = Color{R: 186, G: 85, B: 211}
var Color_MediumPurple = Color{R: 147, G: 112, B: 219}
var Color_MediumSeaGreen = Color{R: 60, G: 179, B: 113}
var Color_MediumSlateBlue = Color{R: 123, G: 104, B: 238}
var Color_MediumSpringGreen = Color{R: 0, G: 250, B: 154}
var Color_MediumTurquoise = Color{R: 72, G: 209, B: 204}
var Color_MediumVioletRed = Color{R: 199, G: 21, B: 133}
var Color_MidnightBlue = Color{R: 25, G: 25, B: 112}
var Color_MintCream = Color{R: 245, G: 255, B: 250}
var Color_MistyRose = Color{R: 255, G: 228, B: 225}
var Color_Moccasin = Color{R: 255, G: 228, B: 181}
var Color_NavajoWhite = Color{R: 255, G: 222, B: 173}
var Color_Navy = Color{R: 0, G: 0, B: 128}
var Color_OldLace = Color{R: 253, G: 245, B: 230}
var Color_Olive = Color{R: 128, G: 128, B: 0}
var Color_OliveDrab = Color{R: 107, G: 142, B: 35}
var Color_Orange = Color{R: 255, G: 165, B: 0}
var Color_OrangeRed = Color{R: 255, G: 69, B: 0}
var Color_Orchid = Color{R: 218, G: 112, B: 214}
var Color_PaleGoldenrod = Color{R: 238, G: 232, B: 170}
var Color_PaleGreen = Color{R: 152, G: 251, B: 152}
var Color_PaleTurquoise = Color{R: 175, G: 238, B: 238}
var Color_PaleVioletRed = Color{R: 219, G: 112, B: 147}
var Color_PapayaWhip = Color{R: 255, G: 239, B: 213}
var Color_PeachPuff = Color{R: 255, G: 218, B: 185}
var Color_Peru = Color{R: 205, G: 133, B: 63}
var Color_Pink = Color{R: 255, G: 192, B: 203}
var Color_Plum = Color{R: 221, G: 160, B: 221}
var Color_PowderBlue = Color{R: 176, G: 224, B: 230}
var Color_Purple = Color{R: 128, G: 0, B: 128}
var Color_Red = Color{R: 255, G: 0, B: 0}
var Color_RosyBrown = Color{R: 188, G: 143, B: 143}
var Color_RoyalBlue = Color{R: 65, G: 105, B: 225}
var Color_SaddleBrown = Color{R: 139, G: 69, B: 19}
var Color_Salmon = Color{R: 250, G: 128, B: 114}
var Color_SandyBrown = Color{R: 244, G: 164, B: 96}
var Color_SeaGreen = Color{R: 46, G: 139, B: 87}
var Color_SeaShell = Color{R: 255, G: 245, B: 238}
var Color_Sienna = Color{R: 160, G: 82, B: 45}
var Color_Silver = Color{R: 192, G: 192, B: 192}
var Color_SkyBlue = Color{R: 135, G: 206, B: 235}
var Color_SlateBlue = Color{R: 106, G: 90, B: 205}
var Color_SlateGray = Color{R: 112, G: 128, B: 144}
var Color_Snow = Color{R: 255, G: 250, B: 250}
var Color_SpringGreen = Color{R: 0, G: 255, B: 127}
var Color_SteelBlue = Color{R: 70, G: 130, B: 180}
var Color_Tan = Color{R: 210, G: 180, B: 140}
var Color_Teal = Color{R: 0, G: 128, B: 128}
var Color_Thistle = Color{R: 216, G: 191, B: 216}
var Color_Tomato = Color{R: 255, G: 99, B: 71}
var Color_Turquoise = Color{R: 64, G: 224, B: 208}
var Color_Violet = Color{R: 238, G: 130, B: 238}
var Color_Wheat = Color{R: 245, G: 222, B: 179}
var Color_White = Color{R: 255, G: 255, B: 255}
var Color_WhiteSmoke = Color{R: 245, G: 245, B: 245}
var Color_Yellow = Color{R: 255, G: 255, B: 0}

// 描述一条线的名字、颜色等
// 名字对应DataGroup里Line的名字
type LineConfig struct {
	Field        string `json:"field"`
	lineColor    Color
	LineColorStr string `json:"color"`
	IsMain       bool   `json:"ismain"`
	Tag          string `json:"tag"`
}

// 一个画板包含一组Line和一组Points
type PaneConfig struct {
	Name   string       `json:"name"`
	Lines  []LineConfig `json:"lines"`
	Points []string     `json:"points"`
}

func NewPaneConfig(name string) *PaneConfig {
	p := new(PaneConfig)
	p.Name = name
	p.Lines = make([]LineConfig, 0)
	p.Points = make([]string, 0)
	return p
}

func (p *PaneConfig) AddLineConfig(field string, color Color, isMain bool, tag string) {
	lc := LineConfig{Field: field, lineColor: color, IsMain: isMain, Tag: tag}
	lc.LineColorStr = lc.lineColor.String()
	p.Lines = append(p.Lines, lc)
}

func (p *PaneConfig) AddPoints(field string) {
	p.Points = append(p.Points, field)
}

// 页面布局方式
type LayoutStyle int

const (
	Layout_Single LayoutStyle = iota
	Layout_Row2
	Layout_Square4
)

// 一个页面布局
type LayoutConfig struct {
	Style LayoutStyle   `json:"layout_style"`
	Panes []*PaneConfig `json:"panes"`
}

func NewLayoutConfig(style LayoutStyle) *LayoutConfig {
	lcfg := new(LayoutConfig)
	lcfg.Panes = make([]*PaneConfig, 0)
	lcfg.Style = style
	return lcfg
}

func (l *LayoutConfig) AddPane(pane *PaneConfig) {
	l.Panes = append(l.Panes, pane)
}

func (l *LayoutConfig) SaveToDir(dir string) {
	util.MakeSureDir(dir)
	util.ObjectToFile(fmt.Sprintf("%s/layout.json", dir), l)
}
