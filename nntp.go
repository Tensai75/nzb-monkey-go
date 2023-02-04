package main

import (
	"fmt"
	"strconv"

	"github.com/Tensai75/nntp"
)

var connectionGuard chan struct{}

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
	if err != nil || conn == nil {
		if conn == nil && len(connectionGuard) > 0 {
			// if no connection was established, empty the guard channel
			<-connectionGuard
		} else {
			// otherwise close the connection
			DisconnectNNTP(conn)
		}
		return nil, fmt.Errorf("Connection to usenet server failed: %v\n", err)
	}
	if err = conn.Authenticate(conf.Directsearch.Username, conf.Directsearch.Password); err != nil {
		DisconnectNNTP(conn)
		return nil, fmt.Errorf("Authentication with usenet server failed: %v\n", err)
	}
	return conn, nil
}

func DisconnectNNTP(conn *nntp.Conn) {
	if conn != nil {
		conn.Quit()
		if len(connectionGuard) > 0 {
			<-connectionGuard
		}
	}
	conn = nil
}
