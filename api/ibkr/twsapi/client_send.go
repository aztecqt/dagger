/*
- @Author: aztec
- @Date: 2024-02-28
- @Description: 跟send有关的逻辑
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/

package twsapi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/aztecqt/dagger/api/ibkr/twsapi/twsmodel"
	"github.com/aztecqt/dagger/util"
)

func formatTime(t time.Time) string {
	return t.UTC().Format("20060102-15:04:05")
}

func (c *Client) sendWithPrefix(prefix string, printRawData bool, params ...interface{}) {
	buf := bytes.NewBuffer(c.sendBuffer)
	buf.Reset()

	// 写入前缀（大部分都没有前缀）
	if len(prefix) > 0 {
		buf.WriteString(prefix)
		binary.Write(buf, binary.BigEndian, byte(0))
	}

	c._sendParams(buf, printRawData, params...)
}

func (c *Client) send(printRawData bool, params ...interface{}) {
	buf := bytes.NewBuffer(c.sendBuffer)
	buf.Reset()
	c._sendParams(buf, printRawData, params...)
}

func (c *Client) _sendParams(buf *bytes.Buffer, printRawData bool, params ...interface{}) {
	if c.conn == nil {
		return
	}

	// 预留长度字段
	lenPos := buf.Len()
	binary.Write(buf, binary.BigEndian, int32(0))

	// 展开参数数组
	paramsDeployed := []interface{}{}
	deployParamList(params, &paramsDeployed)

	// 写入参数，处理max值、Inf值等
	for _, p := range paramsDeployed {
		if str, ok := p.(string); ok {
			buf.WriteString(str)
			binary.Write(buf, binary.BigEndian, byte(0))
		} else if b, ok := p.(bool); ok {
			buf.WriteString(util.ValueIf(b, "1", "0"))
			binary.Write(buf, binary.BigEndian, byte(0))
		} else if i, ok := p.(int); ok {
			if i == -1 {
				fmt.Print()
			}
			if i != math.MaxInt32 {
				buf.WriteString(fmt.Sprintf("%d", i))
			}
			binary.Write(buf, binary.BigEndian, byte(0))
		} else if f, ok := p.(float64); ok {
			if f == math.Inf(1) {
				buf.WriteString("Infinity")
			} else if f != math.MaxFloat64 {
				buf.WriteString(fmt.Sprintf("%v", f))
			}
			binary.Write(buf, binary.BigEndian, byte(0))
		} else {
			buf.WriteString(fmt.Sprintf("%v", p))
			binary.Write(buf, binary.BigEndian, byte(0))
		}
	}

	// 计算内容长度
	contentLen := int32(buf.Len() - lenPos - 4)

	// 插入长度字段
	temp := buf.Bytes()
	buf2 := bytes.NewBuffer(temp[lenPos:])
	buf2.Reset()
	binary.Write(buf2, binary.BigEndian, contentLen)

	// 发送
	c.conn.Write(buf.Bytes())

	if printRawData {
		fmt.Println(visualizeBuffer(buf))
	}
}

// 通用的同步调用过程。把tws的异步api，转换为同步调用过程
func syncResponse[T any](c *Client, tTimeOut *T, fnMsgProc func(m Message) *T) *T {
	// 等待订单快照，或者Error，作为下单的返回结果
	ch := make(chan *T)
	done := false
	hid := c.RegisterMessageHandler(func(m Message) {
		t := fnMsgProc(m)
		if t != nil {
			done = true
			ch <- t
		}
	})

	go func() {
		timeOutTicker := time.NewTicker(time.Second * 5)
		<-timeOutTicker.C
		if !done {
			ch <- tTimeOut
		}
	}()

	msg := <-ch
	ch = nil
	c.UnregisterMessageHandler(hid)
	return msg
}

// 订阅账户概要。每三分钟同步一次账户信息。期间有变化也不会立即更新
// group: 一般填ALL，或者填TWS中设置好的账户分组
// tag：逗号分隔，没有就不填。具体定义见官方文档
func (c *Client) ReqAccountSummary(group, tags string) (reqId int) {
	if !c.IsConnectOk() {
		return
	}

	ver := 1
	reqId = c.nextReqId()
	c.send(false, OutgoingMessage_RequestAccountSummary, ver, reqId, group, tags)
	return
}

// 取消账户概要
// reqid: 当时ReqAccountSummary时传入的reqId
func (c *Client) CancelAccountSummary(reqId int) {
	if !c.IsConnectOk() {
		return
	}

	ver := 1
	c.send(false, OutgoingMessage_CancelAccountSummary, ver, reqId)
}

// 订阅账户更新。有变化则更新，无变化3分钟更新一次
func (c *Client) ReqAccountUpdates(account string) {
	if !c.IsConnectOk() {
		return
	}

	ver := 2
	c.send(false, OutgoingMessage_RequestAccountData, ver, true, account)
}

// 取消订阅账户更新
func (c *Client) CancelAccountUpdates(account string) {
	if !c.IsConnectOk() {
		return
	}

	ver := 2
	c.send(false, OutgoingMessage_RequestAccountData, ver, false, account)
}

func (c *Client) ReqContractDetails(cont twsmodel.Contract) *ContractDetailResponse {
	if !c.IsConnectOk() {
		return &ContractDetailResponse{RespCode: RespCode_ConnectionError}
	}

	ver := 8
	reqId := c.nextReqId()
	c.send(
		false,
		OutgoingMessage_RequestContractData,
		ver,
		reqId,
		cont.ConId,
		cont.Symbol,
		cont.SecType,
		cont.LastTradeDateOrContractMonth,
		cont.Strike,
		cont.Right,
		cont.Multiplier,
		cont.Exchange,
		cont.PrimaryExch,
		cont.Currency,
		cont.LocalSymbol,
		cont.TradingClass,
		cont.IncludeExpired,
		cont.SecIdType,
		cont.SecId,
		cont.IssuerId)

	resp := ContractDetailResponse{}
	return syncResponse(c, &ContractDetailResponse{RespCode: RespCode_TimeOut}, func(m Message) *ContractDetailResponse {
		if m.MsgId == InCommingMessage_ContractData {
			if msg, ok := m.Msg.(*ContractDetailMsg); ok {
				resp.MatchedDetails = append(resp.MatchedDetails, *msg)
			}
		} else if m.MsgId == InCommingMessage_ContractDataEnd {
			return &resp
		} else if m.MsgId == InCommingMessage_Error {
			msg := m.Msg.(*ErrorMsg)
			if msg.RequestId == reqId {
				return &ContractDetailResponse{RespCode: RespCode_Ok, Err: msg}
			}
		}

		return nil
	})
}

// 查询市场规则（其实就是价格增量）
func (c *Client) ReqMarketRule(marketRuleId int) *MarketRuleResponse {
	if !c.IsConnectOk() {
		return nil
	}

	c.send(false, OutgoingMessage_RequestMarketRule, marketRuleId)

	return syncResponse(c, &MarketRuleResponse{RespCode: RespCode_TimeOut}, func(m Message) *MarketRuleResponse {
		if m.MsgId == InCommingMessage_MarketRule {
			msg := m.Msg.(*MarketRuleMsg)
			if msg.Id == marketRuleId {
				return &MarketRuleResponse{RespCode: RespCode_Ok, MarketRule: msg}
			}
		}

		return nil
	})
}

// 请求L1数据
// genericTickList: 逗号分隔的tick类型，可以不填，详见文档
// snapshot: true立即返回一个快照，false则以stream的方式持续返回
// regulatorySnaphsot：监管快照？这个要花钱，1次1美分
func (c *Client) ReqMarketData(cont twsmodel.Contract, genericTickList string, snapshot bool, regulatorySnaphsot bool) (reqId int, resp *MarketDataResponse) {
	if !c.IsConnectOk() {
		return -1, &MarketDataResponse{RespCode: RespCode_ConnectionError}
	}

	ver := 11
	reqId = c.nextReqId()

	c.send(
		false,
		OutgoingMessage_RequestMarketData,
		ver,
		reqId,
		cont.ToParamArray(),
		false, // 这里跳过了关于cont.DeltaNeutralContract的处理
		genericTickList,
		snapshot,
		regulatorySnaphsot,
		"", // 跳过mktDataOptions
	)

	resp = syncResponse(c, &MarketDataResponse{RespCode: RespCode_TimeOut}, func(m Message) *MarketDataResponse {
		if m.MsgId == InCommingMessage_MarketData {
			msg := m.Msg.(*MarketDataTypeMsg)
			if msg.RequestId == reqId {
				return &MarketDataResponse{RespCode: RespCode_Ok}
			}
		} else if m.MsgId == InCommingMessage_Error {
			msg := m.Msg.(*ErrorMsg)
			if msg.RequestId == reqId {
				return &MarketDataResponse{RespCode: RespCode_Ok, Err: msg}
			}
		}

		return nil
	})

	return
}

// 请求tick-by-tick数据
// tickType:Last/AllLast/BidAsk/MidPoint
// numberOfTicks实时数据哪来数量一说？不懂，填0
// ignoreSize: 如果忽略数量变化，则只在价格变化时才更新
func (c *Client) ReqTickByTick(cont twsmodel.Contract, tickType string, numberOfTicks int, ignoreSize bool) (reqId int) {
	if !c.IsConnectOk() {
		return -1
	}

	reqId = c.nextReqId()

	c.send(
		false,
		OutgoingMessage_ReqTickByTickData,
		reqId,
		cont.ToParamArray(),
		tickType,
		numberOfTicks,
		ignoreSize,
	)

	return
}

// 取消tick-by-tick请求
func (c *Client) CancelTickByTick(reqId int) {
	if !c.IsConnectOk() {
		return
	}

	c.send(false, OutgoingMessage_CancelTickByTickData, reqId)
}

// 请求历史数据
// durStr: 3600 S/3 D/2 W/1 M/2 Y
// barSize: 1 sec/5 secs/15 secs/30 secs/1 min/2 mins/3 mins/5 mins/15 mins/30 mins/1 hour/1 day
// whatToShow: TRADES/MIDPOINT/BID/ASK/BID_ASK/...
func (c *Client) ReqHistoricalData(
	cont twsmodel.Contract,
	endTime time.Time,
	durStr, barSize, whatToShow string,
	useRTH int,
	keepUpToDate bool) *HisotricalDataResponse {
	if !c.IsConnectOk() {
		return &HisotricalDataResponse{RespCode: RespCode_ConnectionError}
	}

	reqId := c.nextReqId()

	c.send(
		false,
		OutgoingMessage_RequestHistoricalData,
		reqId,
		cont.ToParamArray(),
		cont.IncludeExpired,
		formatTime(endTime),
		barSize,
		durStr,
		useRTH,
		whatToShow,
		1, // formatDate，只能填1
		keepUpToDate,
		"", // 跳过chartOptions
	)

	return syncResponse(c, &HisotricalDataResponse{RespCode: RespCode_TimeOut}, func(m Message) *HisotricalDataResponse {
		if m.MsgId == InCommingMessage_HistoricalData {
			msg := m.Msg.(*HistoricalDataMsg)
			if msg.RequestId == reqId {
				return &HisotricalDataResponse{RespCode: RespCode_Ok, HistoricalData: msg}
			}
		} else if m.MsgId == InCommingMessage_Error {
			err := m.Msg.(*ErrorMsg)
			if err.RequestId == reqId {
				return &HisotricalDataResponse{RespCode: RespCode_Ok, Err: err}
			}
		}

		return nil
	})
}

// 执行下单
func (c *Client) PlaceOrder(cont twsmodel.Contract, o twsmodel.Order) *OrderResponse {
	if !c.IsConnectOk() {
		return &OrderResponse{RespCode: RespCode_ConnectionError}
	}

	c.send(
		false,
		OutgoingMessage_PlaceOrder,
		o.OrderId,
		cont.ToParamArray(),
		cont.SecIdType, //
		cont.SecId,     // contract在placeorder里多两个字段
		o.ToParamArray(),
	)

	return syncResponse(c, &OrderResponse{RespCode: RespCode_TimeOut}, func(m Message) *OrderResponse {
		if m.MsgId == InCommingMessage_OrderStatus {
			os := m.Msg.(*OrderStatusMsg)
			if os.OrderId == o.OrderId {
				return &OrderResponse{RespCode: RespCode_Ok, OrderStatus: os}
			}
		} else if m.MsgId == InCommingMessage_Error {
			err := m.Msg.(*ErrorMsg)
			if err.RequestId == o.OrderId {
				return &OrderResponse{RespCode: RespCode_Ok, Err: err}
			}
		}

		return nil
	})
}

// 撤单
// manualOrderCancelTime: 手动撤单时间，暂时不清楚怎么用，传空代表立即撤单
func (c *Client) CancelOrder(orderId int, manualOrderCancelTime string) *OrderResponse {
	if !c.IsConnectOk() {
		return &OrderResponse{RespCode: RespCode_ConnectionError}
	}

	ver := 1
	c.send(false, OutgoingMessage_CancelOrder, ver, orderId, manualOrderCancelTime)

	return syncResponse(c, &OrderResponse{RespCode: RespCode_TimeOut}, func(m Message) *OrderResponse {
		if m.MsgId == InCommingMessage_OrderStatus {
			os := m.Msg.(*OrderStatusMsg)
			if os.OrderId == orderId && os.Status == twsmodel.OrderStatus_Cancelled {
				return &OrderResponse{RespCode: RespCode_Ok, OrderStatus: os}
			}
		} else if m.MsgId == InCommingMessage_Error {
			err := m.Msg.(*ErrorMsg)
			if err.RequestId == orderId && err.ErrorCode != 202 /*这个是订单撤销通知，不是错误*/ {
				return &OrderResponse{RespCode: RespCode_Ok, Err: err}
			}
		}

		return nil
	})
}

// 撤销所有订单
func (c *Client) CancelAllOrders() {
	if !c.IsConnectOk() {
		return
	}

	ver := 1
	c.send(false, OutgoingMessage_RequestGlobalCancel, ver)
}

// 查询活动订单
func (c *Client) ReqOpenOrders() {
	if !c.IsConnectOk() {
		return
	}

	ver := 1
	c.send(false, OutgoingMessage_RequestOpenOrders, ver)
}
