package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tensai75/nzbparser"
)

// search engine structure
type SearchEngine struct {
	name        string
	searchURL   string
	downloadURL string
	regexString string
	jsonPath    string
	groupNo     int
	search      func(engine SearchEngine, name string) error
	stringRegx  []RegexPattern
}

type RegexPattern struct {
	pattern     string
	replacement string
}

// search engines map
type SearchEngines map[string]SearchEngine

// global searchEngines map
var searchEngines = SearchEngines{
	"nzbindex": SearchEngine{
		name:        "NZBIndex",
		searchURL:   "https://nzbindex.com/api/search?q=%s",
		downloadURL: "https://nzbindex.com/api/download/%s.nzb",
		jsonPath:    "data.content.0.id",
		search:      jsonSearch,
	},
	"nzbking": SearchEngine{
		name:        "NZBKing",
		searchURL:   "https://nzbking.com/?q=%s",
		downloadURL: "https://nzbking.com/nzb:%s/",
		regexString: `href="\/nzb:(.+?)\/"`,
		groupNo:     1,
		search:      htmlSearch,
		stringRegx: []RegexPattern{
			{
				pattern:     `((\.| |_)-(\.| |_)|\.|_)+`,
				replacement: " ",
			},
		},
	},
	"binsearch": SearchEngine{
		name:        "Binsearch",
		searchURL:   "https://binsearch.info/search?q=%s",
		downloadURL: "https://binsearch.info/nzb?%s=on",
		regexString: `href="\/details\/([^"]+)"`,
		groupNo:     1,
		search:      htmlSearch,
	},
	"easynews": SearchEngine{
		name: "Easynews Search",
		// the search URL does not contain a placeholder for the search string because it is added later depending on the search type (subject or keyword search)
		searchURL:   "https://members.easynews.com/2.0/search/solr-search/?fly=2&YEAAAAAAAAAAAAH=NO&pby=1000&pno=1&sS=0&st=adv&safeO=0&sb=1",
		downloadURL: "https://members.easynews.com/2.0/api/dl-nzb",
		search:      easynewsSearch,
	},
	"directsearch": SearchEngine{
		name:   "NZB direct search",
		search: nzbdirectsearch,
	},
}

func (s *SearchEngine) cleanSearchString(searchString string) string {
	result := searchString
	for _, regexPattern := range s.stringRegx {
		r, err := regexp.Compile(regexPattern.pattern)
		if err != nil {
			continue
		}
		result = r.ReplaceAllString(result, regexPattern.replacement)
	}
	return result
}

// default search function for html response
func htmlSearch(engine SearchEngine, name string) error {
	var err error
	var body string
	var searchRegexp *regexp.Regexp
	var match []string
	searchString := engine.cleanSearchString(args.Header)
	body, err = loadURL(fmt.Sprintf(engine.searchURL, url.QueryEscape(searchString)))
	if err != nil {
		return fmt.Errorf("error calling search URL: %s", err.Error())
	}
	searchRegexp, err = regexp.Compile(engine.regexString)
	if err != nil {
		return fmt.Errorf("error compiling regex: %s", err.Error())
	}
	match = searchRegexp.FindStringSubmatch(body)
	if match == nil {
		return fmt.Errorf("no results found")
	}
	if len(match) < engine.groupNo+1 {
		return fmt.Errorf("invalid regex group number")
	}
	body, err = loadURL(fmt.Sprintf(engine.downloadURL, match[engine.groupNo]))
	if err != nil {
		return fmt.Errorf("error calling download URL: %s", err.Error())
	}
	nzb, err := nzbparser.ParseString(body)
	if err != nil {
		return fmt.Errorf("error parsing NZB file: %s", err.Error())
	}
	if nzb.Files.Len() == 0 {
		return fmt.Errorf("the returned NZB file is empty")
	}
	processResult(nzb, name)
	return nil
}

// default search function for json response
func jsonSearch(engine SearchEngine, name string) error {
	var err error
	var body string
	var result interface{}
	var value string
	searchString := engine.cleanSearchString(args.Header)
	body, err = loadURL(fmt.Sprintf(engine.searchURL, url.QueryEscape(searchString)))
	if err != nil {
		return fmt.Errorf("error calling search URL: %s", err.Error())
	}
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		Log.Debug("JSON parse error: %s", err.Error())
		Log.Debug("Response body: %s", body)
		return fmt.Errorf("not a valid JSON response")
	}
	for value := range strings.SplitSeq(engine.jsonPath, ".") {
		if number, err := strconv.Atoi(value); err == nil {
			if len(result.([]any)) > number && result.([]any)[number] != nil {
				result = result.([]any)[number]
			} else {
				return fmt.Errorf("no results found")
			}
		} else {
			if _, ok := result.(map[string]any)[value]; ok && result.(map[string]any)[value] != nil {
				result = result.(map[string]any)[value]
			} else {
				return fmt.Errorf("no results found")
			}
		}
	}
	if fmt.Sprintf("%T", result) == "float64" {
		value = fmt.Sprintf("%d", int(result.(float64)))
	} else if fmt.Sprintf("%T", result) == "string" {
		value = result.(string)
	} else {
		return fmt.Errorf("no results found")
	}
	body, err = loadURL(fmt.Sprintf(engine.downloadURL, value))
	if err != nil {
		return fmt.Errorf("error calling download URL: %s", err.Error())
	}
	nzb, err := nzbparser.ParseString(body)
	if err != nil {
		return fmt.Errorf("error parsing NZB file: %s", err.Error())
	}
	if nzb.Files.Len() == 0 {
		return fmt.Errorf("the returned NZB file is empty")
	}
	processResult(nzb, name)
	return nil
}
