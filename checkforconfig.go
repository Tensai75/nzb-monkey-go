package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/skratchdot/open-golang/open"
)

func checkForConfig() {

	var configFile = filepath.Join(appPath, confFile)

	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {

		fmt.Println()
		Log.Warn("Configuration file '%s' not found. Creating configuration file ...", confFile)
		defaultConfig := []byte(defaultConfig())
		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
			Log.Error("Error creating configuration file: %s", err.Error())
			exit(1)
		} else {
			Log.Succ("Configuration file '%s' successfully created. Please edit default values.", confFile)
			open.Run(configFile)
		}

		fmt.Println()
		Log.Info("Registering the 'nzblnk' URL protocol ...")

		registerProtocol()

	}
}
