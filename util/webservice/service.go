/*
- @Author: aztec
- @Date: 2024-03-28 17:53:00
- @Description: 包含此类来获取web服务能力
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package webservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aztecqt/dagger/util"
	"github.com/aztecqt/dagger/util/logger"
	"github.com/aztecqt/dagger/util/webservice/keymgr/keymgr"
)

type HttpHandler func(http.ResponseWriter, *http.Request)
type HttpHandlerWithResult func(http.ResponseWriter, *http.Request) bool

const logPrefix = "web-service"

type Service struct {
	sync.Mutex
	server            http.Server
	paths             map[string]HttpHandler
	unknownReqHandler HttpHandlerWithResult // 未明确注册的http请求。返回true表示得到了处理
	EnableLog         bool

	rcAuth     *util.RedisClient
	api2secret map[string]string
	api2tag    map[string]string
}

func (s *Service) Start(port int) {
	s.paths = make(map[string]HttpHandler)

	// 启动服务器
	s.server = http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s,
	}

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			logger.LogPanic(logPrefix, "ListenAndServe appear err: %s", err.Error())
		}
	}()

	logger.LogImportant(logPrefix, "started")
}

func (s *Service) EnableAuth(rcAuth *util.RedisClient) {
	s.rcAuth = rcAuth
	s.api2secret = map[string]string{}
	s.api2tag = map[string]string{}
}

// 对一个http请求进行鉴权
// 成功时，参数2为tag
// 失败是，参数2为错误原因
func (s *Service) Auth(w http.ResponseWriter, r *http.Request) (bool, string) {
	if s.rcAuth == nil {
		// 无需鉴权
		return true, ""
	}

	// 读取apikey
	keyHeaderKey := "API-KEY"
	apikey := r.Header.Get(keyHeaderKey)
	if len(apikey) == 0 {
		return false, fmt.Sprintf("need %s in header", keyHeaderKey)
	}

	// 验证时间戳
	q := r.URL.Query()
	if ts, ok := util.String2Int64(q.Get("timestamp")); ok {
		delta := time.Now().UnixMilli() - ts
		if delta < 0 {
			delta = -delta
		}

		if delta > 60*10*1000 {
			return false, "invalid timestamp"
		}
	}

	s.Lock()
	secretkey := ""
	tag := ""
	sk, skok := s.api2secret[apikey]
	_tag, tagok := s.api2tag[apikey]
	if skok && tagok {
		secretkey = sk
		tag = _tag
	} else {
		if v, ok, _tag := keymgr.GetByApiKey(s.rcAuth, apikey); ok {
			secretkey = v.SecretKey
			tag = _tag
			s.api2secret[apikey] = secretkey
			s.api2tag[apikey] = _tag
		}
	}
	s.Unlock()

	logger.LogDebug(logPrefix, "api user '%s' is authing, apikey=%s, url=%s", tag, apikey, r.URL)

	if len(secretkey) == 0 {
		return false, "unknown apikey"
	}

	// 构建payload：url参数中去掉signature，剩下的作为payload
	signature0 := q.Get("signature")
	q.Del("signature")
	payload := q.Encode()
	logger.LogDebug(logPrefix, "calculate sign, payload=%s, secretkey=****%s", payload, secretkey[len(secretkey)-4:])
	if signature1, err := HmacSHA256Sign(payload, secretkey); err == nil {
		logger.LogDebug(logPrefix, "my sig: %s, his sig: %s", signature1, signature0)
		if signature0 == signature1 {
			return true, tag
		} else {
			return false, "auth failed"
		}
	} else {
		logger.LogDebug(logPrefix, "sign failed")
		return false, "auth failed"
	}
}

func HmacSHA256Sign(message string, secretKey string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secretKey))
	_, err := mac.Write([]byte(message))
	if err != nil {
		return "", err
	}
	str := fmt.Sprintf("%x", (mac.Sum(nil)))
	return str, nil
}

func (s *Service) RegisterPath(path string, h HttpHandler) {
	s.paths[path] = h
}

func (s *Service) SetUnknownReqHandler(h HttpHandlerWithResult) {
	s.unknownReqHandler = h
}

// http.Handler
func (s *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.EnableLog {
		logger.LogDebug(logPrefix, "ServeHttp: %s", req.URL.Path)
	}

	if handler, ok := s.findHandler(req); ok {
		handler(w, req)
	} else {
		if s.unknownReqHandler != nil {
			if !s.unknownReqHandler(w, req) {
				s.onUnknownHttpReq(w, req)
			}
		} else {
			s.onUnknownHttpReq(w, req)
		}
	}
}

func (s *Service) findHandler(req *http.Request) (HttpHandler, bool) {
	if h, ok := s.paths[req.URL.Path]; ok {
		return h, true
	}

	ss := strings.Split(req.URL.Path, "/")
	for len(ss) >= 2 {
		path := strings.Join(ss, "/")
		if h, ok := s.paths[path]; ok {
			return h, true
		}
		ss = ss[:len(ss)-1]
	}

	return nil, false
}

func (s *Service) onUnknownHttpReq(_ http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	sb := strings.Builder{}

	if r.Method == "GET" {
		sb.WriteString("recv unpathed GET request\n")
		sb.WriteString("path: " + r.URL.Path + "\n")
		sb.WriteString(fmt.Sprintf("param count: %d", len(r.Form)))

		for k, v := range r.Header {
			sb.WriteString(fmt.Sprintf("head key: %s, value: %s\n", k, v))
		}

		for k, v := range r.Form {
			sb.WriteString(fmt.Sprintf("form key: %s, value: %s\n", k, v))
		}
	} else if r.Method == "POST" {
		sb.WriteString("recv unpathed POST request\n")
		sb.WriteString("path: " + r.URL.Path + "\n")

		for k, v := range r.PostForm {
			sb.WriteString(fmt.Sprintf("form key: %s, value: %s\n", k, v))
		}

		for k, v := range r.Header {
			sb.WriteString(fmt.Sprintf("head key: %s, value: %s\n", k, v))
		}

		body := make([]byte, 4096)
		n, err := r.Body.Read(body)
		if err != nil {
			sb.WriteString(fmt.Sprintf("body:\n %s", string(body[:n])))
		}
	}

	if s.EnableLog {
		logger.LogDebug(logPrefix, sb.String())
	}
}
