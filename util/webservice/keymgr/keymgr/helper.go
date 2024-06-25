/*
- @Author: aztec
- @Date: 2024-06-05 16:47:00
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package keymgr

import (
	"encoding/hex"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/crypto"
)

const deskey = "*#1rg4#*"
const redisKey = "secrets"

type SecretKey struct {
	ApiKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
}

func (s *SecretKey) Encrypt() {
	s.ApiKey = hex.EncodeToString(crypto.DesCBCEncrypt([]byte(s.ApiKey), []byte(deskey), []byte(deskey)))
	s.SecretKey = hex.EncodeToString(crypto.DesCBCEncrypt([]byte(s.SecretKey), []byte(deskey), []byte(deskey)))
}

func (s *SecretKey) Decrypt() {
	b0, _ := hex.DecodeString(s.ApiKey)
	b1, _ := hex.DecodeString(s.SecretKey)
	s.ApiKey = string(crypto.DesCBCDecrypter(b0, []byte(deskey), []byte(deskey)))
	s.SecretKey = string(crypto.DesCBCDecrypter(b1, []byte(deskey), []byte(deskey)))
}

func Exist(rc *util.RedisClient, tag string) bool {
	_, ok := rc.HGet(redisKey, tag)
	return ok
}

func Save(rc *util.RedisClient, tag string, s SecretKey) bool {
	s.Encrypt()
	if rc.HSet(redisKey, tag, util.Object2String(s)) {
		return true
	} else {
		return false
	}
}

func GetAll(rc *util.RedisClient) map[string]SecretKey {
	keys := map[string]SecretKey{}
	if v, ok := rc.HGetAll(redisKey); ok {
		for tag, v := range v {
			key := SecretKey{}
			if err := util.ObjectFromString(v, &key); err == nil {
				key.Decrypt()
				keys[tag] = key
			}
		}
	}
	return keys
}

func GetByTag(rc *util.RedisClient, tag string) (SecretKey, bool) {
	k := SecretKey{}
	if str, ok := rc.HGet(redisKey, tag); ok {
		if err := util.ObjectFromString(str, &k); err == nil {
			k.Decrypt()
			return k, true
		}
	}
	return k, false
}

func GetByApiKey(rc *util.RedisClient, apikey string) (SecretKey, bool, string) {
	keys := GetAll(rc)
	for tag, k := range keys {
		if k.ApiKey == apikey {
			return k, true, tag
		}
	}

	return SecretKey{}, false, ""
}

func DeleteByTag(rc *util.RedisClient, tag string) bool {
	return rc.HDel(redisKey, tag)
}
