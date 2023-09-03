package main

import (
	"path/filepath"
)

var logFileName = "logfile.txt"
var logFilePath = filepath.Join(appPath, logFileName)

var confFile = "config.txt"
var confPath string

func setConfPath() {

	if args.Config != "" {
		// use config file from arguments if provided
		confPath = filepath.Clean(args.Config)
	} else {
		confPath = filepath.Join(appPath, confFile)
	}

}
