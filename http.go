package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// defaultClient is used for all simple HTTP helpers with a 30s timeout.
var defaultClient = &http.Client{Timeout: defaultTimeout}

// loadURL performs a simple GET request and returns the response body as a string.
func loadURL(rawURL string) (string, error) {
	body, err := loadURLWithHeaders(rawURL, nil)
	return string(body), err
}

// loadURLWithHeaders performs a GET request with optional headers.
func loadURLWithHeaders(rawURL string, headers map[string]string) ([]byte, error) {
	return doRequest(defaultClient, http.MethodGet, rawURL, nil, "", headers)
}

// postURLWithHeaders performs a POST request with a body, content type, and optional headers.
func postURLWithHeaders(rawURL string, body io.Reader, contentType string, headers map[string]string) ([]byte, error) {
	return doRequest(defaultClient, http.MethodPost, rawURL, body, contentType, headers)
}

// doRequest is the core HTTP helper. It executes the request using the provided client.
func doRequest(client *http.Client, method string, rawURL string, body io.Reader, contentType string, headers map[string]string) ([]byte, error) {
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error calling %s:\nstatus: %s\nresponse: %s", rawURL, resp.Status, responseBody)
	}

	return responseBody, nil
}

// request builds and executes an HTTP request for a target, constructing the URL from the
// provided config struct (accessed via reflection) and applying optional headers, query
// parameters, body, and content type.
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

	// append the query parameters
	u, err := url.Parse(fullUrl)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range queryParameters {
		q.Set(k, strings.Join(v, ","))
	}
	u.RawQuery = q.Encode()

	// merge caller headers and basic auth into a single map
	allHeaders := make(map[string]string, len(headers))
	maps.Copy(allHeaders, headers)
	if username := values.FieldByName("BasicauthUsername").String(); username != "" {
		if password := values.FieldByName("BasicauthPassword").String(); password != "" {
			allHeaders["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		}
	}

	client := &http.Client{Transport: transportCfg}
	if t := values.FieldByName("Timeout").Int(); t > 0 {
		client.Timeout = time.Duration(t) * time.Second
	} else {
		client.Timeout = defaultTimeout
	}
	return doRequest(client, httpMethod, u.String(), body, contentType, allHeaders)
}
