/*
- @Author: aztec
- @Date: 2024-02-27 17:07:04
- @Description: 跟recv有关的逻辑
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsapi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/aztecqt/dagger/util"
)

var LogMessage = true

// 接收并拼接消息
func (c *Client) doRecv(conn net.Conn) {
	pos := 0
	buf := make([]byte, 1024*32)
	for {
		if n, err := conn.Read(buf); err == nil || err == io.EOF {
			if n > 0 {
				if pos+n >= len(c.recvBuffer) {
					// 缓冲区爆了
					logError(logPrefix, "recvBuffer overflow, reconnect")
					c.Reconnect("recvBuffer overflow")
					break
				} else {
					copy(c.recvBuffer[pos:], buf[:n])
					pos += n
				}
			}
		} else {
			// 读取失败，基本上是网络断了
			logInfo("read from tcp failed, reconnect. err=%s", err.Error())
			c.Reconnect("read from tcp failed")
			break
		}

		// 看看能不能凑一个消息出来
		for pos > 4 {
			bufMsgSize := bytes.NewBuffer(c.recvBuffer[:4])
			msgSize := int32(0)
			if err := binary.Read(bufMsgSize, binary.BigEndian, &msgSize); err == nil {
				if pos >= int(msgSize)+4 {
					// 收到了一条完整的消息
					c.parseAndProcessMessage(c.recvBuffer[4 : msgSize+4])

					// 剩余的数据复制到缓冲头部
					if pos > int(msgSize)+4 {
						copy(c.recvBuffer, c.recvBuffer[msgSize+4:pos])
						pos -= int(msgSize) + 4
					} else {
						pos = 0
					}
				}
			}
		}
	}
}

func (c *Client) parseAndProcessMessage(msg []byte) {
	buf := bytes.NewBuffer(msg)

	if c.serverVersion == 0 {
		// 是connectAck
		if ver := readInt(buf); ver > 0 {
			if ver > 0 {
				c.serverVersion = ver
			}
		}

		if c.serverVersion == 0 {
			panic("read server version failed")
		} else {
			logInfo(logPrefix, "server version=%d", c.serverVersion)
			c.startApi()
		}
	} else {
		// 是incomingMessage
		if n := readInt(buf); n > 0 {
			msgId := IncommingMessage(n)
			switch msgId {
			case InCommingMessage_Error:
				msg := deserializeAndProcessMessage(msgId, &ErrorMsg{}, buf, c)

				// 1300: 套接端口已经被重设，该连接被丢弃。请重新连接到新的端口
				if msg.ErrorCode == 1300 {
					c.Reconnect(fmt.Sprintf("tws err: code=%d, msg=%s", msg.ErrorCode, msg.ErrorMessage))
				}
			case InCommingMessage_ManagedAccounts:
				msg := deserializeAndProcessMessage(msgId, &ManagedAccountsMsg{}, buf, c)
				c.accounts = msg.ManagedAccounts
			case InCommingMessage_NextValidId:
				msg := deserializeAndProcessMessage(msgId, &NextOrderIdMsg{}, buf, c)
				if c.nextOrderId == 0 {
					c.nextOrderId = msg.NextOrderId // 仅首次赋值，后面自累加
				}
			case InCommingMessage_AccountSummary:
				deserializeAndProcessMessage(msgId, &AccountSummaryMsg{}, buf, c)
			case InCommingMessage_AccountSummaryEnd:
				deserializeAndProcessMessage(msgId, &AccountSummaryEndMsg{}, buf, c)
			case InCommingMessage_AccountValue:
				deserializeAndProcessMessage(msgId, &AccountValueMsg{}, buf, c)
			case InCommingMessage_AccountDownloadEnd:
				deserializeAndProcessMessage(msgId, &AccountDownloadEndMsg{}, buf, c)
			case InCommingMessage_PortfolioValue:
				deserializeAndProcessMessage(msgId, &PortfolioValueMsg{}, buf, c)
			case InCommingMessage_AccountUpdateTime:
				deserializeAndProcessMessage(msgId, &AccountUpdateTimeMsg{}, buf, c)
			case InCommingMessage_ContractData:
				deserializeAndProcessMessage(msgId, &ContractDetailMsg{}, buf, c)
			case InCommingMessage_ContractDataEnd:
				deserializeAndProcessMessage(msgId, &ContractDetailEndMsg{}, buf, c)
			case InCommingMessage_Tickstring:
				deserializeAndProcessMessage(msgId, &TickStringMsg{}, buf, c)
			case InCommingMessage_TickGeneric:
				deserializeAndProcessMessage(msgId, &TickGenericMsg{}, buf, c)
			case InCommingMessage_TickPrice:
				deserializeAndProcessMessage(msgId, &TickPriceMsg{}, buf, c)
			case InCommingMessage_TickSize:
				deserializeAndProcessMessage(msgId, &TickSizeMsg{}, buf, c)
			case InCommingMessage_TickSnapshotEnd:
				deserializeAndProcessMessage(msgId, &TickSnapshotEndMsg{}, buf, c)
			case InCommingMessage_MarketData:
				deserializeAndProcessMessage(msgId, &MarketDataTypeMsg{}, buf, c)
			case InCommingMessage_TickReqParams:
				deserializeAndProcessMessage(msgId, &TickReqParamsMsg{}, buf, c)
			case InCommingMessage_TickByTick:
				deserializeAndProcessMessage(msgId, &TickByTickMsg{}, buf, c)
			case InCommingMessage_MarketRule:
				deserializeAndProcessMessage(msgId, &MarketRuleMsg{}, buf, c)
			case InCommingMessage_HistoricalData:
				deserializeAndProcessMessage(msgId, &HistoricalDataMsg{}, buf, c)
			case InCommingMessage_OrderStatus:
				deserializeAndProcessMessage(msgId, &OrderStatusMsg{}, buf, c)
			case InCommingMessage_OpenOrder:
				deserializeAndProcessMessage(msgId, &OpenOrdersMsg{}, buf, c)
			case InCommingMessage_OpenOrderEnd:
				deserializeAndProcessMessage(msgId, &OpenOrderEndMsg{}, buf, c)
			default:
				logInfo(logPrefix, "unprocessed msgId: %d, content: %s\n", msgId, visualizeBuffer(buf))
			}
		} else {
			panic("read msgid failed")
		}
	}
}

func deserializeAndProcessMessage[T deserializable](msgId IncommingMessage, msg T, buf *bytes.Buffer, c *Client) T {
	msg.deserialize(buf)
	c.onMessage(msgId, msg)
	if LogMessage {
		t := reflect.TypeOf(msg)
		logDebug(logPrefix, "recv: %v(%d)\n%s", t, msgId, util.Object2String(msg))
	}
	return msg
}
