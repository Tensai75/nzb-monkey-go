package main

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
)

// function to save the nzb file
func execute_push(nzb string, category string) error {

	fmt.Println()
	Log.Info("Saving the NZB file ...")

	var basepath string
	var path string
	var err error

	if filepath.IsAbs(conf.Execute.Nzbsavepath) {
		basepath = conf.Execute.Nzbsavepath
	} else {
		basepath = filepath.Join(homePath, conf.Execute.Nzbsavepath)
	}

	if conf.Execute.Category_folder && category != "" {
		path = filepath.Join(basepath, category)
	} else {
		path = basepath
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
		execute_cleanup(basepath)
	}

	// make full filename
	nzbFile := args.Title
	if conf.Execute.Passtofile && args.Password != "" {
		nzbFile += fmt.Sprintf("{{%s}}", args.Password)
	}

	// write file
	if path, err = writeFile(path, nzbFile, nzb, conf.Execute.SaveAsZip, args.Title); err != nil {
		return err
	} else {
		Log.Succ("The NZB file was saved as '%s'", path)
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

func writeFile(path string, fileName string, file string, compress bool, zipFileName string) (string, error) {
	if compress {
		path = filepath.Join(path, zipFileName+".zip")
		archive, err := os.Create(path)
		if err != nil {
			return "", err
		}
		defer archive.Close()
		zipWriter := zip.NewWriter(archive)
		defer zipWriter.Close()
		zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, flate.BestCompression)
		})
		zippedFile, err := zipWriter.Create(fileName + ".nzb")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(zippedFile, strings.NewReader(file)); err != nil {
			return "", err
		}
	} else {
		path = filepath.Join(path, fileName+".nzb")
		if err := os.WriteFile(path, []byte(file), os.ModePerm); err != nil {
			return "", err
		}
	}
	return path, nil
}

func execute_cleanup(path string) {
	Log.Info("Cleaning up nzb folder '%s'", path)
	if files, err := os.ReadDir(path); err == nil {
		delete_files(files, path, 0)
	}
}

func delete_files(files []fs.DirEntry, path string, level int) {
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		if info, err := file.Info(); err == nil {
			// if category folder is active, recursively also delete nzb files in level 1 subfolders
			if file.IsDir() && conf.Execute.Category_folder && level < 1 {
				if files, err := os.ReadDir(filePath); err == nil {
					delete_files(files, filePath, level+1)
				}
			} else {
				if info.Mode().IsRegular() && time.Since(info.ModTime()) > time.Hour*time.Duration(conf.Execute.CleanUpMaxAge*24) && filepath.Ext(file.Name()) == ".nzb" {
					Log.Info("Deleting file '%s'", filePath)
					if err := os.Remove(filePath); err != nil {
						Log.Warn("Error deleting file '%s' during cleanup: %v", filePath, err)
					}
				}
			}
		} else {
			Log.Warn("Error reading info for '%s' during cleanup: %v", filePath, err)
		}
	}
}
