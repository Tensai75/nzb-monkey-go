package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/fatih/color"
)

type Logger struct {
	Error func(string, ...interface{})
	Warn  func(string, ...interface{})
	Info  func(string, ...interface{})
	Succ  func(string, ...interface{})
}

// global error logger variables
var (
	logFile *os.File
	logger  *log.Logger
	Log     = Logger{
		Error: logError,
		Warn:  logWarn,
		Info:  logInfo,
		Succ:  logSuccess,
	}
)

// error logger function
func logError(logText string, vars ...interface{}) {
	logEntry("error", logText, vars...)
}

func logWarn(logText string, vars ...interface{}) {
	logEntry("warn", logText, vars...)
}

func logInfo(logText string, vars ...interface{}) {
	logEntry("info", logText, vars...)
}

func logSuccess(logText string, vars ...interface{}) {
	logEntry("success", logText, vars...)
}

func logEntry(logType string, logText string, vars ...interface{}) {

	// init logger if nil
	if args.Debug && logFile == nil {
		initLogger(logFilePath)
	}

	// log error
	if logType == "error" {
		if args.Debug {
			logger.Printf("ERROR: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		color.Set(color.FgRed)
		fmt.Printf("   ERROR:    %s\n", fmt.Sprintf(logText, vars...))
		color.Unset()
	}

	// log warn
	if logType == "warn" {
		if args.Debug {
			logger.Printf("WARNING: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		color.Set(color.FgYellow)
		fmt.Printf("   WARNING:  %s\n", fmt.Sprintf(logText, vars...))
		color.Unset()
	}

	// log info
	if logType == "info" {
		if args.Debug {
			logger.Printf("INFO: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		fmt.Printf("   %s\n", fmt.Sprintf(logText, vars...))
	}

	// log success
	if logType == "success" {
		if args.Debug {
			logger.Printf("SUCCESS: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		color.Set(color.FgGreen)
		fmt.Printf("   SUCCESS:  %s\n", fmt.Sprintf(logText, vars...))
		color.Unset()
	}

}

func initLogger(file string) {
	var err error
	if logFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
		logger = log.New(logFile, "", log.Ldate|log.Ltime)
		logger.Printf("INFO: %s %s started", appName, appVersion)
	} else {
		color.Set(color.FgRed)
		fmt.Printf("   ERROR: Fatal error while opening log file '%s': %s\n", file, err.Error())
		color.Unset()
		exit(1)
	}
}

func logClose() {
	// clean up
	if logFile != nil {
		logFile.Close()
	}
}
