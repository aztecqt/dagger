/*
- @Author: aztec
- @Date: 2024-02-22 17:50:27
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package binancebrowserapi

// rest错误码和错误消息
// {"code":"000000","msg/message":"xxx", "messageDetail":"xxxx"}
type ErrorMessage struct {
	Code          interface{} `json:"code"`
	Msg           string      `json:"msg"`
	Message       string      `json:"message"`
	MessageDetail string      `json:"messageDetail"`
	Success       bool        `json:"success"`
}

type PlaceMarginOrderReq struct {
	Symbol         string `json:"symbol"`
	Type           string `json:"type"` // LIMIT/...
	Side           string `json:"side"` // BUY/SELL
	Quantity       string `json:"quantity"`
	Price          string `json:"price"`
	TimeInForce    string `json:"timeInForce"`    // GTC
	SideEffectType string `json:"sideEffectType"` // NO_SIDE_EFFECT
}

type PlaceMarginOrderResp struct {
	ErrorMessage
	Data struct {
		Symbol        string `json:"symbol"`
		OrderId       string `json:"orderId"`
		ClientOrderId string `json:"clientOrderId"`
		Status        string `json:"status"`
	} `json:"data"`
}

type CancelMarginOrderReq struct {
	Symbols  []string `json:"symbols"`
	OrderIds []int64  `json:"orderIds"`
}

type CancelMarginOrderUnit struct {
	Symbol        string `json:"symbol"`
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Msg           string `json:"msg"`
	Status        string `json:"status"`
}

type CancelMarginOrderResp struct {
	ErrorMessage
	Data struct {
		Errors   []CancelMarginOrderUnit `json:"errors"`
		Corrects []CancelMarginOrderUnit `json:"corrects"`
	} `json:"data"`
}
