/*
 * @Author: aztec
 * @Date: 2022-11-01 15:15:59
 * @Description:
 *
 * Copyright (c) 2022 by aztec, All Rights Reserved.
 */
package influxdb

import (
	"fmt"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/util"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/influxdata/influxdb/client/v2"
)

var logPrefix = "influxdb-helper"

type ConnConfig struct {
	Addr     string `json:"addr"`
	UserName string `json:"username"`
	Password string `json:"password"`
}

func MakeQuery(fields []string, db, rp, mm string, tags map[string]string, t0, t1 time.Time) client.Query {
	// 拼command
	sbCmd := strings.Builder{}
	sbCmd.WriteString("SELECT ")

	if len(fields) == 0 {
		sbCmd.WriteString("* ")
	} else {
		for i, v := range fields {
			sbCmd.WriteString(fmt.Sprintf(`"%s"`, v))
			if i != len(fields)-1 {
				sbCmd.WriteString(",")
			} else {
				sbCmd.WriteString(" ")
			}
		}
	}

	sbCmd.WriteString(fmt.Sprintf(`FROM "%s"."%s"."%s" `, db, rp, mm))
	sbCmd.WriteString(fmt.Sprintf(`WHERE time > '%s' AND time < '%s' `, t0.UTC().Format("2006-01-02 15:04:05"), t1.UTC().Format("2006-01-02 15:04:05")))

	if tags != nil {
		for k, v := range tags {
			sbCmd.WriteString(fmt.Sprintf(`AND "%s"='%s'`, k, v))
		}
	}

	return client.NewQuery(sbCmd.String(), "", "")
}

func DBExist(conn client.Client, dbName string) bool {
	respSrcDBs, err := conn.Query(client.NewQuery("show databases", "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DBExist: %s", err.Error())
		return false
	}

	if len(respSrcDBs.Results) > 0 {
		if len(respSrcDBs.Results[0].Series) > 0 {
			s := respSrcDBs.Results[0].Series[0]
			for _, v := range s.Values {
				name := v[0].(string)
				if name == dbName {
					return true
				}
			}
		}
	}

	return false
}

func CreateDB(conn client.Client, dbName string) bool {
	q := fmt.Sprintf("create database \"%s\"", dbName)
	_, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "CreateDB: %s", err.Error())
		return false
	} else {
		return true
	}
}

// 获取所有rp
func GetRetentionPolicies(conn client.Client, dbName string) []string {
	q := fmt.Sprintf("show retention policies on \"%s\"", dbName)
	respSrcDBs, err := conn.Query(client.NewQuery(q, "", ""))
	rps := make([]string, 0)
	if err != nil {
		logger.LogImportant(logPrefix, "GetRetentionPolicies: %s", err.Error())
		return rps
	}

	if len(respSrcDBs.Results) > 0 {
		if len(respSrcDBs.Results[0].Series) > 0 {
			s := respSrcDBs.Results[0].Series[0]
			for _, v := range s.Values {
				rp := v[0].(string)
				rps = append(rps, rp)
			}
		}
	}

	return rps
}

// 判断一个rp是否存在
func RetentionPolicyExist(conn client.Client, dbName, rp string) bool {
	rps := GetRetentionPolicies(conn, dbName)
	for _, v := range rps {
		if v == rp {
			return true
		}
	}

	return false
}

// 创建rp
func CreateRetentionPolicy(conn client.Client, dbName, rp string) bool {
	q := fmt.Sprintf("create retention policy \"%s\" on \"%s\" duration inf REPLICATION 1", rp, dbName)
	_, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "CreateRetentionPolicy: %s", err.Error())
		return false
	} else {
		return true
	}
}

// 获取所有measurement名称，以及他们包含的tag
func GetMeasurementsWithTags(conn client.Client, dbName string) map[string]*hashset.Set {
	q := fmt.Sprintf("show tag keys on %s", dbName)
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "GetMeasurementsWithTags1: %s", err.Error())
	}

	rst := make(map[string]*hashset.Set)
	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		for _, s := range resp.Results[0].Series {
			for _, v := range s.Values {
				measurement := s.Name
				tag := v[0]
				if _, ok := rst[measurement]; !ok {
					rst[measurement] = hashset.New()
				}
				rst[measurement].Add(tag)
			}
		}
	} else {
		logger.LogImportant(logPrefix, "no result")
		return nil
	}

	return rst
}

// 获取一个数据表中的最新一条数据的时间
func GetLatestDataTime(conn client.Client, dbName, rp, mn string) time.Time {
	q := fmt.Sprintf("select * from \"%s\".\"%s\".\"%s\" ORDER BY time DESC limit 1", dbName, rp, mn)
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "GetLatestDataTime1: %s", err.Error())
	}

	t := time.Time{}

	func() {
		defer func() {
			recover()
		}()
		str := resp.Results[0].Series[0].Values[0][0].(string)
		t, err = time.Parse(time.RFC3339, str)
		if err != nil {
			logger.LogImportant(logPrefix, "GetLatestDataTime2: %s", err.Error())
		}
	}()

	return t
}

// 将一次查询结果保存到数据库中
func SaveQueryResultToDB(conn client.Client, dbName, rp, mn string, resp *client.Response, tagKeys *hashset.Set) int {
	batchPoint, err := client.NewBatchPoints(client.BatchPointsConfig{
		Precision:       "ns",
		Database:        dbName,
		RetentionPolicy: rp,
	})

	if err != nil {
		logger.LogImportant(logPrefix, "SaveQueryResultToDB1: %s", err.Error())
		return 0
	}

	// 目前就支持1个series
	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		s := resp.Results[0].Series[0]
		for _, v := range s.Values {
			var t time.Time
			tags := make(map[string]string)
			fields := make(map[string]interface{})
			for col := 0; col < len(v); col++ {
				if col == 0 {
					t, _ = time.Parse(time.RFC3339, v[0].(string))
				} else {
					if v[col] != nil {
						if tagKeys.Contains(s.Columns[col]) {
							tags[s.Columns[col]] = v[col].(string)
						} else {
							if f, ok := util.String2Float64(fmt.Sprintf("%v", v[col])); ok {
								fields[s.Columns[col]] = f
							} else {
								fields[s.Columns[col]] = v[col]
							}
						}
					}
				}
			}
			pt, err := client.NewPoint(mn, tags, fields, t)
			if err == nil {
				batchPoint.AddPoint(pt)
			}
		}
	}

	err = conn.Write(batchPoint)
	if err != nil {
		logger.LogImportant(logPrefix, "SaveQueryResultToDB2: %s", err.Error())
		return 0
	} else {
		return len(batchPoint.Points())
	}
}

// 删除measure中的数据
func DeleteMeasurement(conn client.Client, dbName, rp, mn string) bool {
	q := fmt.Sprintf("delete from %s", mn)
	_, err := conn.Query(client.NewQueryWithRP(q, dbName, rp, ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", err.Error())
		return false
	} else {
		return true
	}
}

// 删除measure中的包含某tag的数据
func DeleteMeasurementWithTag(conn client.Client, dbName, rp, mn string, tagKey, tagValue string) bool {
	q := fmt.Sprintf("delete from %s where \"%s\"='%s'", mn, tagKey, tagValue)
	_, err := conn.Query(client.NewQueryWithRP(q, dbName, rp, ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", err.Error())
		return false
	} else {
		return true
	}
}
