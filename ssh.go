package main

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/hsyan2008/go-logger/logger"

	"golang.org/x/crypto/ssh"
)

func connectSsh(addr, user, auth string, timeout time.Duration) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			getAuth(auth),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout * time.Second,
	}

	return ssh.Dial("tcp", addr, config)
}

func getAuth(auth string) ssh.AuthMethod {
	//是文件
	var key []byte

	if _, err := os.Stat(auth); err == nil {
		key, _ = ioutil.ReadFile(auth)
	}

	//密码
	if len(key) == 0 {
		if len(auth) < 50 {
			return ssh.Password(auth)
		} else {
			key = []byte(auth)
		}
	}

	signer, _ := ssh.ParsePrivateKey(key)
	return ssh.PublicKeys(signer)
}

func keepalive(s *ssh.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warn("keepalive error")
			err = errors.New("keepalive error")
		}
	}()
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
