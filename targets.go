package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"golang.org/x/exp/slices"
)

// nzb file target structure
type Target struct {
	name          string
	getCategories func() (Categories, error)
	push          func(string, string) error
}

// nzb file targets map
type Targets map[string]Target

// global nzb files targets map
var targets = Targets{
	"NZBGET": Target{
		name:          "NZBGet",
		getCategories: nzbget_getCategories,
		push:          nzbget_push,
	},
	"SABNZBD": Target{
		name:          "SABnzbd",
		getCategories: sabnzbd_getCategories,
		push:          sabnzbd_push,
	},
	"SYNOLOGYDLS": Target{
		name: "Synology DownloadStation",
		push: synologyds_push,
	},
	"EXECUTE": Target{
		name: "Download folder",
		push: execute_push,
	},
}

// http request function for the targets
func request(conf interface{}, httpMethod string, path string, headers map[string]string, queryParameters url.Values, body io.Reader, contentType string) ([]byte, error) {

	values := reflect.ValueOf(conf)
	transportCfg := http.DefaultTransport.(*http.Transport).Clone()

	// generate URL
	var scheme string
	if values.FieldByName("Ssl").Bool() {
		scheme = "https://"
		if values.FieldByName("SkipCheck").Bool() {
			transportCfg.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	} else {
		scheme = "http://"
	}
	fullUrl := fmt.Sprintf("%s%s", scheme, values.FieldByName("Host").String())
	if values.FieldByName("Port").Int() > 0 {
		fullUrl += fmt.Sprintf(":%d", values.FieldByName("Port").Int())
	}
	if strings.Trim(values.FieldByName("Basepath").String(), " /") != "" {
		fullUrl += fmt.Sprintf("/%s", strings.Trim(values.FieldByName("Basepath").String(), " /"))
	}
	if path != "" {
		fullUrl += fmt.Sprintf("/%s", strings.Trim(path, " /"))
	}

	// set up client
	client := &http.Client{Transport: transportCfg}
	u, err := url.Parse(fullUrl)
	if err != nil {
		return nil, err
	}

	// append the query parameters.
	q := u.Query()
	for k, v := range queryParameters {
		q.Set(k, strings.Join(v, ","))
	}
	// set the query to the encoded parameters
	u.RawQuery = q.Encode()

	// regardless of GET or POST, we can safely add the body
	req, err := http.NewRequest(httpMethod, u.String(), body)
	if err != nil {
		return nil, err
	}

	// for each header passed, add the header value to the request
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// if content type is provided, add to header
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// if basic auth username and password are set add auth header
	if values.FieldByName("BasicauthUsername").String() != "" && values.FieldByName("BasicauthPassword").String() != "" {
		req.SetBasicAuth(values.FieldByName("BasicauthUsername").String(), values.FieldByName("BasicauthPassword").String())
	}

	// finally, do the request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("calling %s returned empty response", u.String())
	}

	defer res.Body.Close()

	// read the response data
	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error calling %s:\nstatus: %s\nresponseData: %s", u.String(), res.Status, responseData)
	}

	return responseData, nil
}

func createMultipartBody(nzb string, filename string, compression string, compressionTypes []string) (*bytes.Buffer, string, error) {
	availableCompressions := map[string]func(*bytes.Buffer) io.WriteCloser{
		"zip": func(buffer *bytes.Buffer) io.WriteCloser {
			return gzip.NewWriter(buffer)
		},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("nzbfile", filename)
	if err != nil {
		return nil, "", err
	}

	if _, ok := availableCompressions[compression]; ok && slices.Contains(compressionTypes, compression) {
		before := len(nzb)
		fmt.Println()
		Log.Info("Compression ...")
		Log.Info("Uncompressed: %12s", prettyByteSize(before))

		compressedBuffer := &bytes.Buffer{}
		compressionWriter := availableCompressions[compression](compressedBuffer)

		if _, err := io.Copy(compressionWriter, strings.NewReader(nzb)); err != nil {
			compressionWriter.Close()
			return nil, "", err
		}

		if err := compressionWriter.Close(); err != nil {
			return nil, "", err
		}

		after := compressedBuffer.Len()
		Log.Info("Compressed: %14s", prettyByteSize(after))
		Log.Info("Compression: %12s", fmt.Sprintf("-%.2f%s", 100-(float64(after)/float64(before)*100), " %"))

		if _, err := io.Copy(part, compressedBuffer); err != nil {
			return nil, "", err
		}
	} else {
		if _, err := io.Copy(part, strings.NewReader(nzb)); err != nil {
			return nil, "", err
		}
	}

	contentType := writer.FormDataContentType()

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, contentType, nil
}
