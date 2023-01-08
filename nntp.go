package main

import (
	"fmt"
	"strconv"

	"github.com/Tensai75/nntp"
)

var connectionGuard chan struct{}

var connections = 0

func ConnectNNTP() (*nntp.Conn, error) {
	if connectionGuard == nil {
		connectionGuard = make(chan struct{}, conf.Directsearch.Connections)
	}
	connectionGuard <- struct{}{} // will block if guard channel is already filled
	var conn *nntp.Conn
	var err error
	if conf.Directsearch.SSL {
		conn, err = nntp.DialTLS("tcp", conf.Directsearch.Host+":"+strconv.Itoa(conf.Directsearch.Port), nil)

	} else {
		conn, err = nntp.Dial("tcp", conf.Directsearch.Host+":"+strconv.Itoa(conf.Directsearch.Port))
	}
	if err != nil {
		conn.Quit()
		return nil, fmt.Errorf("Connection to usenet server failed: %v\n", err)
	}
	if err := conn.Authenticate(conf.Directsearch.Username, conf.Directsearch.Password); err != nil {
		conn.Quit()
		return nil, fmt.Errorf("Authentication with usenet server failed: %v\n", err)

	}
	return conn, nil
}

func DisconnectNNTP(conn *nntp.Conn) {
	if conn != nil {
		conn.Quit()
		select {
		case <-connectionGuard:
			// go on
		default:
			// go on
		}
	}
	conn = nil
}
