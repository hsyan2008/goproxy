package main

import (
	"io"
	"net"
	"os"
	"runtime"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
)

var sshClient *ssh.Client
var err error
var config tomlConfig

func main() {
	logger.SetLogGoID(true)

	if _, err = toml.DecodeFile("main.toml", &config); err != nil {
		logger.Warn("load config error", err)
		os.Exit(1)
	}

	logger.Info(config)

	if config.Ssh.Enable && config.Ssh.Addr != "" {
		go func() {
			for {
				checkSsh()
				time.Sleep(config.Keep * time.Second)
			}
		}()
	}

	go startSocket5(config.Socket5)
	go startSocket5(config.Socket5Ssh)
	go startHttp(config.Http)
	go startHttp(config.HttpSsh)

	runtime.Goexit()
}

func copyNet(des, src net.Conn) {
	defer func() {
		_ = des.Close()
		_ = src.Close()
	}()
	_, _ = io.Copy(des, src)
}

var timeout time.Duration = 10

func dial(addr string, overssh bool) (conn net.Conn, err error) {
	if sshClient == nil || !overssh {
		logger.Warn("不通过ssh连接", addr)
		conn, err = net.DialTimeout("tcp", addr, timeout*time.Second)
	} else {
		logger.Debug("通过ssh连接", addr)
		conn, err = sshClient.Dial("tcp", addr)
		if err != nil {
			checkSsh()
			conn, err = sshClient.Dial("tcp", addr)
		}
	}

	return conn, err
}

func checkSsh() {
	logger.Info("keepalive")
	if keepalive(sshClient) != nil {
		logger.Info("start to connect ssh")
		sshClient, err = connectSsh(config.Ssh.Addr, config.Ssh.User, config.Ssh.Auth, config.Ssh.Timeout)
		if err != nil {
			logger.Warn("ssh connection fail:", err)
			os.Exit(1)
		} else {
			logger.Info("ssh connection success")
		}
	}
}

type tomlConfig struct {
	Title      string `toml:"title"`
	Keep       time.Duration
	Timeout    time.Duration
	Http       Config
	HttpSsh    Config `toml:"http_ssh"`
	Socket5    Config
	Socket5Ssh Config `toml:"socket5_ssh"`
	Ssh        Ssh
}

type Config struct {
	Addr    string
	Overssh bool
}

type Ssh struct {
	Addr    string `toml:"addr"`
	User    string `toml:"user"`
	Auth    string `toml:"auth"`
	Timeout time.Duration
	Enable  bool
}
