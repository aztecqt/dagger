/*
 * @Author: aztec
 * @Date: 2022-12-12 18:01:06
 * @Description: 将仓位线性的映射到一个价格范围中。一般用于网格交易。合约现货都适用
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package adv

import (
	"fmt"

	"aztecqt/dagger/cex/common"
	"aztecqt/dagger/util"
	"github.com/shopspring/decimal"
)

type PriceMap2Position struct {
	px0, px1   float64 // 价格范围
	pos0, pos1 float64 // 仓位范围
	market     common.CommonMarket
}

func NewPriceMap2Position(market common.CommonMarket) *PriceMap2Position {
	p2p := new(PriceMap2Position)
	p2p.market = market
	return p2p
}

func (p *PriceMap2Position) SetPriceRange(px0, px1 float64) {
	p.px0 = p.market.AlignPriceNumber(decimal.NewFromFloat(px0)).InexactFloat64()
	p.px1 = p.market.AlignPriceNumber(decimal.NewFromFloat(px1)).InexactFloat64()
}

func (p *PriceMap2Position) SetPositionRange(pos0, pos1 float64) {
	p.pos0 = p.market.AlignSize(decimal.NewFromFloat(pos0)).InexactFloat64()
	p.pos1 = p.market.AlignSize(decimal.NewFromFloat(pos1)).InexactFloat64()
}

func (p *PriceMap2Position) Valid() bool {
	return (p.pos0 > 0 || p.pos1 > 0) && p.px0 > 0 && p.px1 > 0
}

func (p *PriceMap2Position) GetPosition(px float64) float64 {
	if p.px0 != p.px1 {
		r := util.ClampFloat((px-p.px0)/(p.px1-p.px0), 0, 1)
		return util.LerpFloat(p.pos0, p.pos1, r)
	} else {
		return p.pos0
	}
}

func (p *PriceMap2Position) String() string {
	if p.pos0 == 0 {
		return fmt.Sprintf("px[%v-%v] max %v", p.px0, p.px1, p.pos1)
	} else {
		return fmt.Sprintf("px[%v-%v] pos[%v-%v]", p.px0, p.px1, p.pos0, p.pos1)
	}
}
