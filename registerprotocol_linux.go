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
	var terminalFailed bool

	if homeDir, err = os.UserHomeDir(); err == nil {

		desktopFilePath = filepath.Join(homeDir, ".local/share/applications")
		desktopFile = filepath.Join(desktopFilePath, "nzblnk.desktop")

		var terminals = map[string]string{
			"gnome-terminal": fmt.Sprintf("--hide-menubar --geometry=100x16 --working-directory=\"%s\" -e \"%s %%u\"", appPath, appExec),
			"konsole":        fmt.Sprintf("--p tabtitle=\"%s\" --hide-menubar --hide-tabbar --workdir=\"%s\" --nofork -e \"%s %%u\"", appName, appPath, appExec),
			"xfce4-terminal": fmt.Sprintf("--title=\"%s\" --hide-menubar --geometry=100x16 --working-directory=\"%s\" -e \"%s %%u\"", appName, appPath, appExec),
			"mate-terminal":  fmt.Sprintf("--title=\"%s\" --hide-menubar --geometry=100x16 --working-directory=\"%s\" -e \"%s %%u\"", appName, appPath, appExec),
			"lxterminal":     fmt.Sprintf("--title=\"%s\" --geometry=100x16 --working-directory=\"%s\" -e \"%s %%u\"", appName, appPath, appExec),
			"lxterm":         fmt.Sprintf("-geometry 100x16+200+200 -e \"%s %%u\"", appExec),
			"uxterm":         fmt.Sprintf("-geometry 100x16+200+200 -e \"%s %%u\"", appExec),
			"xterm":          fmt.Sprintf("-geometry 100x16+200+200 -e \"%s %%u\"", appExec),
		}

		Log.Info("Searching for terminal emulators ...")
		for name, command := range terminals {
			fmt.Println()
			Log.Info("Searching for '%s' ... ", name)
			if path, _ := exec.LookPath(name); path != "" {
				Log.Succ("Found! Using '%s' as terminal emulator.", name)
				desktopCommand = fmt.Sprintf("%s %s", path, command)
				break
			}
		}

		if desktopCommand == "" {
			terminalFailed = true
			fmt.Println()
			Log.Warn("No terminal emulator found!")
			Log.Info("Please enter the path to your favorite terminal emulator in:")
			Log.Info("%s", desktopFilePath)
			Log.Info("and change parameters if necessary.")
			desktopCommand = fmt.Sprintf("<Replace with path to terminal emulator> --title \"%s\" --hide-menubar --geometry=100x40 --working-directory=\"%s\" --command=\"%s %%u\"", appName, appPath, appExec)
		}

		desktopFileContent = fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=NZBlnk
Exec=%s
Path=%s
MimeType=x-scheme-handler/nzblnk;
NoDisplay=true
Terminal=false				
`, desktopCommand, appPath)

		fmt.Println()
		Log.Info("Writing desktop file '%s' ... ", desktopFile)
		os.MkdirAll(desktopFilePath, os.ModePerm)
		if err = os.WriteFile(desktopFile, []byte(desktopFileContent), 0644); err == nil {
			Log.Succ("Desktop file successfully written")
		} else {
			Log.Error("Writing desktop file failed: %s", err.Error())
			Log.Error("Unable to register 'nzblnk' URL protocol")
			exit(1)
		}

		fmt.Println()
		Log.Info("Adding nzblnk to mimeapps.list ... ")
		if path, err := exec.LookPath("xdg-mime"); path != "" {
			cmd := exec.Command(path, "default", "nzblnk.desktop", "x-scheme-handler/nzblnk")
			if err = cmd.Run(); err == nil {
				Log.Succ("Nzblnk successfully added to mimeapps.list")
			} else {
				Log.Error("Adding nzblnk to mimeapps.list failed: %s", err.Error())
				Log.Error("Unable to register 'nzblnk' URL protocol")
				exit(1)
			}
		} else {
			Log.Error("Adding nzblnk to mimeapps.list failed: %s", err.Error())
			Log.Error("Unable to register 'nzblnk' URL protocol")
			exit(1)
		}

	}

	if !terminalFailed {
		Log.Succ("URL protocol 'nzblnk' successfully registered to '%s'", appExec)
	} else {
		Log.Succ("URL protocol 'nzblnk' registered to '%s'", appExec)
		Log.Warn("Don't forget to change the nzblnk.desktop file or %s will not work!", appName)
	}
	exit(0)

}
