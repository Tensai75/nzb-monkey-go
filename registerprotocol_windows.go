package main

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func registerProtocol() {

	var k registry.Key
	var err error
	prefix := "SOFTWARE\\Classes\\"
	urlScheme := "nzblnk"
	basePath := prefix + urlScheme
	permission := uint32(registry.QUERY_VALUE | registry.SET_VALUE)
	baseKey := registry.CURRENT_USER

	// create key
	k, _, err = registry.CreateKey(baseKey, basePath, permission)

	// set description
	err = k.SetStringValue("", appName+" app")
	err = k.SetStringValue("URL Protocol", "")

	// set icon
	k, _, err = registry.CreateKey(baseKey, basePath+"\\DefaultIcon", permission)
	err = k.SetStringValue("", appExec+",1")

	// create tree
	_, _, err = registry.CreateKey(baseKey, basePath+"\\shell", permission)
	_, _, err = registry.CreateKey(baseKey, basePath+"\\shell\\open", permission)
	k, _, err = registry.CreateKey(baseKey, basePath+"\\shell\\open\\command", permission)

	// set open command
	cmdString := fmt.Sprintf("\"%s\" \"%%1\"", appExec)
	err = k.SetExpandStringValue("", cmdString)

	if err != nil {
		Log.Error("Unable to register 'nzblnk' URL protocol: %s", err.Error())
		exit(1)
	} else {
		Log.Succ("URL protocol 'nzblnk' successfully registered to '%s'", appExec)
		exit(0)
	}

}
