/*
- @Author: aztec
- @Date: 2023-11-22 15:58:19
- @Description:
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package cam

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aztecqt/dagger/cex/binance"
	"github.com/aztecqt/dagger/cex/okexv5"
	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/datavisual"
	"github.com/shopspring/decimal"
)

type RespCommon struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (r RespCommon) Ok() bool {
	return r.Code == "ok" || r.Status == "ok"
}

type RespPong struct {
	RespCommon
}

type Fund struct {
	Alias string `json:"fund_alias"`
	Name  string `json:"fund_name"`
}

type RespFundList struct {
	RespCommon
	Data struct {
		AllFunds []Fund `json:"all_funds"`
	} `json:"data"`
}

type BasicInfo struct {
	BaseAsset string          `json:"asset_base_str"`    // 基础资产类型
	NetAsset  decimal.Decimal `json:"net_asset"`         // 净资产
	Nav       decimal.Decimal `json:"nav"`               // 单位净值
	Nav24hAgo decimal.Decimal `json:"nav_24h_ago"`       // 单位净值(24小时之前的值)
	Pnl7      decimal.Decimal `json:"last_7d_pnl_base"`  // 7日年化
	Pnl30     decimal.Decimal `json:"last_30d_pnl_base"` // 30日年化
}

type RespBasicInfo struct {
	RespCommon
	BasicInfo BasicInfo `json:"data"`
}

// 目前仅取Alias、Name，以及7/30日成交量。用来代替RespFundList
type FundDetailInfo struct {
	Alias        string          `json:"fund_alias"`
	Name         string          `json:"fund_name"`
	VolumeUsd7d  decimal.Decimal `json:"trade_volume_7d_usd"`
	VolumeUsd30d decimal.Decimal `json:"trade_volume_30d_usd"`
}

type FundDetailList struct {
	TotalNetAssetUsd decimal.Decimal  `json:"total_net_asset_usd"`
	Details          []FundDetailInfo `json:"fund_details"`
}

type RespFundDetailListInner struct {
	RespCommon
	DetailInfoList FundDetailList `json:"data"`
}

type RespFundDetailList struct {
	Status int                     `json:"status"` // 这个跟RespCommon不一样,0代表数据没准备好，1代表已经准备好了
	Data   RespFundDetailListInner `json:"data"`
}

type RespTaskid struct {
	RespCommon
	TaskId string `json:"task_id"`
}

type Asset struct {
	Asset  string          `json:"symbol_str"`
	Amount decimal.Decimal `json:"amount"`
	Price  decimal.Decimal `json:"price"`
	Equity decimal.Decimal `json:"total_equity_base"`
}

type AssetsOfAccount struct {
	Name         string          `json:"symbol_str"`
	Equity       decimal.Decimal `json:"total_equity_base"`
	EquityUsd    decimal.Decimal `json:"total_equity_usd"`
	NetAssetRate decimal.Decimal `json:"net_asset_rate"`
	Assets       []Asset         `json:"children"`
}

type AssetsOfExchange struct {
	Name         string            `json:"symbol_str"`
	Equity       decimal.Decimal   `json:"total_equity_base"`
	EquityUsd    decimal.Decimal   `json:"total_equity_usd"`
	NetAssetRate decimal.Decimal   `json:"net_asset_rate"`
	Accounts     []AssetsOfAccount `json:"children"`
}

type AssetsOfFund struct {
	BaseCurrency string             `json:"currency_str"`
	Alias        string             `json:"fund_alias"`
	Name         string             `json:"fund_name"`
	Equity       decimal.Decimal    `json:"total_equity_base"`
	EquityUsd    decimal.Decimal    `json:"total_equity_usd"`
	NetAssetRate decimal.Decimal    `json:"net_asset_rate"`
	Exchanges    []AssetsOfExchange `json:"asset_data"`
}

func (a *AssetsOfFund) Walk(fn func(ex, acc string, a Asset)) {
	for _, aoe := range a.Exchanges {
		for _, aoa := range aoe.Accounts {
			for _, ass := range aoa.Assets {
				fn(aoe.Name, aoa.Name, ass)
			}
		}
	}
}

type RespAssets struct {
	RespCommon
	Assets AssetsOfFund `json:"data"`
}

type Position struct {
	AmountWrap struct {
		Amount           decimal.Decimal `json:"amount"`
		ContractValue    decimal.Decimal `json:"contract_val"`
		ContractValueCcy string          `json:"contract_val_currency"`
		MarginType       string          `json:"margin_type"`
	} `json:"amount"`

	AvgPrice         decimal.Decimal `json:"average_entry_price"`
	MarkPrice        decimal.Decimal `json:"mark_price"`
	Symbol           string          `json:"symbol_str"`
	UnrealizedPnl    decimal.Decimal `json:"unrealized_pnl"`
	UnrealizedPnlCcy string          `json:"unrealized_pnl_currency_str"`
	ValueUsd         decimal.Decimal `json:"value_usd"`
}

type PositionsOfAccount struct {
	Name      string          `json:"symbol_str"`
	ValueUsd  decimal.Decimal `json:"value_usd"`
	Positions []Position      `json:"children"`
}

type PositionsOfExchange struct {
	Name     string               `json:"symbol_str"`
	ValueUsd decimal.Decimal      `json:"value_usd"`
	Accounts []PositionsOfAccount `json:"children"`
}

type PositionsOfFund struct {
	Alias     string                `json:"fund_alias"`
	Name      string                `json:"fund_name"`
	ValueUsd  decimal.Decimal       `json:"value_usd"`
	Exchanges []PositionsOfExchange `json:"position_data"`
}

func (p *PositionsOfFund) Walk(fn func(ex, acc string, p Position)) {
	for _, poe := range p.Exchanges {
		for _, poa := range poe.Accounts {
			for _, pos := range poa.Positions {
				fn(poe.Name, poa.Name, pos)
			}
		}
	}
}

func (p *PositionsOfFund) Standardize() {
	for _, poe := range p.Exchanges {
		for _, poa := range poe.Accounts {
			for i := range poa.Positions {
				poa.Positions[i].AmountWrap.ContractValueCcy = strings.ToUpper(poa.Positions[i].AmountWrap.ContractValueCcy)
			}
		}
	}
}

type RespPositions struct {
	RespCommon
	Positions PositionsOfFund `json:"data"`
}

type Risk struct {
	ExposureRate          decimal.Decimal `json:"fund_exposure_rate"`
	LeverageWithoutOption decimal.Decimal `json:"leverage_ratio_without_option"`
	LeverageWithOption    decimal.Decimal `json:"leverage_ratio_with_option"`
	SharpeRatio           decimal.Decimal `json:"sharpe_ratio"`
	Volatility            decimal.Decimal `json:"volatility"`
}

type RespRisk struct {
	RespCommon
	Risk Risk `json:"data"`
}

type OrderRecord struct {
	Exchange           string          `json:"account_venue"`
	AccountAlias       string          `json:"account_alias"`
	OrderTimeStampNano int64           `json:"order_time"`  //
	SymbolType         string          `json:"symbol_type"` // spot/swap
	Symbol             string          `json:"symbol"`      // binance/avax.usdt binance/avax.usdt.td binance/avax.usd.td
	Dir                string          `json:"direction"`   // b/s
	Amount             decimal.Decimal `json:"amount"`
	DealAmount         decimal.Decimal `json:"dealt_amount"`
	TotalValue         decimal.Decimal `json:"total_value"`
	Price              decimal.Decimal `json:"price"`
	AvgDealPrice       decimal.Decimal `json:"average_dealt_price"`
	Status             string          `json:"status"`

	BaseCcy    string
	IsMaker    bool
	OrderTime  time.Time
	IsUsdtSwap bool
	InstId     string // 符合交易所规范的instrumentId
}

func (o *OrderRecord) parse() {
	o.IsMaker = o.Price.Equal(o.AvgDealPrice)
	o.OrderTime = time.UnixMilli(o.OrderTimeStampNano / 1e6)

	// 是否为u本位合约
	if o.SymbolType == "swap" {
		if strings.Contains(o.Symbol, "usdt") {
			o.IsUsdtSwap = true
		} else {
			o.IsUsdtSwap = false
		}
	}

	// 计算baseCcy/instid
	ss0 := strings.Split(o.Symbol, "/")
	if len(ss0) == 2 {
		ss1 := strings.Split(ss0[1], ".")
		o.BaseCcy = ss1[0]

		if o.Exchange == "BINANCE" {
			if o.SymbolType == "swap" {
				if ss1[1] == "usdt" {
					o.InstId = binance.CCyCttypeToInstId(ss1[0], "usdt_swap")
				} else if ss1[1] == "usd" {
					o.InstId = binance.CCyCttypeToInstId(ss1[0], "usd_swap")
				}
			} else if o.SymbolType == "spot" {
				o.InstId = binance.SpotTypeToInstId(ss1[0], ss1[1])
			}
		} else if o.Exchange == "OKX" {
			if o.SymbolType == "swap" {
				if ss1[1] == "usdt" {
					o.InstId = okexv5.CCyCttypeToInstId(ss1[0], "usdt_swap")
				} else if ss1[1] == "usd" {
					o.InstId = okexv5.CCyCttypeToInstId(ss1[0], "usd_swap")
				}
			} else if o.SymbolType == "spot" {
				o.InstId = okexv5.SpotTypeToInstId(ss1[0], ss1[1])
			}
		}
	}

	// 修正TotalValue的计算方式。cam币本位合约的Value计算方式是错误的
	if strings.Contains(o.Symbol, "usd.td") {
		ContractValue := decimal.NewFromInt(10)
		if o.Symbol == "btc.usd.td" {
			ContractValue = decimal.NewFromInt(100)
		}
		o.TotalValue = o.DealAmount.Mul(ContractValue)
	}
}

func (o OrderRecord) ToDataVisualPoint() datavisual.Point {
	return datavisual.Point{
		Time:  o.OrderTime,
		Value: o.AvgDealPrice.InexactFloat64(),
		Tag:   util.ValueIf(o.Dir == "s", datavisual.PointTag_Sell, datavisual.PointTag_Buy)}
}

type RespOrderRecordInner struct {
	RespCommon
	OrderRecords []OrderRecord `json:"data"`
}

func (r *RespOrderRecordInner) parse() {
	for i := range r.OrderRecords {
		r.OrderRecords[i].parse()
	}
}

type RespOrderRecord struct {
	Status int                  `json:"status"`
	Data   RespOrderRecordInner `json:"data"`
}

type DealRecord struct {
	Exchange          string          `json:"exchange_alias"`
	AccountAlias      string          `json:"account_alias"`
	DealTimeStampNano int64           `json:"dealt_time"`
	OrderId           string          `json:"order_id"`
	SymbolType        string          `json:"symbol_type"`  // spot/swap
	Symbol            string          `json:"trading_pair"` // binance/avax.usdt binance/avax.usdt.td binance/avax.usd.td
	Dir               string          `json:"direction"`    // b/s
	Amount            decimal.Decimal `json:"dealt_amount"`
	Price             decimal.Decimal `json:"dealt_price"`
	Fee               decimal.Decimal `json:"commission"`
	FeeCcy            string          `json:"commission_ccy"`
	DealType          string          `json:"transaction_type"`

	TotalValue decimal.Decimal
	BaseCcy    string
	IsMaker    bool
	DealTime   time.Time
	IsUsdtSwap bool
	InstId     string // 符合交易所规范的instrumentId
}

func (d *DealRecord) Encode() string {
	getSymbolTypeVal := func(symbolType string) byte {
		if symbolType == "spot" {
			return 1
		} else if symbolType == "swap" {
			return 2
		} else {
			return 0
		}
	}
	getTradeDirVal := func(dir string) byte {
		if dir == "b" {
			return 1
		} else if dir == "s" {
			return 2
		} else {
			return 0
		}
	}
	getDealTypeVal := func(dealType string) byte {
		if dealType == "maker" {
			return 1
		} else if dealType == "taker" {
			return 2
		} else {
			return 0
		}
	}
	getCoinsBySymbol := func(symbol string) (string, string) {
		base := ""
		quote := ""
		words := strings.Split(symbol, "/")
		if len(words) > 0 {
			symbol = words[len(words)-1]
			words = strings.Split(symbol, ".")
			if len(words) >= 2 {
				base = words[0]
				quote = words[1]
			}
		}
		return base, quote
	}
	orderId := d.OrderId
	words := strings.Split(orderId, "-")
	if len(words) > 0 {
		orderId = words[len(words)-1]
	}
	baseCoin, quoteCoin := getCoinsBySymbol(d.Symbol)
	ord, _ := strconv.ParseUint(orderId, 10, 64)
	amount, _ := d.Amount.Float64()
	price, _ := d.Price.Float64()
	fee, _ := d.Fee.Float64()
	dataLen := 35
	bytes := make([]byte, 128)
	binary.BigEndian.PutUint64(bytes, ord)                             //订单号
	binary.LittleEndian.PutUint64(bytes[8:], math.Float64bits(amount)) //amount
	binary.LittleEndian.PutUint64(bytes[16:], math.Float64bits(price)) //price
	binary.LittleEndian.PutUint64(bytes[24:], math.Float64bits(fee))   //fee amount
	bytes[32] = getSymbolTypeVal(d.SymbolType)                         //SymbolType  0:spot, 1:swap
	bytes[33] = getTradeDirVal(d.Dir)                                  //Dir 0:buy, 1:sell
	bytes[34] = getDealTypeVal(d.DealType)                             //DealType 0:maker, 1:taker

	eBytes := []byte(d.Exchange)
	bytes[dataLen] = byte(len(eBytes))
	dataLen++
	copy(bytes[dataLen:], eBytes)
	dataLen = dataLen + len(eBytes)

	eBaseCoin := []byte(baseCoin)
	bytes[dataLen] = byte(len(eBaseCoin))
	dataLen++
	copy(bytes[dataLen:], eBaseCoin)
	dataLen = dataLen + len(eBaseCoin)

	eQuoteCoin := []byte(quoteCoin)
	bytes[dataLen] = byte(len(eQuoteCoin))
	dataLen++
	copy(bytes[dataLen:], eQuoteCoin)
	dataLen = dataLen + len(eQuoteCoin)

	eFeeCoin := []byte(d.FeeCcy)
	bytes[dataLen] = byte(len(eFeeCoin))
	dataLen++
	copy(bytes[dataLen:], eFeeCoin)
	dataLen = dataLen + len(eFeeCoin)

	return base64.StdEncoding.EncodeToString(bytes[:dataLen])
}

func (d *DealRecord) Decode(data string) bool {
	getSymbolTypeStr := func(symbolType byte) string {
		if symbolType == 1 {
			return "spot"
		} else if symbolType == 2 {
			return "swap"
		} else {
			return "null"
		}
	}
	getTradeDirStr := func(dir byte) string {
		if dir == 1 {
			return "b"
		} else if dir == 2 {
			return "s"
		} else {
			return "null"
		}
	}
	getDealTypeStr := func(dealType byte) string {
		if dealType == 1 {
			return "maker"
		} else if dealType == 2 {
			return "taker"
		} else {
			return "null"
		}
	}

	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}

	d.OrderId = strconv.FormatUint(binary.BigEndian.Uint64(bytes), 10)
	d.Amount = decimal.NewFromFloat(math.Float64frombits(binary.LittleEndian.Uint64(bytes[8:])))
	d.Price = decimal.NewFromFloat(math.Float64frombits(binary.LittleEndian.Uint64(bytes[16:])))
	d.Fee = decimal.NewFromFloat(math.Float64frombits(binary.LittleEndian.Uint64(bytes[24:])))
	d.SymbolType = getSymbolTypeStr(bytes[32])
	d.Dir = getTradeDirStr(bytes[33])
	d.DealType = getDealTypeStr(bytes[34])
	var pos byte = 35
	d.Exchange = string(bytes[pos+1 : pos+bytes[pos]+1])
	pos = pos + bytes[pos] + 1

	baseCoin := string(bytes[pos+1 : pos+bytes[pos]+1])
	pos = pos + bytes[pos] + 1

	quoteCoin := string(bytes[pos+1 : pos+bytes[pos]+1])
	pos = pos + bytes[pos] + 1

	d.FeeCcy = string(bytes[pos+1 : pos+bytes[pos]+1])

	prefix := ""
	suffix := ""
	if d.Exchange == "BINANCE" {
		if d.SymbolType == "swap" {
			prefix = "binancef/"
			suffix = ".td"
		} else {
			prefix = "binance/"
		}
	} else if d.Exchange == "OKX" {
		if d.SymbolType == "swap" {
			prefix = "okswap/"
			suffix = ".td"
		} else {
			prefix = "okspot/"
		}
	}
	d.Symbol = prefix + baseCoin + "." + quoteCoin + suffix
	return true
}

func (o *DealRecord) Parse() {
	o.IsMaker = o.DealType == "maker"
	o.DealTime = time.UnixMilli(o.DealTimeStampNano / 1e6)

	// 是否为u本位合约
	if o.SymbolType == "swap" {
		if strings.Contains(o.Symbol, "usdt") {
			o.IsUsdtSwap = true
		} else {
			o.IsUsdtSwap = false
		}
	}

	// 计算baseCcy/instid
	ss0 := strings.Split(o.Symbol, "/")
	if len(ss0) == 2 {
		ss1 := strings.Split(ss0[1], ".")
		o.BaseCcy = ss1[0]

		if o.Exchange == "BINANCE" {
			if o.SymbolType == "swap" {
				if ss1[1] == "usdt" {
					o.InstId = binance.CCyCttypeToInstId(ss1[0], "usdt_swap")
				} else if ss1[1] == "usd" {
					o.InstId = binance.CCyCttypeToInstId(ss1[0], "usd_swap")
				}
			} else if o.SymbolType == "spot" {
				o.InstId = binance.SpotTypeToInstId(ss1[0], ss1[1])
			}
		} else if o.Exchange == "OKX" {
			if o.SymbolType == "swap" {
				if ss1[1] == "usdt" {
					o.InstId = okexv5.CCyCttypeToInstId(ss1[0], "usdt_swap")
				} else if ss1[1] == "usd" {
					o.InstId = okexv5.CCyCttypeToInstId(ss1[0], "usd_swap")
				}
			} else if o.SymbolType == "spot" {
				o.InstId = okexv5.SpotTypeToInstId(ss1[0], ss1[1])
			}
		}
	}

	// 计算交易额
	if strings.Contains(o.Symbol, "usd.td") {
		ContractValue := decimal.NewFromInt(10)
		if o.Symbol == "btc.usd.td" {
			ContractValue = decimal.NewFromInt(100)
		}
		o.TotalValue = o.Amount.Mul(ContractValue)
	} else {
		o.TotalValue = o.Amount.Mul(o.Price)
	}
}

func (o DealRecord) ToDataVisualPoint() datavisual.Point {
	return datavisual.Point{
		Time:  o.DealTime,
		Value: o.Price.InexactFloat64(),
		Tag:   util.ValueIf(o.Dir == "s", datavisual.PointTag_Sell, datavisual.PointTag_Buy)}
}

type RespDealRecordInner struct {
	RespCommon
	DealRecord []DealRecord `json:"data"`
}

func (r *RespDealRecordInner) parse() {
	for i := range r.DealRecord {
		r.DealRecord[i].Parse()
	}
}

type RespDealRecord struct {
	Status int                 `json:"status"`
	Data   RespDealRecordInner `json:"data"`
}

type AccountInfo struct {
	Alias string `json:"alias"`
	Name  string `json:"name"`
}

type RespAccountInfo struct {
	Data []AccountInfo `json:"data"`
}

func (r *RespAccountInfo) GetAccountNameByAlias(alias string) (string, bool) {
	for _, ai := range r.Data {
		if ai.Alias == alias {
			return ai.Name, true
		}
	}

	return "", false
}
