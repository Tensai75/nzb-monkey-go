package main

import (
	"os"
	"path/filepath"
)

var logFileName = "nzb-monkey-go.log"
var logFilePath = filepath.Join("/tmp", logFileName)

var confPath string

func setConfPath() {

	oldConfFile := filepath.Join(appPath, "config.txt")
	confDir := filepath.Join(homePath, ".config")
	confFile := filepath.Join(confDir, "nzb-monkey-go.conf")

	if args.Config != "" {
		// use config file from arguments if provided
		confPath = filepath.Clean(args.Config)
	} else {
		confPath = confFile

		// make sure the directory for the config file exists, otherwise the config file cannot be saved
		if err := os.MkdirAll(confDir, os.ModePerm); err != nil {
			Log.Error("Error creating config dir %s: %s", confDir, err.Error())
			exit(1)
		}
		if _, err := os.Stat(oldConfFile); err == nil {
			// if config.txt in the application path already exists move it to the new place
			Log.Info("Old config file detected. Moving config file from %s to %s", oldConfFile, confFile)
			if err := os.Rename(oldConfFile, confFile); err != nil {
				Log.Error("Error while moving config file from %s to %s: %s", oldConfFile, confFile, err.Error())
				exit(1)
			}
		}

	}

}
