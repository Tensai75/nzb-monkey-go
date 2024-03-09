package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
)

// function to save the nzb file
func execute_push(nzb string, category string) error {

	fmt.Println()
	Log.Info("Saving the NZB file ...")

	var path string
	var err error

	if filepath.IsAbs(conf.Execute.Nzbsavepath) {
		path = conf.Execute.Nzbsavepath
	} else {
		path = filepath.Join(homePath, conf.Execute.Nzbsavepath)
	}

	if conf.Execute.Category_folder && category != "" {
		path = filepath.Join(path, category)
	}

	if path, err = filepath.Abs(path); err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		Log.Warn("Path '%s' does not exist", path)
		for {
			fmt.Printf("   Creating path '%s'? (y/N): ", path)
			str := inputReader()
			if str == "y" || str == "Y" {
				fmt.Println()
				Log.Info("Creating path '%s' ...", path)
				if err := os.MkdirAll(path, os.ModePerm); err != nil {
					return fmt.Errorf("Unable to save NZB file. Error creating path '%s': %s", path, err.Error())
				}
				break

			} else if str == "N" {
				return fmt.Errorf("Unable to save NZB file, path '%s' does not exist", path)
			}
		}
	}

	// clean up files before writing new one
	if conf.Execute.CleanUpEnable {
		go execute_cleanup(path)
	}

	// make full filename
	nzbFile := args.Title
	if conf.Execute.Passtofile && args.Password != "" {
		nzbFile += fmt.Sprintf("{{%s}}", args.Password)
	}
	nzbFile += ".nzb"

	path = filepath.Join(path, nzbFile)

	// write file
	if err := os.WriteFile(path, []byte(nzb), os.ModePerm); err != nil {
		return err
	} else {
		Log.Succ("The NZB file was saved to '%s'", path)
	}

	// copy password to clipboard
	if conf.Execute.Passtoclipboard {
		fmt.Println()
		Log.Info("Copying password to clipboard ...")
		if err := clipboard.WriteAll(args.Password); err != nil {
			Log.Warn("Unable to copy password to clipboard: %s", err.Error())
		}
	}

	// execute default program
	if !conf.Execute.Dontexecute {
		fmt.Println()
		Log.Info("Executing default program for NZB files ...")
		if err := open.Run(path); err != nil {
			Log.Warn("Unable to execute default program: %s", err.Error())
		}
	}

	return nil

}

func execute_cleanup(path string) {
	files, err := os.ReadDir(path)
	if err == nil {
		for _, file := range files {
			filePath := filepath.Join(path, file.Name())
			info, err := file.Info()
			if err == nil {
				if info.Mode().IsRegular() {
					if time.Since(info.ModTime()) > time.Hour*time.Duration(conf.Execute.CleanUpMaxAge*24) && filepath.Ext(file.Name()) == ".nzb" {
						if err := os.Remove(filePath); err != nil {
							Log.Warn("Error deleting file '%s' during cleanup: %v", filePath, err)
						}
					}
				}
			} else {
				Log.Warn("Error reading info for file '%s' during cleanup: %v", filePath, err)
			}
		}
	}
}
