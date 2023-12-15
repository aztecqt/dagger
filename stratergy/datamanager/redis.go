/*
 * @Author: aztec
 * @Date: 2022-12-27 18:04:32
 * @Description: datamanager的redis部分，主要用于存储策略的状态、同步在线参数等
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package datamanager

import (
	"encoding/json"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/cex/common"
	"github.com/aztecqt/dagger/stratergy"
	"github.com/aztecqt/dagger/util"
)

const (
	Key_Brief    = "brief"
	Key_Observer = "observers"
	Key_Params   = "params"
	Key_Detail   = "detail"
)

type Redis struct {
	s         stratergy.Stratergy
	name      string // 策略名
	class     string // 策略类型
	logPrefix string
	rc        *util.RedisClient
	exchanges []common.CEx
	chStop    chan int
	observer  stratergy.Observer
}

func InitRedis(
	s stratergy.Stratergy,
	redisAddr, redisPass string, redisDB int) (*Redis, *util.RedisClient) {
	r := new(Redis)
	if s.Params() != nil {
		s.Params().Version = -1
	}

	r.s = s
	r.name = s.Name()
	r.class = s.Class()
	r.logPrefix = "DataManager.Redis"
	r.chStop = make(chan int)

	// 建立redis连接
	r.rc = new(util.RedisClient)
	r.rc.Init(redisAddr, redisPass, redisDB)
	r.checkKeys()
	r.Start()
	return r, r.rc
}

func (m *Redis) Start() {
	go m.update()
}

func (m *Redis) Stop() {
	m.chStop <- 0
}

func (m *Redis) AddExchange(ex common.CEx) {
	m.exchanges = append(m.exchanges, ex)
}

func (m *Redis) update() {
	ticker := time.NewTicker(time.Millisecond * 500)
	lastWriteStatusTime := time.Unix(0, 0)

	for {
		select {
		case <-ticker.C:
			m.readObserverList()
			m.readParams()
			m.writeBrief()

			// 2 levels of write frequence
			deltaMs := time.Now().UnixMilli() - lastWriteStatusTime.UnixMilli()
			if m.observed() || (!m.observed() && deltaMs > 10000) {
				m.writeDetail()
				lastWriteStatusTime = time.Now()
			}

		case <-m.chStop:
			m.writeBrief()
			m.writeDetail()
		}
	}
}

func (m *Redis) observed() bool {
	return time.Now().UnixMilli()-m.observer.AliveTimeMs < 10*1000
}

func (m *Redis) readObserverList() {
	rst, ok := m.rc.HGet(Key_Observer, m.name)
	if ok && len(rst) > 0 {
		if json.Unmarshal([]byte(rst), &m.observer) != nil {
			logger.LogImportant(m.logPrefix, "unmarshal observer failed, str:%s", rst)
		}
	}
}

// 确保redis中各个key都存在
func (m *Redis) checkKeys() {
	if !m.rc.HExists(Key_Observer, m.name) {
		m.rc.HSet(Key_Observer, m.name, "")
	}

	if !m.rc.HExists(Key_Brief, m.name) {
		m.rc.HSet(Key_Brief, m.name, "")
	}

	if !m.rc.HExists(Key_Params, m.name) {
		b, _ := json.Marshal(m.s.Params())
		m.rc.HSet(Key_Params, m.name, string(b))
	}

	if m.readParams() {
		// 由于策略参数定义可能有更改
		// 成功读取参数内容后，再向数据库覆写一次，以便体现最新的参数结构
		p := m.s.Params()
		b, _ := json.Marshal(p)
		if !m.rc.HSet(Key_Params, m.name, b) {
			logger.LogPanic(m.logPrefix, "rewrite config error!")
		}
	} else if m.s.Params() != nil {
		logger.LogPanic(m.logPrefix, "load config error!")
	}
}

func (m *Redis) readParams() bool {
	if m.s.Params() == nil {
		return false
	}

	rst, ok := m.rc.HGet(Key_Params, m.name)
	if ok {
		if ok && len(rst) > 0 {
			p := stratergy.Param{}
			b := []byte(rst)
			if json.Unmarshal(b, &p) == nil {
				if p.Version > m.s.Params().Version || m.s.Params().Version == -1 /*first time*/ {
					bData, err := json.Marshal(p.Data)
					if err != nil {
						logger.LogPanic(m.logPrefix, "marshal error:%s", err.Error())
					} else {
						m.s.OnParamChanged(bData) // 回调给策略
					}
				}
				return true
			} else {
				return false
			}
		} else {
			// 未读取到数据也算是读取成功
			m.s.OnParamChanged(nil)
			return true
		}
	} else {
		logger.LogImportant(m.logPrefix, "read params failed")
		return false
	}
}

func (m *Redis) writeBrief() {
	brief := stratergy.Brief{}
	brief.ClassName = m.class
	brief.TimeStamp = time.Now().UnixMilli()

	b, err := json.Marshal(brief)
	if err == nil {
		if !m.rc.HSet(Key_Brief, m.name, string(b)) {
			logger.LogImportant(m.logPrefix, "write brief failed!")
		}
	} else {
		logger.LogImportant(m.logPrefix, "write brief failed with marshal error:%s", err.Error())
	}
}

func (m *Redis) writeDetail() {
	detail := stratergy.Detail{}
	detail.Class = m.class
	detail.Observed = m.observed()
	detail.TimeStamp = time.Now().UnixMilli()
	detail.Status = m.s.Status()
	if m.s.Params() != nil {
		detail.ParamVersion = m.s.Params().Version
	}

	detail.Exchanges = map[string]interface{}{}
	for _, ex := range m.exchanges {
		export := stratergy.CExExport{}
		export.From(ex)
		detail.Exchanges[ex.Name()] = export
	}

	b, err := json.Marshal(detail)
	if err == nil {
		if !m.rc.HSet(Key_Detail, m.name, string(b)) {
			logger.LogImportant(m.logPrefix, "write status failed!")
		}
	} else {
		logger.LogImportant(m.logPrefix, "write status failed with marshal error:%s", err.Error())
	}
}
