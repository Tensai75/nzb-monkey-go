package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
)

// target functions for NZBGet
// function to get the categories
func nzbget_getCategories() (Categories, error) {

	// response structure
	type responseStruct struct {
		Result []struct {
			Name  string `json:"Name"`
			Value string `json:"Value"`
		} `json:"result"`
	}

	var categories Categories

	if response, err := request(conf.Nzbget, "GET", "jsonrpc/config", nil, nil, nil, ""); err != nil {
		return nil, err
	} else {
		var jsonResponse responseStruct
		if err := json.Unmarshal(response, &jsonResponse); err != nil {
			return nil, err
		}
		if len(jsonResponse.Result) > 0 {
			categoryRegexp := regexp.MustCompile(`Category\d+\.Name`)
			for _, item := range jsonResponse.Result {
				if categoryRegexp.Match([]byte(item.Name)) {
					categories = append(categories, item.Value)
				}
			}
		} else {
			return nil, fmt.Errorf("received an empty response")
		}
	}
	return categories, nil
}

// function to push the nzb file to the queue
func nzbget_push(nzb string, category string) error {

	fmt.Println()
	Log.Info("Pushing the NZB file to NZBGet...")

	// response structure
	type responseStruct struct {
		Result int `json:"result"`
	}

	// if category is empty set to default category
	if category == "" && conf.Nzbget.Category != "" {
		category = conf.Nzbget.Category
	}

	// if category is provided as argument use category from arguments
	if args.Category != "" {
		category = args.Category
	}

	// prepare body data
	var data = map[string]interface{}{
		"version": "1.1",
		"id":      0,
		"method":  "append",
		"params": []interface{}{
			args.Title + ".nzb",                         // Filename
			b64.StdEncoding.EncodeToString([]byte(nzb)), // Content (NZB File)
			category,              // Category
			0,                     // Priority
			false,                 // AddToTop
			conf.Nzbget.Addpaused, // AddPaused
			"",                    // DupeKey
			0,                     // DupeScore
			"ALL",                 // DupeMode
			map[string]interface{}{
				"*unpack:password": args.Password, // Post processing parameter: Password
			},
		},
	}

	if body, err := json.Marshal(data); err != nil {
		return fmt.Errorf("cannot create body data: %v", err)
	} else {
		if response, err := request(conf.Nzbget, "POST", "jsonrpc", nil, nil, bytes.NewBuffer(body), ""); err != nil {
			return err
		} else {
			var jsonResponse responseStruct
			if err := json.Unmarshal(response, &jsonResponse); err != nil {
				return err
			} else {
				if jsonResponse.Result > 0 {
					Log.Succ("The NZB file was pushed to NZBGet")
				} else {
					return fmt.Errorf("received an empty or unknown response")
				}
			}
		}
	}
	return nil
}
