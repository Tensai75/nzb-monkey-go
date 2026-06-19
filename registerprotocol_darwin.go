package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func registerProtocol() {

	// Build an .app bundle next to the binary so macOS Launch Services
	// can register the nzblnk:// URI scheme.
	appBundle := filepath.Join(appPath, "NZBMonkey.app")
	contentsDir := filepath.Join(appBundle, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	fmt.Println()
	Log.Info("Creating app bundle '%s' ...", appBundle)

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		Log.Error("Failed to create app bundle directory: %s", err.Error())
		exitWithError()
	}

	// Symlink (or copy) the binary into the bundle.
	binaryLink := filepath.Join(macosDir, "nzb-monkey-go")
	_ = os.Remove(binaryLink) // remove stale link if present
	if err := os.Symlink(appExec, binaryLink); err != nil {
		Log.Error("Failed to link binary into app bundle: %s", err.Error())
		exitWithError()
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key>
  <string>nzb-monkey-go</string>
  <key>CFBundleName</key>
  <string>NZBMonkey</string>
  <key>CFBundleExecutable</key>
  <string>nzb-monkey-go</string>
  <key>CFBundleURLTypes</key>
  <array>
    <dict>
      <key>CFBundleURLName</key>
      <string>NZBLNK Protocol</string>
      <key>CFBundleURLSchemes</key>
      <array>
        <string>nzblnk</string>
      </array>
    </dict>
  </array>
</dict>
</plist>
`)
	plistPath := filepath.Join(contentsDir, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		Log.Error("Failed to write Info.plist: %s", err.Error())
		exitWithError()
	}
	Log.Succ("App bundle created")

	// Register with macOS Launch Services.
	fmt.Println()
	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	Log.Info("Registering app bundle with Launch Services ...")
	cmd := exec.Command(lsregister, "-f", appBundle)
	if out, err := cmd.CombinedOutput(); err != nil {
		Log.Error("lsregister failed: %s\n%s", err.Error(), string(out))
		exitWithError()
	}

	Log.Succ("URL protocol 'nzblnk' successfully registered to '%s'", appExec)
	exit(0)
}

func exitWithError() {
	Log.Error("Unable to register 'nzblnk' URL protocol")
	exit(1)
}
