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
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/aztecqt/dagger/util/logger"

	"github.com/aztecqt/dagger/util"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/influxdata/influxdb/client/v2"
)

var logPrefix = "influxdb-helper"

type ConnConfig struct {
	Addr              string `json:"addr"`
	UserName          string `json:"username"`
	Password          string `json:"password"`
	DataBase          string `json:"db"`
	RetentionPolicies string `json:"rp"`
}

func CreateConn(cfg ConnConfig) client.Client {
	if conn, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     cfg.Addr,
		Username: cfg.UserName,
		Password: cfg.Password,
	}); err == nil {
		if _, _, err := conn.Ping(time.Second * 10); err == nil {
			return conn
		}
		return nil
	} else {
		return nil
	}
}

func MakeQuery(fields []string, db, rp, mm string, tags map[string]string, t0, t1 time.Time, limit int) client.Query {
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
			sbCmd.WriteString(fmt.Sprintf(`AND "%s"='%s' `, k, v))
		}
	}

	if limit > 0 {
		sbCmd.WriteString(fmt.Sprintf("LIMIT %d", limit))
	}

	return client.NewQuery(sbCmd.String(), "", "")
}

// 写入数据
func Write(conn client.Client, db, rp, mm string, tm time.Time, tags, fields []interface{}) {
	go func() {
		defer util.DefaultRecover()

		if len(tags)%2 != 0 {
			logger.LogImportant(logPrefix, "len(tags) must be odd")
		}

		if len(fields)%2 != 0 {
			logger.LogImportant(logPrefix, "len(fields) must be odd")
		}

		var mTags map[string]string
		var mFields map[string]interface{}
		if len(tags) > 0 {
			mTags = make(map[string]string)
			for i := 0; i < len(tags); i += 2 {
				mTags[tags[i].(string)] = tags[i+1].(string)
			}
		}

		if len(fields) > 0 {
			mFields = make(map[string]interface{})
			for i := 0; i < len(fields); i += 2 {
				mFields[fields[i].(string)] = fields[i+1]
			}
		}

		batchPoint, err := client.NewBatchPoints(client.BatchPointsConfig{Database: db})
		if err != nil {
			logger.LogImportant(logPrefix, "influx creat batchPoint failed, err=%s", err.Error())
		}

		pt, err := client.NewPoint(mm, mTags, mFields, tm)
		if err != nil {
			logger.LogImportant(logPrefix, "influx creat point failed, err=%s", err.Error())
		}

		batchPoint.AddPoint(pt)

		err = conn.Write(batchPoint)
		if err != nil {
			logger.LogImportant(logPrefix, "influx write point failed, err=%s", err.Error())
		}
	}()
}

// 保存日志
func SaveLog(conn client.Client, str string) {
	host, herr := os.Hostname()
	user, uerr := user.Current()
	userName := ""
	if herr != nil {
		host = "unknown host"
	}

	if uerr != nil {
		userName = "unknown user"
	} else {
		userName = user.Username
	}

	Write(conn, "_internal", "monitor", "log", time.Now(), []interface{}{"host", host, "user", userName}, []interface{}{"content", str})
}

func DBExist(conn client.Client, dbName string) bool {
	respSrcDBs, err := conn.Query(client.NewQuery("show databases", "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DBExist: %s", err.Error())
		return false
	}

	if len(respSrcDBs.Err) > 0 {
		logger.LogImportant(logPrefix, "DBExist: %s", respSrcDBs.Err)
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
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "CreateDB: %s", err.Error())
		return false
	} else if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "CreateDB: %s", resp.Err)
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

	if len(respSrcDBs.Err) > 0 {
		logger.LogImportant(logPrefix, "GetRetentionPolicies: %s", respSrcDBs.Err)
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
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "CreateRetentionPolicy: %s", err.Error())
		return false
	} else if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "CreateRetentionPolicy: %s", resp.Err)
		return false
	} else {
		return true
	}
}

// 获取所有measurement名称，以及他们包含的tag
func GetMeasurementsWithTags(conn client.Client, dbName string) map[string]*hashset.Set {
	rst := make(map[string]*hashset.Set)

	// 先取全量
	q := fmt.Sprintf("show measurements on %s", dbName)
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "GetMeasurementsWithTags: %s", err.Error())
		return nil
	}
	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		for _, s := range resp.Results[0].Series {
			for _, v := range s.Values {
				measurement := v[0].(string)
				rst[measurement] = hashset.New()
			}
		}
	}

	q = fmt.Sprintf("show tag keys on %s", dbName)
	resp, err = conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "GetMeasurementsWithTags: %s", err.Error())
		return nil
	}

	if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "GetMeasurementsWithTags: %s", resp.Err)
		return nil
	}

	if len(resp.Results) > 0 && len(resp.Results[0].Series) > 0 {
		for _, s := range resp.Results[0].Series {
			for _, v := range s.Values {
				measurement := s.Name
				tag := v[0]
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
	t := time.Time{}
	q := fmt.Sprintf("select * from \"%s\".\"%s\".\"%s\" ORDER BY time DESC limit 1", dbName, rp, mn)
	resp, err := conn.Query(client.NewQuery(q, "", ""))
	if err != nil {
		logger.LogImportant(logPrefix, "GetLatestDataTime1: %s", err.Error())
		return t
	}

	if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "GetLatestDataTime1: %s", resp.Err)
		return t
	}

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
	resp, err := conn.Query(client.NewQueryWithRP(q, dbName, rp, ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", err.Error())
		return false
	} else if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", resp.Err)
		return false
	} else {
		return true
	}
}

// 删除measure中的包含某tag的数据
func DeleteMeasurementWithTag(conn client.Client, dbName, rp, mn string, tagKey, tagValue string) bool {
	q := fmt.Sprintf("delete from %s where \"%s\"='%s'", mn, tagKey, tagValue)
	resp, err := conn.Query(client.NewQueryWithRP(q, dbName, rp, ""))
	if err != nil {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", err.Error())
		return false
	} else if len(resp.Err) > 0 {
		logger.LogImportant(logPrefix, "DeleteMeasurement: %s", resp.Err)
		return false
	} else {
		return true
	}
}
