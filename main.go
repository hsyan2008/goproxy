package main

import (
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
)

var sshClient *ssh.Client
var err error
var config tomlConfig
var pac struct {
	Prehosts      []string
	Hosts         map[string]int
	Preblockhosts []string
	Blockhosts    map[string]int
}

func main() {
	logger.SetLogGoID(true)

	if _, err = toml.DecodeFile("main.toml", &config); err != nil {
		logger.Warn("load config error", err)
		os.Exit(1)
	}

	// logger.Info(config)

	if _, err = toml.DecodeFile("pac.toml", &pac); err != nil {
		logger.Warn("load pac config error", err)
		os.Exit(1)
	}

	pac.Hosts = make(map[string]int)
	pac.Blockhosts = make(map[string]int)
	for _, v := range pac.Prehosts {
		pac.Hosts[v] = 1
	}
	for _, v := range pac.Preblockhosts {
		pac.Blockhosts[v] = 1
	}

	// logger.Info(pac)

	if config.Ssh.Enable && config.Ssh.Addr != "" {
		checkSsh()
		if sshClient == nil {
			logger.Warn("init ssh connection fail")
			os.Exit(1)
		}
		go func() {
			for {
				time.Sleep(config.Keep * time.Second)
				checkSsh()
			}
		}()
	}

	logger.Warnf("%#v", config)

	for _, v := range config.Service {
		if v.IsHttp {
			go startHttp(v)
		} else {
			go startSocket5(v)
		}
	}

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
	if overssh {
		conn, err = sshClient.Dial("tcp", addr)
		if err != nil {
			checkSsh()
			conn, err = sshClient.Dial("tcp", addr)
		}
	} else {
		conn, err = net.DialTimeout("tcp", addr, timeout*time.Second)
	}

	return conn, err
}

//检查是否在pac列表里
func checkPac(addr string) bool {
	if len(pac.Hosts) == 0 {
		return false
	}
	host := strings.Split(addr, ":")[0]
	hosts := strings.Split(host, ".")
	pos := 1
	for pos <= len(hosts) {
		tmp := hosts[len(hosts)-pos:]
		tmp1 := strings.Join(tmp, ".")
		if _, ok := pac.Hosts[tmp1]; ok {
			return true
		} else {
			pos++
		}
	}

	return false
}

//检查是否在黑名单
func checkBlock(addr string) bool {
	if len(pac.Blockhosts) == 0 {
		return false
	}
	host := strings.Split(addr, ":")[0]
	hosts := strings.Split(host, ".")
	pos := 1
	for pos <= len(hosts) {
		tmp := hosts[len(hosts)-pos:]
		tmp1 := strings.Join(tmp, ".")
		if _, ok := pac.Blockhosts[tmp1]; ok {
			return true
		} else {
			pos++
		}
	}

	return false
}

var mut = new(sync.Mutex)

func checkSsh() {
	mut.Lock()
	defer mut.Unlock()
	logger.Info("keepalive")
	if keepalive(sshClient) != nil {
		if sshClient != nil {
			_ = sshClient.Close()
		}
		logger.Info("start to connect ssh")
		sshClient, err = connectSsh(config.Ssh.Addr, config.Ssh.User, config.Ssh.Auth, config.Ssh.Timeout)
		if err != nil {
			logger.Warn("ssh connection fail:", err)
			// os.Exit(1)
		} else {
			logger.Info("ssh connection success")
		}
	}
}

type tomlConfig struct {
	Title   string `toml:"title"`
	Keep    time.Duration
	Timeout time.Duration
	Service map[string]Config
	Ssh     Ssh
}

type Config struct {
	Addr    string `toml:"addr"`
	Overssh bool   `toml:"overssh"`
	Overpac bool   `toml:"overpac"`
	IsHttp  bool   `toml:"ishttp"`
}

type Ssh struct {
	Addr    string `toml:"addr"`
	User    string `toml:"user"`
	Auth    string `toml:"auth"`
	Timeout time.Duration
	Enable  bool
}
