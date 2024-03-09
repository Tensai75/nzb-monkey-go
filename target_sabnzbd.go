package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
)

// target functions for SABnzbd
// function to get the categories
func sabnzbd_getCategories() (Categories, error) {

	// response struct
	type Response struct {
		Categories Categories `json:"categories"`
	}
	var categories Response

	// prepare query
	query := make(url.Values)

	// add values
	query.Add("mode", "get_cats")
	query.Add("output", "json")
	query.Add("apikey", conf.Sabnzbd.Nzbkey)

	if response, err := request(conf.Sabnzbd, "GET", "api", nil, query, nil, ""); err != nil {
		return nil, err
	} else {
		if err := json.Unmarshal(response, &categories); err != nil {
			return nil, err
		} else {
			if len(categories.Categories) > 1 {
				return categories.Categories[1:], nil
			} else {
				return nil, nil
			}
		}
	}
}

// function to push the nzb file to the queue
func sabnzbd_push(nzb string, category string) error {

	fmt.Println()
	Log.Info("Pushing the NZB file to SABnzbd...")

	// response structure
	type responseStruct struct {
		Status  bool     `json:"status"`
		Nzo_ids []string `json:"nzo_ids"`
	}

	// if category is empty set to default category
	if category == "" && conf.Sabnzbd.Category != "" {
		category = conf.Sabnzbd.Category
	}

	// if category is provided as argument use category from arguments
	if args.Category != "" {
		category = args.Category
	}

	// set addPaused option
	addPaused := "-100"
	if conf.Sabnzbd.Addpaused {
		addPaused = "-2"
	}

	// prepare query
	query := make(url.Values)

	// add values
	query.Add("mode", "addfile")
	query.Add("output", "json")
	query.Add("apikey", conf.Sabnzbd.Nzbkey)
	query.Add("nzbname", args.Title+".nzb")
	query.Add("password", args.Password)
	query.Add("cat", category)
	query.Add("priority", addPaused)

	// prepare body data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("nzbfile", args.Title+".nzb")
	io.Copy(part, strings.NewReader(nzb))
	writer.Close()

	if response, err := request(conf.Sabnzbd, "POST", "api", nil, query, body, writer.FormDataContentType()); err != nil {
		return err
	} else {
		var jsonResponse responseStruct
		if err := json.Unmarshal(response, &jsonResponse); err != nil {
			return err
		} else {
			if jsonResponse.Status && len(jsonResponse.Nzo_ids) > 0 {
				Log.Succ("The NZB file was pushed to SABnzbd")
			} else {
				return fmt.Errorf("received an empty or unknown response")
			}
		}
	}
	return nil
}
