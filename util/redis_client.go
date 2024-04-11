/*
 * @Author: aztec
 * @Date: 2022-04-10 11:35:16
  - @LastEditors: Please set LastEditors
  - @LastEditTime: 2024-03-20 19:27:11
 * @FilePath: \dagger\util\redis_client.go
 * @Description: redis客户端。其实也就封装了个日志输出，跟直接用没啥区别
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
*/

package util

import (
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/go-redis/redis"
)

const redisLogPrefix = "redis"

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type RedisClient struct {
	Addr     string
	Password string
	DB       int
	c        *redis.Client
}

// 127.0.0.1:6666@password@dbindex
func (r *RedisClient) InitFromConfigStr(configstr string) {
	ss := strings.Split(configstr, "@")
	if len(ss) == 3 {
		r.Init(ss[0], ss[1], String2IntPanic(ss[2]), false)
	} else {
		logger.LogPanic(redisLogPrefix, "wrong config format: %s(need: 127.0.0.1:6666@password@dbindex)", configstr)
	}
}

func (r *RedisClient) InitFromConfig(cfg RedisConfig) {
	r.Init(cfg.Addr, cfg.Password, cfg.DB, false)
}

func (r *RedisClient) CloneWithDiffrentDbIndex(db int) *RedisClient {
	clone := &RedisClient{}
	clone.Init(r.Addr, r.Password, db, false)
	return clone
}

func (r *RedisClient) Init(addr, pass string, db int, readOnly bool) {
	r.Addr = addr
	r.Password = pass
	r.DB = db
	opt := redis.Options{
		Addr:         addr,
		Password:     pass,
		DB:           db,
		MaxRetries:   2,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}
	r.c = redis.NewClient(&opt)

	for i := 0; i < 10; i++ {
		_, err := r.c.Ping().Result()
		if err == nil {
			logger.LogImportant(redisLogPrefix, "connected to host:%s", opt.Addr)
			break
		} else if i == 9 {
			logger.LogPanic(redisLogPrefix, "failed to connect host:%s", opt.Addr)
		}
		time.Sleep(time.Millisecond * 100)
	}
}

func (r *RedisClient) LogCmdError(cmd interface{}, err error) {
	if !strings.Contains(err.Error(), "nil") { // 取不到值属于正常现象，不用报错
		logger.LogImportant(redisLogPrefix, "failed, cmd=%v, err=%s", cmd, err.Error())
	}
}

func (r *RedisClient) LogCmdResultNotOk(cmd interface{}, rst string) {
	logger.LogImportant(redisLogPrefix, `cmd result not "OK", cmd=%v, rst=%s`, cmd, rst)
}

func (r *RedisClient) Exists(key string) bool {
	cmd := r.c.Exists(key)
	len, err := cmd.Result()
	if err == nil {
		return len > 0
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) Del(key string) bool {
	cmd := r.c.Del(key)
	_, err := cmd.Result()
	if err == nil {
		return true
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) HExists(key, field string) bool {
	cmd := r.c.HExists(key, field)
	rst, err := cmd.Result()
	if err == nil {
		return rst
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) Set(key string, value interface{}) bool {
	cmd := r.c.Set(key, value, time.Second*30)
	_, err := cmd.Result()
	if err == nil {
		return true
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) Get(key string) (string, bool) {
	cmd := r.c.Get(key)
	rst, err := cmd.Result()
	if err == nil {
		return rst, true
	} else {
		r.LogCmdError(cmd, err)
		return "", false
	}
}

func (r *RedisClient) LLen(key string) (int64, bool) {
	cmd := r.c.LLen(key)
	len, err := cmd.Result()
	if err == nil {
		return len, true
	} else {
		r.LogCmdError(cmd, err)
		return 0, false
	}
}

func (r *RedisClient) LIndex(key string, index int64) (string, bool) {
	cmd := r.c.LIndex(key, index)
	val, err := cmd.Result()
	if err == nil {
		return val, true
	} else {
		r.LogCmdError(cmd, err)
		return "", false
	}
}

func (r *RedisClient) LRange(key string, start, stop int64) ([]string, bool) {
	cmd := r.c.LRange(key, start, stop)
	vals, err := cmd.Result()
	if err == nil {
		return vals, true
	} else {
		r.LogCmdError(cmd, err)
		return nil, false
	}
}

func (r *RedisClient) LPush(key string, values ...interface{}) (int64, bool) {
	cmd := r.c.LPush(key, values...)
	len, err := cmd.Result()
	if err == nil {
		return len, true
	} else {
		r.LogCmdError(cmd, err)
		return 0, false
	}
}

func (r *RedisClient) RPush(key string, values ...interface{}) (int64, bool) {
	cmd := r.c.RPush(key, values...)
	len, err := cmd.Result()
	if err == nil {
		return len, true
	} else {
		r.LogCmdError(cmd, err)
		return 0, false
	}
}

func (r *RedisClient) LSet(key string, index int64, value interface{}) bool {
	cmd := r.c.LSet(key, index, value)
	rst, err := cmd.Result()
	if err == nil {
		if rst == "OK" {
			return true
		} else {
			r.LogCmdResultNotOk(cmd, rst)
			return false
		}
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) LTrim(key string, start, stop int64) bool {
	cmd := r.c.LTrim(key, start, stop)
	rst, err := cmd.Result()
	if err == nil {
		if rst == "OK" {
			return true
		} else {
			r.LogCmdResultNotOk(cmd, rst)
			return false
		}
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) HLen(key string) int64 {
	cmd := r.c.HLen(key)
	l, err := cmd.Result()
	if err == nil {
		return l
	} else {
		return 0
	}
}

func (r *RedisClient) HGet(key, field string) (string, bool) {
	cmd := r.c.HGet(key, field)
	rst, err := cmd.Result()
	if err == nil {
		return rst, true
	} else {
		r.LogCmdError(cmd, err)
		return "", false
	}
}

func (r *RedisClient) HGetBytes(key, field string) ([]byte, bool) {
	cmd := r.c.HGet(key, field)
	rst, err := cmd.Bytes()
	if err == nil {
		return rst, true
	} else {
		r.LogCmdError(cmd, err)
		return nil, false
	}
}

func (r *RedisClient) HGetAll(key string) (map[string]string, bool) {
	cmd := r.c.HGetAll(key)
	rst, err := cmd.Result()
	if err == nil {
		return rst, true
	} else {
		r.LogCmdError(cmd, err)
		return nil, false
	}
}

func (r *RedisClient) HSet(key, field string, value interface{}) bool {
	cmd := r.c.HSet(key, field, value)
	_, err := cmd.Result()
	if err == nil {
		return true
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) HSetCheckNew(key, field string, value interface{}) (success, isnew bool) {
	cmd := r.c.HSet(key, field, value)
	rst, err := cmd.Result()
	if err == nil {
		return true, rst
	} else {
		r.LogCmdError(cmd, err)
		return false, false
	}
}

func (r *RedisClient) HMSet(key string, fields map[string]interface{}) bool {
	cmd := r.c.HMSet(key, fields)
	str, err := cmd.Result()
	if err == nil {
		return str == "OK"
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}

func (r *RedisClient) HDel(key string, fields ...string) bool {
	cmd := r.c.HDel(key, fields...)
	rst, err := cmd.Result()
	if err == nil {
		return rst > 0
	} else {
		r.LogCmdError(cmd, err)
		return false
	}
}
