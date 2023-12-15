/*
 * @Author: aztec
 * @Date: 2023-05-08 09:04:53
 * @Description: key分配器
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */

package util

import "time"

type KeyAndAsignTime struct {
	key      string
	lastTime int64
}

type KeyGroup struct {
	keys          []*KeyAndAsignTime // key-上次使用时间ms
	interator     int                // 循环索引
	minIntervalMS int64              // 同一个key的最小使用间隔
}

func (k *KeyGroup) Init(minIntervalMS int64) {
	k.keys = make([]*KeyAndAsignTime, 0)
	k.minIntervalMS = minIntervalMS
}

func (k *KeyGroup) AddKey(key string) {
	for _, kaat := range k.keys {
		if kaat.key == key {
			return
		}
	}

	k.keys = append(k.keys, &KeyAndAsignTime{key: key, lastTime: 0})
}

func (k *KeyGroup) GetKey() string {
	for {
		for i := 0; i < len(k.keys); i++ {
			reali := k.interator % len(k.keys)
			kaat := k.keys[reali]
			nowMS := time.Now().UnixMilli()
			if nowMS-kaat.lastTime > k.minIntervalMS {
				kaat.lastTime = nowMS
				return kaat.key
			}

			k.interator++
		}

		time.Sleep(time.Millisecond * time.Duration(k.minIntervalMS/10))
	}
}
