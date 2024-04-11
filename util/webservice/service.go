/*
- @Author: aztec
- @Date: 2024-03-28 17:53:00
- @Description: 包含此类来获取web服务能力
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package webservice

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aztecqt/dagger/util/logger"
)

type HttpHandler func(http.ResponseWriter, *http.Request)

const logPrefix = "web-service"

type Service struct {
	server    http.Server
	paths     map[string]HttpHandler
	EnableLog bool
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

func (s *Service) RegisterPath(path string, h HttpHandler) {
	s.paths[path] = h
}

// http.Handler
func (s *Service) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.EnableLog {
		logger.LogDebug(logPrefix, "ServeHttp: %s", req.URL.Path)
	}

	if res, ok := s.paths[req.URL.Path]; ok {
		res(w, req)
	} else {
		s.onUnknownHttpReq(w, req)
	}
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
