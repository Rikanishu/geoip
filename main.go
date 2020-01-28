package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	conf, err := ParseConfig("./config.yaml")
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.SetLevel(logrus.Level(conf.LogLevel))
	go func() {
		//for pprof
		http.ListenAndServe("localhost:12951", nil)
	}()

	storage := NewCountryStorage(conf, BuildCountryDataSource(conf))
	//todo: add graceful shutdown
	server := NewServer(conf, storage)
	server.Run()
}
