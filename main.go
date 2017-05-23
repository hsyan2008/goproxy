//支持http和https
//https://tools.ietf.org/html/draft-luotonen-web-proxy-tunneling-01

package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/hsyan2008/go-logger/logger"
)

func main() {
	lister, err := net.Listen("tcp", ":18888")
	if err != nil {
		logger.Warn("listen error:", err)
	}

	for {
		conn, err := lister.Accept()
		if err != nil {
			continue
		}
		go hand(conn)
	}
}

func hand(conn net.Conn) {

	r := bufio.NewReader(conn)

	req, _ := http.ReadRequest(r)
	logger.Warn(req.Host)

	req.Header.Del("Proxy-Connection")
	//否则远程连接不会关闭，导致Copy卡住
	req.Header.Set("Connection", "close")

	if req.Method == "CONNECT" {
		con, err := net.Dial("tcp", req.Host)
		if err != nil {
			logger.Warn(err)
			return
		}

		_, _ = io.WriteString(conn, "HTTP/1.0 200 Connection Established\r\n\r\n")

		go copyNet(conn, con)
		go copyNet(con, conn)
	} else {
		hosts := strings.Split(req.Host, ":")
		if len(hosts) == 1 {
			hosts = append(hosts, "80")
		}
		con, err := net.Dial("tcp", strings.Join(hosts, ":"))
		if err != nil {
			logger.Warn(err)
			return
		}
		err = req.Write(con)
		if err != nil {
			logger.Warn(err)
			return
		}
		go copyNet(conn, con)
	}
}

func copyNet(des, src net.Conn) {
	defer func() {
		_ = des.Close()
		_ = src.Close()
	}()
	_, _ = io.Copy(des, src)
}
