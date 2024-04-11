/*
- @Author: aztec
- @Date: 2024-03-01 12:12:14
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package twsmodel

import "fmt"

const (
	AccountSummaryTag_AccountType                    = "AccountType"
	AccountSummaryTag_NetLiquidation                 = "NetLiquidation"
	AccountSummaryTag_TotalCashValue                 = "TotalCashValue"
	AccountSummaryTag_SettledCash                    = "SettledCash"
	AccountSummaryTag_AccruedCash                    = "AccruedCash"
	AccountSummaryTag_BuyingPower                    = "BuyingPower"
	AccountSummaryTag_EquityWithLoanValue            = "EquityWithLoanValue"
	AccountSummaryTag_PreviousDayEquityWithLoanValue = "PreviousDayEquityWithLoanValue"
	AccountSummaryTag_GrossPositionValue             = "GrossPositionValue"
	AccountSummaryTag_ReqTEquity                     = "ReqTEquity"
	AccountSummaryTag_ReqTMargin                     = "ReqTMargin"
	AccountSummaryTag_SMA                            = "SMA"
	AccountSummaryTag_InitMarginReq                  = "InitMarginReq"
	AccountSummaryTag_MaintMarginReq                 = "MaintMarginReq"
	AccountSummaryTag_AvailableFunds                 = "AvailableFunds"
	AccountSummaryTag_ExcessLiquidity                = "ExcessLiquidity"
	AccountSummaryTag_Cushion                        = "Cushion"
	AccountSummaryTag_FullInitMarginReq              = "FullInitMarginReq"
	AccountSummaryTag_FullMaintMarginReq             = "FullMaintMarginReq"
	AccountSummaryTag_FullAvailableFunds             = "FullAvailableFunds"
	AccountSummaryTag_FullExcessLiquidity            = "FullExcessLiquidity"
	AccountSummaryTag_LookAheadNextChange            = "LookAheadNextChange"
	AccountSummaryTag_LookAheadInitMarginReq         = "LookAheadInitMarginReq"
	AccountSummaryTag_LookAheadMaintMarginReq        = "LookAheadMaintMarginReq"
	AccountSummaryTag_LookAheadAvailableFunds        = "LookAheadAvailableFunds"
	AccountSummaryTag_LookAheadExcessLiquidity       = "LookAheadExcessLiquidity"
	AccountSummaryTag_HighestSeverity                = "HighestSeverity"
	AccountSummaryTag_DayTradesRemaining             = "DayTradesRemaining"
	AccountSummaryTag_Leverage                       = "Leverage"
	AccountSummaryTag_LEDGER                         = "$LEDGER"
	AccountSummaryTag_LEDGER_CCY                     = "$LEDGER:%s"
	AccountSummaryTag_LEDGER_All                     = "$LEDGER:ALL"
)

func AccountSummaryTag_LedgerCurrency(ccy string) string {
	return fmt.Sprintf(AccountSummaryTag_LEDGER_CCY, ccy)
}
