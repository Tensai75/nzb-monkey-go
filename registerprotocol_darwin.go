package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func registerProtocol() {

	// Only works when running from inside the distributed NZBMonkey.app bundle.
	// Expected layout: NZBMonkey.app/Contents/MacOS/nzb-monkey-go
	macosDir := filepath.Dir(appExec)
	contentsDir := filepath.Dir(macosDir)
	appBundle := filepath.Dir(contentsDir)

	plistPath := filepath.Join(contentsDir, "Info.plist")
	if filepath.Base(macosDir) != "MacOS" {
		exitWithError()
		return
	}
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		exitWithError()
		return
	}

	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	cmd := exec.Command(lsregister, "-f", appBundle)
	if out, err := cmd.CombinedOutput(); err != nil {
		Log.Error("lsregister failed: %s\n%s", err.Error(), string(out))
		Log.Error("Unable to register 'nzblnk' URL protocol")
		exit(1)
	}

	Log.Succ("URL protocol 'nzblnk' successfully registered to '%s'", appExec)
	exit(0)
}

func exitWithError() {
	Log.Warn("'--register' requires the NZBMonkey.app bundle from the macOS release zip.")
	Log.Warn("Download, unzip, and place NZBMonkey.app in /Applications or ~/Applications,")
	Log.Warn("then run: NZBMonkey.app/Contents/MacOS/nzb-monkey-go --register")
	Log.Error("Unable to register 'nzblnk' URL protocol")
	exit(1)
}
