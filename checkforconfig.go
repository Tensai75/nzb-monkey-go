package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/skratchdot/open-golang/open"
)

func checkForConfig() {

	if _, err := os.Stat(confPath); errors.Is(err, os.ErrNotExist) {

		fmt.Println()
		Log.Warn("Configuration file '%s' not found. Creating configuration file ...", confPath)
		defaultConfig := []byte(defaultConfig())
		if err := os.WriteFile(confPath, defaultConfig, 0644); err != nil {
			Log.Error("Error creating configuration file: %s", err.Error())
			exit(1)
		} else {
			Log.Succ("Configuration file '%s' successfully created. Please edit default values.", confPath)
			open.Run(confPath)
		}

		fmt.Println()
		Log.Info("Registering the 'nzblnk' URL protocol ...")

		registerProtocol()

	}
}
