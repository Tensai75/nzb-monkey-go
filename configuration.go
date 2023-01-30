package main

import (
	"path/filepath"
	"sort"
	"strconv"

	"gopkg.in/ini.v1"
)

// configuration structure
type Configuration struct {
	General struct {
		Target            string `ini:"target"`
		Categorize        string `ini:"categorize"`
		Success_wait_time int    `ini:"success_wait_time"`
		Error_wait_time   int    `ini:"error_wait_time"`
	} `ini:"GENERAL"`
	Execute struct {
		Passtofile       bool   `ini:"passtofile"`
		Passtoclipboard  bool   `ini:"passtoclipboard"`
		Nzbsavepath      string `ini:"nzbsavepath"`
		Dontexecute      bool   `ini:"dontexecute"`
		Clean_up_enable  bool   `ini:"clean_up_enable"`
		Clean_up_max_age int    `ini:"clean_up_max_age"`
	} `ini:"EXECUTE"`
	Sabnzbd struct {
		Host               string `ini:"host"`
		Port               int    `ini:"port"`
		Ssl                bool   `ini:"ssl"`
		Nzbkey             string `ini:"nzbkey"`
		Basicauth_username string `ini:"basicauth_username"`
		Basicauth_password string `ini:"basicauth_password"`
		Basepath           string `ini:"basepath"`
		Category           string `ini:"category"`
		Addpaused          bool   `ini:"addpaused"`
	} `ini:"SABNZBD"`
	Nzbget struct {
		Host               string `ini:"host"`
		Port               int    `ini:"port"`
		Ssl                bool   `ini:"ssl"`
		Basicauth_username string `ini:"user"`
		Basicauth_password string `ini:"pass"`
		Basepath           string `ini:"basepath"`
		Category           string `ini:"category"`
		Addpaused          bool   `ini:"addpaused"`
	} `ini:"NZBGET"`
	Synologyds struct {
		Host               string `ini:"host"`
		Port               int    `ini:"port"`
		Ssl                bool   `ini:"ssl"`
		Username           string `ini:"user"`
		Password           string `ini:"pass"`
		Basicauth_username string
		Basicauth_password string
		Basepath           string `ini:"basepath"`
	} `ini:"SYNOLOGYDLS"`
	Nzbcheck struct {
		Skip_failed                  bool    `ini:"skip_failed"`
		Max_missing_segments_percent float64 `ini:"max_missing_segments_percent"`
		Max_missing_files            int     `ini:"max_missing_files"`
		Best_nzb                     bool    `ini:"best_nzb"`
	} `ini:"NZBCheck"`
	Categories    map[string]string `ini:"-"` // will hold the categories regex patterns
	Searchengines []string          `ini:"-"` // will hold the search engines
	Directsearch  struct {
		Host        string `ini:"host"`
		Port        int    `ini:"port"`
		SSL         bool   `ini:"ssl"`
		Username    string `ini:"username"`
		Password    string `ini:"password"`
		Connections int    `ini:"connections"`
		Hours       int    `ini:"hours"`
		Step        int    `ini:"step"`
		Scans       int    `ini:"scans"`
		Skip        bool   `ini:"skip"`
	} `ini:"DIRECTSEARCH"`
}

// global configuration variable
var (
	conf Configuration
)

func loadConfig() {

	conf = Configuration{}

	var configFile string

	if args.Config != "" {
		configFile = filepath.Clean(args.Config)
	} else {
		configFile = filepath.Join(appPath, confFile)
	}

	iniOption := ini.LoadOptions{
		IgnoreInlineComment: true,
	}
	cfg, err := ini.LoadSources(iniOption, configFile)
	if err != nil {
		Log.Error("Unable to load configuration file '%s': %s", configFile, err.Error())
		exit(1)
	}

	err = cfg.MapTo(&conf)
	if err != nil {
		Log.Error("Unable to parse configuration file: %s", err.Error())
		exit(1)
	}

	// load categories
	conf.Categories = make(map[string]string)
	if cfg.HasSection("CATEGORIZER") {
		for _, key := range cfg.Section("CATEGORIZER").Keys() {
			conf.Categories[key.Name()] = key.Value()
		}
	}

	// load searchengines
	searchengines := make(map[string]int)
	if cfg.HasSection("SEARCHENGINES") {
		for _, key := range cfg.Section("SEARCHENGINES").Keys() {
			if _, ok := searchEngines[key.Name()]; ok {
				value, err := strconv.Atoi(key.Value())
				if err != nil {
					Log.Error("Unknown value for searchengine '%s' in configuration file: %s", key.Name(), key.Value())
					exit(1)
				}
				// only load the available searchengines to be used
				if _, ok := searchEngines[key.Name()]; ok && value != 0 {
					searchengines[key.Name()] = value
				}
			} else {
				Log.Error("Unknown searchengine '%s' in configuration file", key.Name())
				exit(1)
			}
		}
		// sort the searchengines
		engines := make([]string, 0, len(searchengines))
		for engine := range searchengines {
			engines = append(engines, engine)
		}
		sort.SliceStable(engines, func(i, j int) bool {
			return searchengines[engines[i]] < searchengines[engines[j]]
		})
		// add the searchengines to the config
		conf.Searchengines = engines
	}
}
