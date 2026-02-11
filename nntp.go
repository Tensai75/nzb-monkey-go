package main

import (
	"strings"
	"time"

	"github.com/Tensai75/nntpPool"
)

var (
	pool    nntpPool.ConnectionPool
	maxConn uint32
)

func initNntpPool() error {
	var err error

	go func() {
		for {
			select {
			case v := <-nntpPool.LogChan:
				Log.Debug("NNTPPool%v", v)
			case w := <-nntpPool.WarnChan:
				warning := w.Error()
				if strings.Contains(warning, "502") {
					Log.Debug("NNTPPool%v", warning)
				} else {
					Log.Warn("NNTPPool%v", warning)
				}
			}
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			used, total := pool.Conns()
			Log.Debug("NNTPPool: %d of %d connections in use", used, total)
			if total > maxConn {
				maxConn = total
			}
		}
	}()

	pool, err = nntpPool.New(&nntpPool.Config{
		Name:                  "",
		Host:                  conf.Directsearch.Host,
		Port:                  uint32(conf.Directsearch.Port),
		SSL:                   conf.Directsearch.SSL,
		SkipSSLCheck:          true,
		User:                  conf.Directsearch.Username,
		Pass:                  conf.Directsearch.Password,
		ConnWaitTime:          time.Duration(10) * time.Second,
		MaxConns:              uint32(conf.Directsearch.Connections),
		IdleTimeout:           30 * time.Second,
		HealthCheck:           true,
		MaxConnErrors:         3,
		MaxTooManyConnsErrors: 0,
	}, 0)
	if err != nil {
		return err
	}
	return nil
}
