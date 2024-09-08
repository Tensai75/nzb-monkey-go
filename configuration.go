package main

import (
	"sort"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type General struct {
	Target            string `ini:"target"`
	Targets           []string
	Categorize        string `ini:"categorize"`
	Success_wait_time int    `ini:"success_wait_time"`
	Error_wait_time   int    `ini:"error_wait_time"`
	Debug             bool   `ini:"debug"`
}

type Execute struct {
	Passtofile      bool   `ini:"passtofile"`
	Passtoclipboard bool   `ini:"passtoclipboard"`
	Nzbsavepath     string `ini:"nzbsavepath"`
	Category_folder bool   `ini:"category_folder"`
	Dontexecute     bool   `ini:"dontexecute"`
	SaveAsZip       bool   `ini:"save_as_zip"`
	CleanUpEnable   bool   `ini:"clean_up_enable"`
	CleanUpMaxAge   int    `ini:"clean_up_max_age"`
}

type SABnzbd struct {
	Host              string `ini:"host"`
	Port              int    `ini:"port"`
	Ssl               bool   `ini:"ssl"`
	SkipCheck         bool   `ini:"skip_check"`
	Nzbkey            string `ini:"nzbkey"`
	BasicauthUsername string `ini:"basicauth_username"`
	BasicauthPassword string `ini:"basicauth_password"`
	Basepath          string `ini:"basepath"`
	Category          string `ini:"category"`
	Addpaused         bool   `ini:"addpaused"`
	Compression       string `ini:"compression"`
}

type NZBGet struct {
	Host              string `ini:"host"`
	Port              int    `ini:"port"`
	Ssl               bool   `ini:"ssl"`
	SkipCheck         bool   `ini:"skip_check"`
	BasicauthUsername string `ini:"user"`
	BasicauthPassword string `ini:"pass"`
	Basepath          string `ini:"basepath"`
	Category          string `ini:"category"`
	Addpaused         bool   `ini:"addpaused"`
}

type SynologyDS struct {
	Host              string `ini:"host"`
	Port              int    `ini:"port"`
	Ssl               bool   `ini:"ssl"`
	SkipCheck         bool   `ini:"skip_check"`
	Username          string `ini:"user"`
	Password          string `ini:"pass"`
	BasicauthUsername string
	BasicauthPassword string
	Basepath          string `ini:"basepath"`
}

type NZBcheck struct {
	SkipFailed                bool    `ini:"skip_failed"`
	MaxMissingSegmentsPercent float64 `ini:"max_missing_segments_percent"`
	MaxMissingFiles           int     `ini:"max_missing_files"`
	BestNZB                   bool    `ini:"best_nzb"`
}

type CategorySettings struct {
	name  string
	regex string
}

type DirectSearch struct {
	Host           string `ini:"host"`
	Port           int    `ini:"port"`
	SSL            bool   `ini:"ssl"`
	Username       string `ini:"username"`
	Password       string `ini:"password"`
	Connections    int    `ini:"connections"`
	Hours          int    `ini:"hours"`
	ForwardHours   int    `ini:"forward_hours"`
	Step           int    `ini:"step"`
	Scans          int    `ini:"scans"`
	Skip           bool   `ini:"skip"`
	FirstGroupOnly bool   `ini:"first_group_only"`
}

// configuration structure
type Configuration struct {
	General       General            `ini:"GENERAL"`
	Execute       Execute            `ini:"EXECUTE"`
	Sabnzbd       SABnzbd            `ini:"SABNZBD"`
	Nzbget        NZBGet             `ini:"NZBGET"`
	Synologyds    SynologyDS         `ini:"SYNOLOGYDLS"`
	Nzbcheck      NZBcheck           `ini:"NZBCheck"`
	Categories    []CategorySettings `ini:"-"` // will hold the categories regex patterns
	Searchengines []string           `ini:"-"` // will hold the search engines
	Directsearch  DirectSearch       `ini:"DIRECTSEARCH"`
}

// global configuration variable
var (
	conf Configuration
)

func loadConfig() {

	conf = Configuration{
		Directsearch: DirectSearch{
			Connections:  20,
			Hours:        12,
			ForwardHours: 12,
			Step:         20000,
			Scans:        50,
			Skip:         true,
		},
	}

	iniOption := ini.LoadOptions{
		IgnoreInlineComment: true,
	}
	cfg, err := ini.LoadSources(iniOption, confPath)
	if err != nil {
		Log.Error("Unable to load configuration file '%s': %s", confPath, err.Error())
		exit(1)
	}

	err = cfg.MapTo(&conf)
	if err != nil {
		Log.Error("Unable to parse configuration file: %s", err.Error())
		exit(1)
	}

	// load categories
	if cfg.HasSection("CATEGORIZER") {
		for _, key := range cfg.Section("CATEGORIZER").Keys() {
			conf.Categories = append(conf.Categories, struct {
				name  string
				regex string
			}{name: key.Name(), regex: key.Value()})
		}
	}

	// load searchengines
	searchengines := make(map[string]int)
	if cfg.HasSection("SEARCHENGINES") {
		for _, key := range cfg.Section("SEARCHENGINES").Keys() {
			if _, ok := searchEngines[key.Name()]; ok {
				value, err := strconv.Atoi(key.Value())
				if err != nil {
					Log.Warn("Unknown value for searchengine '%s' in configuration file: %s", key.Name(), key.Value())
				} else {
					if value != 0 {
						searchengines[key.Name()] = value
					}
				}
			} else {
				if key.Name() == "binsearch_alternative" {
					Log.Warn("Searchengine '%s' is no longer available", key.Name())
				} else {
					Log.Warn("Unknown searchengine '%s' in configuration file", key.Name())
				}
			}
		}
		if len(searchengines) == 0 {
			Log.Error("No searchengine set in configuration file")
			exit(1)
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

	// check debug parameter
	if !args.Debug {
		args.Debug = conf.General.Debug
	}

	// check target parameter
	for _, target := range strings.Split(conf.General.Target, ",") {
		target = strings.TrimSpace(target)
		if _, ok := targets[target]; ok {
			conf.General.Targets = append(conf.General.Targets, target)
		} else {
			Log.Warn("Undefined target '%s'", target)
		}
	}
	if len(conf.General.Targets) == 0 {
		Log.Error("Configuration error: no valid targets")
		exit(1)
	}
}
