package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/Tensai75/nzbparser"
)

type easynewsSearchResponse struct {
	Data []easynewsResult `json:"data"`
}

type easynewsResult struct {
	ID        string `json:"0"`
	Poster    string `json:"7"`
	FileName  string `json:"10"`
	Extension string `json:"11"`
	SetID     string `json:"19"`
	Sig       string `json:"sig"`
}

func easynewsSearch(engine SearchEngine, name string) error {
	searchString := engine.cleanSearchString(args.Header)
	searchURL := engine.searchURL
	if conf.Easynews.SubjectSearchOnly {
		searchURL += "&sbj=" + url.QueryEscape(searchString)
	} else {
		searchURL += "&gps=" + url.QueryEscape(searchString)
	}
	auth := base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", conf.Easynews.Username, conf.Easynews.Password))
	headers := map[string]string{
		"Authorization": "Basic " + auth,
	}
	body, err := loadURLWithHeaders(searchURL, headers)
	if err != nil {
		return fmt.Errorf("error calling search URL: %s", err.Error())
	}
	results, err := checkResponse(body)
	if err != nil {
		return fmt.Errorf("error checking search response: %s", err.Error())
	}
	formData, contentType, err := makeDownloadFormData(results)
	if err != nil {
		return fmt.Errorf("error creating download form data: %s", err.Error())
	}
	downloadURL := engine.downloadURL
	response, err := postURLWithHeaders(downloadURL, formData, contentType, headers)
	if err != nil {
		return fmt.Errorf("error calling download URL: %s", err.Error())
	}
	if nzb, err := nzbparser.ParseString(string(response)); err != nil {
		return fmt.Errorf("error parsing NZB file: %s", err.Error())
	} else {
		if nzb.Files.Len() > 0 {
			processResult(nzb, name)
		} else {
			return fmt.Errorf("the returned NZB file is empty")
		}
	}
	return nil
}

func checkResponse(response []byte) ([]easynewsResult, error) {
	var responseJSON easynewsSearchResponse

	if err := json.Unmarshal(response, &responseJSON); err != nil {
		Log.Debug("JSON parse error: %s", err.Error())
		Log.Debug("Response body: %s", response)
		return nil, fmt.Errorf("not a valid JSON response")
	}

	if len(responseJSON.Data) == 0 {
		return nil, fmt.Errorf("no results")
	}

	var firstGroupKey string
	var groupedResults []easynewsResult

	for _, item := range responseJSON.Data {
		basefilename, ok := easynewsBaseFilename(item.FileName)
		if !ok {
			continue
		}

		groupKey := basefilename + "\x00" + item.Poster
		if firstGroupKey == "" {
			firstGroupKey = groupKey
		}
		if groupKey == firstGroupKey {
			groupedResults = append(groupedResults, item)
		}
	}

	if len(groupedResults) > 0 {
		return groupedResults, nil
	}

	return nil, fmt.Errorf("no results")
}

func easynewsBaseFilename(fileName string) (string, bool) {
	basefilename, _, found := strings.Cut(fileName, ".")
	if !found {
		basefilename = fileName
	}
	if basefilename == "" {
		return "", false
	}
	return basefilename, true
}

func makeDownloadFormData(results []easynewsResult) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("autoNZB", "1"); err != nil {
		writer.Close()
		return nil, "", err
	}
	for i, item := range results {
		key := fmt.Sprintf("%d&sig=%s", i, item.Sig)
		value := fmt.Sprintf("%s|%s", item.ID, encodeFileName(item.FileName, item.Extension))
		if err := writer.WriteField(key, value); err != nil {
			writer.Close()
			return nil, "", err
		}
	}
	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body, contentType, nil
}

func encodeFileName(baseName, extension string) string {
	encodedBaseName := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(baseName)), "=")
	encodedExtension := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(extension)), "=")
	return encodedBaseName + ":" + encodedExtension
}
