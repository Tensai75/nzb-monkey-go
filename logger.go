package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	color "github.com/TwiN/go-color"
	"github.com/acarl005/stripansi"
)

type Logger struct {
	Error func(string, ...interface{})
	Warn  func(string, ...interface{})
	Info  func(string, ...interface{})
	Succ  func(string, ...interface{})
}

// global error logger variables
var (
	logFileName = "./logfile.txt"
	logFile     *os.File
	logger      *log.Logger
	Log         = Logger{
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
	if args.Debug == true && logFile == nil {
		initLogger(filepath.Join(appPath, logFileName))
	}

	// log error
	if logType == "error" {
		if args.Debug == true {
			logger.Printf("ERROR: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		fmt.Printf("   %sERROR:    %s%s\n", color.Red, fmt.Sprintf(logText, vars...), color.Reset)
	}

	// log warn
	if logType == "warn" {
		if args.Debug == true {
			logger.Printf("WARNING: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		fmt.Printf("   %sWARNING:  %s%s\n", color.Yellow, fmt.Sprintf(logText, vars...), color.Reset)
	}

	// log info
	if logType == "info" {
		if args.Debug == true {
			logger.Printf("INFO: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		fmt.Printf("   %s%s%s\n", color.Reset, fmt.Sprintf(logText, vars...), color.Reset)
	}

	// log success
	if logType == "success" {
		if args.Debug == true {
			logger.Printf("INFO: %s\n", stripansi.Strip(strings.Trim(fmt.Sprintf(logText, vars...), "\n")))
		}
		fmt.Printf("   %s%s%s\n", color.Green, fmt.Sprintf(logText, vars...), color.Reset)
	}

}

func initLogger(file string) {
	var err error
	if logFile, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666); err == nil {
		logger = log.New(logFile, "", log.Ldate|log.Ltime)
		logger.Printf("INFO: %s %s started", appName, appVersion)
	} else {
		fmt.Printf("   %sERROR: Fatal error while opening log file '%s': %s%s\n", color.Red, file, err.Error(), color.Reset)
		exit(1)
	}
}
