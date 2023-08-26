/*
 * @Author: aztec
 * @Date: 2022-10-27 09:08:45
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package etherscanapi

type EthBlockNumberResp struct {
	Result string `json:"result"`
}

type EthGetBlockByNumberResp struct {
	Result struct {
		Number       string                   `json:"number"`
		TimeStamp    string                   `json:"timestamp"`
		Transactions []map[string]interface{} `json:"transactions"`
	} `json:"result"`
}

type GetBalanceResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}
