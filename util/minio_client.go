/*
- @Author: aztec
- @Date: 2023-12-05 15:45:14
- @Description: minio的简单封装
- @
- @Copyright (c) 2023 by aztec, All Rights Reserved.
*/
package util

import (
	"bytes"
	"io"

	"github.com/minio/minio-go/v6"
)

type MinioConfig struct {
	Endpoint string `json:"endpoint"`
	Key      string `json:"key"`
	Secret   string `json:"secret"`
}

type MinioClient struct {
	client *minio.Client
}

func NewMinioClient(cfg MinioConfig) *MinioClient {
	c := &MinioClient{}
	if clt, err := minio.New(cfg.Endpoint, cfg.Key, cfg.Secret, false); err == nil {
		c.client = clt
		return c
	} else {
		return nil
	}
}

func (c *MinioClient) Client() *minio.Client {
	return c.client
}

func (c *MinioClient) SaveBytes(bucketName, objName string, data []byte) (int64, error) {
	defer DefaultRecover()
	return c.client.PutObject(bucketName, objName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
}

func (c *MinioClient) LoadBytes(bucketName, objName string) ([]byte, error) {
	defer DefaultRecover()
	if obj, err := c.client.GetObject(bucketName, objName, minio.GetObjectOptions{}); err == nil || err == io.EOF {
		if stat, err := obj.Stat(); err == nil {
			b := make([]byte, stat.Size)
			obj.Read(b)
			return b, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}
