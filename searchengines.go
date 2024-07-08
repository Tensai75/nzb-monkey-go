package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		searchURL:   "https://nzbindex.com/search/json?sort=agedesc&hidespam=1&q=%s",
		downloadURL: "https://nzbindex.com/download/%s/",
		jsonPath:    "results.0.id",
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
	if body, err = loadURL(fmt.Sprintf(engine.searchURL, url.QueryEscape(searchString))); err == nil {
		if searchRegexp, err = regexp.Compile(engine.regexString); err == nil {
			if match = searchRegexp.FindStringSubmatch(body); match != nil {
				if len(match) >= engine.groupNo+1 {
					if body, err = loadURL(fmt.Sprintf(engine.downloadURL, match[engine.groupNo])); err == nil {
						if nzb, err := nzbparser.ParseString(body); err != nil {
							return err
						} else {
							if nzb.Files.Len() > 0 {
								processResult(nzb, name)
							} else {
								return fmt.Errorf("The returned NZB file is empty")
							}
						}
					}
				}
			} else {
				return fmt.Errorf("No results found")
			}
		}
	}
	return err
}

// default search function for json response
func jsonSearch(engine SearchEngine, name string) error {
	var err error
	var body string
	var result interface{}
	var value string
	searchString := engine.cleanSearchString(args.Header)
	if body, err = loadURL(fmt.Sprintf(engine.searchURL, url.QueryEscape(searchString))); err == nil {
		if err = json.Unmarshal([]byte(body), &result); err == nil {
			for _, value := range strings.Split(engine.jsonPath, ".") {
				if number, err := strconv.Atoi(value); err == nil {
					if len(result.([]interface{})) > number {
						result = result.([]interface{})[number]
					} else {
						return fmt.Errorf("No results found")
					}
				} else {
					if _, ok := result.(map[string]interface{})[value]; ok {
						result = result.(map[string]interface{})[value]
					} else {
						return fmt.Errorf("No results found")
					}
				}
			}
			if fmt.Sprintf("%T", result) == "float64" {
				value = fmt.Sprintf("%d", int(result.(float64)))
			} else if fmt.Sprintf("%T", result) == "string" {
				value = result.(string)
			} else {
				return fmt.Errorf("No results found")
			}
			if body, err = loadURL(fmt.Sprintf(engine.downloadURL, value)); err == nil {
				if nzb, err := nzbparser.ParseString(body); err != nil {
					return err
				} else {
					if nzb.Files.Len() > 0 {
						processResult(nzb, name)
					} else {
						return fmt.Errorf("The returned NZB file is empty")
					}
				}
			} else {
				return err
			}
		} else {
			return err
		}
	} else {
		return err
	}
	return nil
}

func loadURL(url string) (string, error) {
	if resp, err := http.Get(url); err != nil {
		return "", err
	} else {
		defer resp.Body.Close()
		if body, err := io.ReadAll(resp.Body); err != nil {
			return "", err
		} else {
			return string(body), nil
		}
	}
}
