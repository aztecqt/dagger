/*
- @Author: aztec
- @Date: 2024-03-06 11:43:58
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

const (
	OrderStatus_PendingSubmit = "PendingSubmit"
	OrderStatus_PendingCancel = "PendingCancel"
	OrderStatus_PreSubmitted  = "PreSubmitted"
	OrderStatus_Submitted     = "Submitted"
	OrderStatus_ApiCancelled  = "ApiCancelled"
	OrderStatus_Cancelled     = "Cancelled"
	OrderStatus_Filled        = "Filled"
	OrderStatus_Inactive      = "Inactive"
)

type OrderState struct {
	Status               string  `json:"status"`               // Status 表示订单当前状态
	InitMarginBefore     string  `json:"initMarginBefore"`     // InitMarginBefore 表示账户当前初始保证金
	MaintMarginBefore    string  `json:"maintMarginBefore"`    // MaintMarginBefore 表示账户当前维持保证金
	EquityWithLoanBefore string  `json:"equityWithLoanBefore"` // EquityWithLoanBefore 表示账户当前贷款权益
	InitMarginChange     string  `json:"initMarginChange"`     // InitMarginChange 表示账户初始保证金的变化
	MaintMarginChange    string  `json:"maintMarginChange"`    // MaintMarginChange 表示账户维持保证金的变化
	EquityWithLoanChange string  `json:"equityWithLoanChange"` // EquityWithLoanChange 表示账户贷款权益的变化
	InitMarginAfter      string  `json:"initMarginAfter"`      // InitMarginAfter 表示订单对账户初始保证金的影响
	MaintMarginAfter     string  `json:"maintMarginAfter"`     // MaintMarginAfter 表示订单对账户维持保证金的影响
	EquityWithLoanAfter  string  `json:"equityWithLoanAfter"`  // EquityWithLoanAfter 表示订单对账户贷款权益的影响
	Commission           float64 `json:"commission"`           // Commission 表示生成的佣金
	MinCommission        float64 `json:"minCommission"`        // MinCommission 表示执行的最低佣金
	MaxCommission        float64 `json:"maxCommission"`        // MaxCommission 表示执行的最高佣金
	CommissionCurrency   string  `json:"commissionCurrency"`   // CommissionCurrency 表示生成的佣金货币
	WarningText          string  `json:"warningText"`          // WarningText 如果订单受保证，则提供描述性消息
	CompletedTime        string  `json:"completedTime"`        // CompletedTime 表示完成时间
	CompletedStatus      string  `json:"completedStatus"`      // CompletedStatus 表示完成状态
}
