package main

import (
	"time"

	"github.com/Tensai75/nntpPool"
)

var (
	pool nntpPool.ConnectionPool
)

func initNntpPool() error {
	var err error

	go func() {
		for {
			select {
			case v := <-nntpPool.LogChan:
				Log.Info("NNTPPool%v\n", v)
			case w := <-nntpPool.WarnChan:
				Log.Warn("NNTPPool%v\n", w.Error())
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
		MaxConnErrors:         3,
		MaxTooManyConnsErrors: 3,
	}, 0)
	if err != nil {
		return err
	}
	return nil
}
