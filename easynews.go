package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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
	Sig       string `json:"sig"`
}

func easynewsSearch(engine SearchEngine, name string) error {
	searchString := engine.cleanSearchString(args.Header)
	searchURL := fmt.Sprintf(engine.searchURL, url.QueryEscape(searchString))
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", conf.Easynews.Username, conf.Easynews.Password)))
	headers := map[string]string{
		"Authorization": "Basic " + auth,
	}
	body, err := loadURLWithHeaders(searchURL, headers)
	if err != nil {
		return fmt.Errorf("Error calling search URL: %s", err)
	}
	results, err := checkresponse(body)
	if err != nil {
		return fmt.Errorf("Error checking search response: %s", err)
	}
	formData, contentType, err := makeDownloadFormData(results)
	if err != nil {
		return fmt.Errorf("Error creating download form data: %s", err)
	}
	downloadURL := engine.downloadURL
	response, err := postURLWithHeaders(downloadURL, formData, contentType, headers)
	if err != nil {
		return fmt.Errorf("Error calling download URL: %s", err)
	}
	if nzb, err := nzbparser.ParseString(string(response)); err != nil {
		return fmt.Errorf("Error parsing NZB file: %s", err)
	} else {
		if nzb.Files.Len() > 0 {
			processResult(nzb, name)
		} else {
			return fmt.Errorf("The returned NZB file is empty")
		}
	}
	return nil
}

func checkresponse(response []byte) ([]easynewsResult, error) {
	var responseJSON easynewsSearchResponse

	if err := json.Unmarshal(response, &responseJSON); err != nil {
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

func loadURLWithHeaders(rawURL string, headers map[string]string) ([]byte, error) {
	return doRequestWithHeaders(http.MethodGet, rawURL, nil, "", headers)
}

func postURLWithHeaders(rawURL string, body io.Reader, contentType string, headers map[string]string) ([]byte, error) {
	return doRequestWithHeaders(http.MethodPost, rawURL, body, contentType, headers)
}

func doRequestWithHeaders(method string, rawURL string, body io.Reader, contentType string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return responseBody, nil
}
