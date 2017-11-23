package main

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/hsyan2008/go-logger/logger"

	"golang.org/x/crypto/ssh"
)

type Ssh struct {
	Addr    string `toml:"addr"`
	User    string `toml:"user"`
	Auth    string `toml:"auth"`
	Phrase  string `toml:"phrase"`
	Timeout time.Duration
	Enable  bool
}

func connectSsh(sshConfig Ssh) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: sshConfig.User,
		Auth: []ssh.AuthMethod{
			getAuth(sshConfig),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshConfig.Timeout * time.Second,
	}

	return ssh.Dial("tcp", sshConfig.Addr, config)
}

func getAuth(sshConfig Ssh) ssh.AuthMethod {
	//是文件
	var key []byte
	var err error

	if _, err = os.Stat(sshConfig.Auth); err == nil {
		key, _ = ioutil.ReadFile(sshConfig.Auth)
	}

	//密码
	if len(key) == 0 {
		if len(sshConfig.Auth) < 50 {
			return ssh.Password(sshConfig.Auth)
		} else {
			key = []byte(sshConfig.Auth)
		}
	}

	var signer ssh.Signer
	if sshConfig.Phrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(sshConfig.Phrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		panic("err private key:" + err.Error())
	}
	return ssh.PublicKeys(signer)
}

func keepalive(s *ssh.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warn("keepalive error")
			err = errors.New("keepalive error")
		}
	}()
	if s == nil {
		return errors.New("ssh Client is nil")
	}

	sess, err := s.NewSession()
	if err != nil {
		logger.Warn("keepalive NewSession error")
		return err
	}
	defer func() {
		_ = sess.Close()
	}()
	if err = sess.Shell(); err != nil {
		logger.Warn("keepalive shell error")
		return err
	}
	err = sess.Wait()
	if err != nil {
		logger.Warn("keepalive wait", err)
	}

	return
}
