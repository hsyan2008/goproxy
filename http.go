//支持http和https
//https://tools.ietf.org/html/draft-luotonen-web-proxy-tunneling-01

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/hsyan2008/go-logger/logger"
)

func startHttp(config Config) {
	if config.Addr == "" {
		logger.Warn("no addr")
		return
	}
	lister, err := net.Listen("tcp", config.Addr)
	if err != nil {
		logger.Warn("http/https listen error:", err)
	}
	logger.Info("start http/https listen ", config.Addr, "overssh", config.Overssh, "overpac", config.Overpac)

	for {
		conn, err := lister.Accept()
		if err != nil {
			continue
		}
		go handHttp(conn, config)
	}
}

func handHttp(conn net.Conn, config Config) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)

			buf := make([]byte, 1<<20)
			num := runtime.Stack(buf, false)
			logger.Warn(num, string(buf))

			_ = conn.Close()
		}
	}()

	r := bufio.NewReader(conn)

	req, err := http.ReadRequest(r)
	if err != nil {
		logger.Error(conn.RemoteAddr().String(), "http ReadRequest error:", err)
		return
	}

	req.Header.Del("Proxy-Connection")
	//否则远程连接不会关闭，导致Copy卡住
	req.Header.Set("Connection", "close")

	var logpre string

	if config.Overpac {
		if checkBlock(req.Host) {
			logger.Warnf("%s %s in block list", conn.RemoteAddr().String(), req.Host)
			_ = conn.Close()
			return
		}
		if checkPac(req.Host) {
			config.Overssh = true
		} else {
			config.Overssh = false
		}
	}

	if config.Overssh {
		logpre = fmt.Sprintf("%s %s 通过ssh %s", conn.RemoteAddr().Network(), conn.RemoteAddr().String(), req.Host)
	} else {
		logpre = fmt.Sprintf("%s %s 不通过ssh %s", conn.RemoteAddr().Network(), conn.RemoteAddr().String(), req.Host)
	}

	if req.Method == "CONNECT" {
		logger.Info(logpre, "正在建立连接...")
		con, err := dial(req.Host, config.Overssh)
		if err != nil {
			logger.Warn(err)
			return
		}
		logger.Info(logpre, "连接建立成功")

		_, _ = io.WriteString(conn, "HTTP/1.0 200 Connection Established\r\n\r\n")

		go copyNet(conn, con)
		go copyNet(con, conn)
	} else {
		hosts := strings.Split(req.Host, ":")
		if len(hosts) == 1 {
			hosts = append(hosts, "80")
		}
		logger.Info(logpre, "正在建立连接...")
		con, err := dial(strings.Join(hosts, ":"), config.Overssh)
		if err != nil {
			logger.Warn(req.Host, err)
			return
		}
		logger.Info(logpre, "连接建立成功")
		err = req.Write(con)
		if err != nil {
			logger.Warn(err)
			return
		}
		go copyNet(conn, con)
	}
}
