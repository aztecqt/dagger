/*
- @Author: aztec
- @Date: 2024-02-27 15:59:31
- @Description:
- @
- @Copyright (c) 2024 by aztec All Rights Reserved.
*/
package twsapi

type IncommingMessage int

const (
	InCommingMessage_NotValid                             IncommingMessage = -1
	InCommingMessage_TickPrice                            IncommingMessage = 1
	InCommingMessage_TickSize                             IncommingMessage = 2
	InCommingMessage_OrderStatus                          IncommingMessage = 3
	InCommingMessage_Error                                IncommingMessage = 4
	InCommingMessage_OpenOrder                            IncommingMessage = 5
	InCommingMessage_AccountValue                         IncommingMessage = 6
	InCommingMessage_PortfolioValue                       IncommingMessage = 7
	InCommingMessage_AccountUpdateTime                    IncommingMessage = 8
	InCommingMessage_NextValidId                          IncommingMessage = 9
	InCommingMessage_ContractData                         IncommingMessage = 10
	InCommingMessage_ExecutionData                        IncommingMessage = 11
	InCommingMessage_MarketDepth                          IncommingMessage = 12
	InCommingMessage_MarketDepthL2                        IncommingMessage = 13
	InCommingMessage_NewsBulletins                        IncommingMessage = 14
	InCommingMessage_ManagedAccounts                      IncommingMessage = 15
	InCommingMessage_ReceiveFA                            IncommingMessage = 16
	InCommingMessage_HistoricalData                       IncommingMessage = 17
	InCommingMessage_BondContractData                     IncommingMessage = 18
	InCommingMessage_ScannerParameters                    IncommingMessage = 19
	InCommingMessage_ScannerData                          IncommingMessage = 20
	InCommingMessage_TickOptionComputation                IncommingMessage = 21
	InCommingMessage_TickGeneric                          IncommingMessage = 45
	InCommingMessage_Tickstring                           IncommingMessage = 46
	InCommingMessage_TickEFP                              IncommingMessage = 47 //TICK EFP 47
	InCommingMessage_CurrentTime                          IncommingMessage = 49
	InCommingMessage_RealTimeBars                         IncommingMessage = 50
	InCommingMessage_FundamentalData                      IncommingMessage = 51
	InCommingMessage_ContractDataEnd                      IncommingMessage = 52
	InCommingMessage_OpenOrderEnd                         IncommingMessage = 53
	InCommingMessage_AccountDownloadEnd                   IncommingMessage = 54
	InCommingMessage_ExecutionDataEnd                     IncommingMessage = 55
	InCommingMessage_DeltaNeutralValidation               IncommingMessage = 56
	InCommingMessage_TickSnapshotEnd                      IncommingMessage = 57
	InCommingMessage_MarketData                           IncommingMessage = 58
	InCommingMessage_CommissionsReport                    IncommingMessage = 59
	InCommingMessage_Position                             IncommingMessage = 61
	InCommingMessage_PositionEnd                          IncommingMessage = 62
	InCommingMessage_AccountSummary                       IncommingMessage = 63
	InCommingMessage_AccountSummaryEnd                    IncommingMessage = 64
	InCommingMessage_VerifyMessageApi                     IncommingMessage = 65
	InCommingMessage_VerifyCompleted                      IncommingMessage = 66
	InCommingMessage_DisplayGroupList                     IncommingMessage = 67
	InCommingMessage_DisplayGroupUpdated                  IncommingMessage = 68
	InCommingMessage_VerifyAndAuthMessageApi              IncommingMessage = 69
	InCommingMessage_VerifyAndAuthCompleted               IncommingMessage = 70
	InCommingMessage_PositionMulti                        IncommingMessage = 71
	InCommingMessage_PositionMultiEnd                     IncommingMessage = 72
	InCommingMessage_AccountUpdateMulti                   IncommingMessage = 73
	InCommingMessage_AccountUpdateMultiEnd                IncommingMessage = 74
	InCommingMessage_SecurityDefinitionOptionParameter    IncommingMessage = 75
	InCommingMessage_SecurityDefinitionOptionParameterEnd IncommingMessage = 76
	InCommingMessage_SoftDollarTier                       IncommingMessage = 77
	InCommingMessage_FamilyCodes                          IncommingMessage = 78
	InCommingMessage_SymbolSamples                        IncommingMessage = 79
	InCommingMessage_MktDepthExchanges                    IncommingMessage = 80
	InCommingMessage_TickReqParams                        IncommingMessage = 81
	InCommingMessage_SmartComponents                      IncommingMessage = 82
	InCommingMessage_NewsArticle                          IncommingMessage = 83
	InCommingMessage_TickNews                             IncommingMessage = 84
	InCommingMessage_NewsProviders                        IncommingMessage = 85
	InCommingMessage_HistoricalNews                       IncommingMessage = 86
	InCommingMessage_HistoricalNewsEnd                    IncommingMessage = 87
	InCommingMessage_HeadTimestamp                        IncommingMessage = 88
	InCommingMessage_HistogramData                        IncommingMessage = 89
	InCommingMessage_HistoricalDataUpdate                 IncommingMessage = 90
	InCommingMessage_RerouteMktDataReq                    IncommingMessage = 91
	InCommingMessage_RerouteMktDepthReq                   IncommingMessage = 92
	InCommingMessage_MarketRule                           IncommingMessage = 93
	InCommingMessage_PnL                                  IncommingMessage = 94
	InCommingMessage_PnLSingle                            IncommingMessage = 95
	InCommingMessage_HistoricalTick                       IncommingMessage = 96
	InCommingMessage_HistoricalTickBidAsk                 IncommingMessage = 97
	InCommingMessage_HistoricalTickLast                   IncommingMessage = 98
	InCommingMessage_TickByTick                           IncommingMessage = 99
	InCommingMessage_OrderBound                           IncommingMessage = 100
	InCommingMessage_CompletedOrder                       IncommingMessage = 101
	InCommingMessage_CompletedOrdersEnd                   IncommingMessage = 102
	InCommingMessage_ReplaceFAEnd                         IncommingMessage = 103
	InCommingMessage_WshMetaData                          IncommingMessage = 104
	InCommingMessage_WshEventData                         IncommingMessage = 105
	InCommingMessage_HistoricalSchedule                   IncommingMessage = 106
	InCommingMessage_UserInfo                             IncommingMessage = 107
)

type OutgoingMessage int

const (
	OutgoingMessage_RequestMarketData                           OutgoingMessage = 1
	OutgoingMessage_CancelMarketData                            OutgoingMessage = 2
	OutgoingMessage_PlaceOrder                                  OutgoingMessage = 3
	OutgoingMessage_CancelOrder                                 OutgoingMessage = 4
	OutgoingMessage_RequestOpenOrders                           OutgoingMessage = 5
	OutgoingMessage_RequestAccountData                          OutgoingMessage = 6
	OutgoingMessage_RequestExecutions                           OutgoingMessage = 7
	OutgoingMessage_RequestIds                                  OutgoingMessage = 8
	OutgoingMessage_RequestContractData                         OutgoingMessage = 9
	OutgoingMessage_RequestMarketDepth                          OutgoingMessage = 10
	OutgoingMessage_CancelMarketDepth                           OutgoingMessage = 11
	OutgoingMessage_RequestNewsBulletins                        OutgoingMessage = 12
	OutgoingMessage_CancelNewsBulletin                          OutgoingMessage = 13
	OutgoingMessage_ChangeServerLog                             OutgoingMessage = 14
	OutgoingMessage_RequestAutoOpenOrders                       OutgoingMessage = 15
	OutgoingMessage_RequestAllOpenOrders                        OutgoingMessage = 16
	OutgoingMessage_RequestManagedAccounts                      OutgoingMessage = 17
	OutgoingMessage_RequestFA                                   OutgoingMessage = 18
	OutgoingMessage_ReplaceFA                                   OutgoingMessage = 19
	OutgoingMessage_RequestHistoricalData                       OutgoingMessage = 20
	OutgoingMessage_ExerciseOptions                             OutgoingMessage = 21
	OutgoingMessage_RequestScannerSubscription                  OutgoingMessage = 22
	OutgoingMessage_CancelScannerSubscription                   OutgoingMessage = 23
	OutgoingMessage_RequestScannerParameters                    OutgoingMessage = 24
	OutgoingMessage_CancelHistoricalData                        OutgoingMessage = 25
	OutgoingMessage_RequestCurrentTime                          OutgoingMessage = 49
	OutgoingMessage_RequestRealTimeBars                         OutgoingMessage = 50
	OutgoingMessage_CancelRealTimeBars                          OutgoingMessage = 51
	OutgoingMessage_RequestFundamentalData                      OutgoingMessage = 52
	OutgoingMessage_CancelFundamentalData                       OutgoingMessage = 53
	OutgoingMessage_ReqCalcImpliedVolat                         OutgoingMessage = 54
	OutgoingMessage_ReqCalcOptionPrice                          OutgoingMessage = 55
	OutgoingMessage_CancelImpliedVolatility                     OutgoingMessage = 56
	OutgoingMessage_CancelOptionPrice                           OutgoingMessage = 57
	OutgoingMessage_RequestGlobalCancel                         OutgoingMessage = 58
	OutgoingMessage_RequestMarketDataType                       OutgoingMessage = 59
	OutgoingMessage_RequestPositions                            OutgoingMessage = 61
	OutgoingMessage_RequestAccountSummary                       OutgoingMessage = 62
	OutgoingMessage_CancelAccountSummary                        OutgoingMessage = 63
	OutgoingMessage_CancelPositions                             OutgoingMessage = 64
	OutgoingMessage_VerifyRequest                               OutgoingMessage = 65
	OutgoingMessage_VerifyMessage                               OutgoingMessage = 66
	OutgoingMessage_QueryDisplayGroups                          OutgoingMessage = 67
	OutgoingMessage_SubscribeToGroupEvents                      OutgoingMessage = 68
	OutgoingMessage_UpdateDisplayGroup                          OutgoingMessage = 69
	OutgoingMessage_UnsubscribeFromGroupEvents                  OutgoingMessage = 70
	OutgoingMessage_StartApi                                    OutgoingMessage = 71
	OutgoingMessage_VerifyAndAuthRequest                        OutgoingMessage = 72
	OutgoingMessage_VerifyAndAuthMessage                        OutgoingMessage = 73
	OutgoingMessage_RequestPositionsMulti                       OutgoingMessage = 74
	OutgoingMessage_CancelPositionsMulti                        OutgoingMessage = 75
	OutgoingMessage_RequestAccountUpdatesMulti                  OutgoingMessage = 76
	OutgoingMessage_CancelAccountUpdatesMulti                   OutgoingMessage = 77
	OutgoingMessage_RequestSecurityDefinitionOptionalParameters OutgoingMessage = 78
	OutgoingMessage_RequestSoftDollarTiers                      OutgoingMessage = 79
	OutgoingMessage_RequestFamilyCodes                          OutgoingMessage = 80
	OutgoingMessage_RequestMatchingSymbols                      OutgoingMessage = 81
	OutgoingMessage_RequestMktDepthExchanges                    OutgoingMessage = 82
	OutgoingMessage_RequestSmartComponents                      OutgoingMessage = 83
	OutgoingMessage_RequestNewsArticle                          OutgoingMessage = 84
	OutgoingMessage_RequestNewsProviders                        OutgoingMessage = 85
	OutgoingMessage_RequestHistoricalNews                       OutgoingMessage = 86
	OutgoingMessage_RequestHeadTimestamp                        OutgoingMessage = 87
	OutgoingMessage_RequestHistogramData                        OutgoingMessage = 88
	OutgoingMessage_CancelHistogramData                         OutgoingMessage = 89
	OutgoingMessage_CancelHeadTimestamp                         OutgoingMessage = 90
	OutgoingMessage_RequestMarketRule                           OutgoingMessage = 91
	OutgoingMessage_ReqPnL                                      OutgoingMessage = 92
	OutgoingMessage_CancelPnL                                   OutgoingMessage = 93
	OutgoingMessage_ReqPnLSingle                                OutgoingMessage = 94
	OutgoingMessage_CancelPnLSingle                             OutgoingMessage = 95
	OutgoingMessage_ReqHistoricalTicks                          OutgoingMessage = 96
	OutgoingMessage_ReqTickByTickData                           OutgoingMessage = 97
	OutgoingMessage_CancelTickByTickData                        OutgoingMessage = 98
	OutgoingMessage_ReqCompletedOrders                          OutgoingMessage = 99
	OutgoingMessage_ReqWshMetaData                              OutgoingMessage = 100
	OutgoingMessage_CancelWshMetaData                           OutgoingMessage = 101
	OutgoingMessage_ReqWshEventData                             OutgoingMessage = 102
	OutgoingMessage_CancelWshEventData                          OutgoingMessage = 103
	OutgoingMessage_ReqUserInfo                                 OutgoingMessage = 104
)
