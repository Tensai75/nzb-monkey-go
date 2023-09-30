package main

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/Tensai75/nntp"
)

type safeConn struct {
	closed bool
	*nntp.Conn
}

var (
	initConnGuard   sync.Once
	connectionGuard chan struct{}
)

func ConnectNNTP() (*safeConn, error) {
	initConnGuard.Do(func() {
		connectionGuard = make(chan struct{}, conf.Directsearch.Connections)
	})
	connectionGuard <- struct{}{} // will block if guard channel is already filled
	var conn *nntp.Conn
	var err error
	if conf.Directsearch.SSL {
		conn, err = nntp.DialTLS("tcp", conf.Directsearch.Host+":"+strconv.Itoa(conf.Directsearch.Port), nil)
	} else {
		conn, err = nntp.Dial("tcp", conf.Directsearch.Host+":"+strconv.Itoa(conf.Directsearch.Port))
	}
	safeConn := safeConn{
		Conn: conn,
	}
	if err != nil {
		safeConn.Close()
		return nil, fmt.Errorf("Connection to usenet server failed: %v\r\n", err)
	}
	if err = safeConn.Authenticate(conf.Directsearch.Username, conf.Directsearch.Password); err != nil {
		safeConn.Close()
		return nil, fmt.Errorf("Authentication with usenet server failed: %v\r\n", err)
	}
	return &safeConn, nil
}

func (c *safeConn) Close() {
	mutex.Lock()
	defer mutex.Unlock()
	if !c.closed {
		if c.Conn != nil {
			c.Quit()
		}
		if len(connectionGuard) > 0 {
			<-connectionGuard
		}
		c.closed = true
	}
}
