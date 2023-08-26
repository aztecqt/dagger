/*
 * @Author: aztec
 * @Date: 2023-05-08
 * @Description: 我自己定义的基于web3的接口
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package myweb3

import "aztecqt/dagger/util"

const Network_EthMainnet string = "ether-main"
const Network_EthGoerli string = "ether-goerli"
const Network_EthSepolia string = "ether-sepolia"
const Network_Arbitrum string = "arbiturm"
const Network_Optimism string = "optimism"
const Network_Avalanche string = "avalanche"
const Network_ZkSyncEra string = "zksync-era"
const Network_ZkSyncEraTestnet string = "zksync-era-testnet"

type GetBalanceResponse struct {
	Message   string            `json:"message"`
	ResultRaw map[string]string `json:"result"`
}

func (r *GetBalanceResponse) Result() map[string]float64 {
	result := make(map[string]float64)
	for k, v := range r.ResultRaw {
		result[k] = util.String2Float64Panic(v)
	}
	return result
}

func (r *GetBalanceResponse) Balance(ccy string) (float64, bool) {
	if str, ok := r.ResultRaw[ccy]; ok {
		b, ok := util.String2Float64(str)
		return b, ok
	} else {
		return 0, false
	}
}
