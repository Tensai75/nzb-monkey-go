package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func registerProtocol() {

	var err error
	var homeDir string
	var desktopCommand string
	var desktopFile string
	var desktopFilePath string
	var desktopFileContent string

	homeDir, err = os.UserHomeDir()
	if err != nil {
		Log.Error("Unable to determine user home directory: %s", err.Error())
		exitWithError()
	}

	desktopFilePath = filepath.Join(homeDir, ".local/share/applications")
	desktopFile = filepath.Join(desktopFilePath, "nzblnk.desktop")
	desktopCommand = fmt.Sprintf("%s '%%u'", appExec)
	desktopFileContent = fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=NZBlnk
Exec=%s
Path=%s
MimeType=x-scheme-handler/nzblnk;
NoDisplay=true
Terminal=true
`, desktopCommand, appPath)

	fmt.Println()
	Log.Info("Writing desktop file '%s' ... ", desktopFile)
	err = os.MkdirAll(desktopFilePath, os.ModePerm)
	if err != nil {
		Log.Error("Failed to create directory '%s': %s", desktopFilePath, err.Error())
		exitWithError()
	}
	err = os.WriteFile(desktopFile, []byte(desktopFileContent), 0644)
	if err != nil {
		Log.Error("Writing desktop file failed: %s", err.Error())
		exitWithError()
	}
	Log.Succ("Desktop file successfully written")

	fmt.Println()
	Log.Info("Adding nzblnk to mimeapps.list ... ")
	xdgPath, _ := exec.LookPath("xdg-mime")
	if xdgPath == "" {
		Log.Error("xdg-mime not found; cannot add nzblnk to mimeapps.list")
		exitWithError()
	}
	cmd := exec.Command(xdgPath, "default", "nzblnk.desktop", "x-scheme-handler/nzblnk")
	err = cmd.Run()
	if err != nil {
		Log.Error("Adding nzblnk to mimeapps.list failed: %s", err.Error())
		exitWithError()
	}
	Log.Succ("Nzblnk successfully added to mimeapps.list")

	Log.Succ("URL protocol 'nzblnk' successfully registered to '%s'", appExec)
	exit(0)
}

func exitWithError() {
	Log.Error("Unable to register 'nzblnk' URL protocol")
	exit(1)
}
