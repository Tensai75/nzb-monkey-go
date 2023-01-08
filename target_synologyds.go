package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	color "github.com/TwiN/go-color"
)

// ds response structure
type dsResponseStruct struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   struct {
		Code float64 `json:"code"`
	} `json:"Error"`
}

// ds options structure
type dsOptions struct {
	sid        string
	path       string
	minVersion int
	maxVersion int
}

// target functions for Synology Diskstation
// function to push the nzb file to the queue
func synologyds_push(nzb string, category string) error {

	fmt.Printf("\n   Pushing the NZB file to Synology DownloadStation...\n")

	if result, err := synologyds_authenticate(); err != nil {
		return err
	} else {

		path := fmt.Sprintf("/%s/SYNO.DownloadStation2.Task", result.path)

		// prepare query
		query := make(url.Values)

		// add values
		// sid is required as get parameter!
		query.Add("_sid", result.sid)

		// prepare body data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// add parameters
		writer.WriteField("api", "SYNO.DownloadStation2.Task")
		writer.WriteField("method", "create")
		writer.WriteField("version", fmt.Sprintf("%d", result.maxVersion))
		writer.WriteField("type", "\"file\"")
		writer.WriteField("destination", "\"\"")
		writer.WriteField("create_list", "false")
		writer.WriteField("mtime", fmt.Sprintf("%d", time.Now().Unix()))
		writer.WriteField("size", fmt.Sprintf("%d", len(nzb)))
		writer.WriteField("file", "[\"torrent\"]")
		writer.WriteField("extract_password", fmt.Sprintf("\"%s\"", args.Password))

		// add the nzb file
		part, _ := writer.CreateFormFile("torrent", args.Title+".nzb")
		io.Copy(part, strings.NewReader(nzb))
		writer.Close()

		if response, err := request(conf.Synologyds, "POST", "webapi"+path, nil, query, body, writer.FormDataContentType()); err != nil {
			return err
		} else {
			var jsonResponse dsResponseStruct
			if err := json.Unmarshal(response, &jsonResponse); err != nil {
				return err
			} else {
				if jsonResponse.Success {
					fmt.Printf("%s   SUCCESS:  The NZB file was pushed to Synology DownloadStation %s\n", color.Green, color.Reset)
					return nil
				} else if jsonResponse.Error.Code > 0 {
					return synologyds_checkError(int(jsonResponse.Error.Code))
				}
			}
		}

	}

	return fmt.Errorf("unknown response")
}

func synologyds_authenticate() (dsOptions, error) {

	// return value
	var options dsOptions

	// prepare query
	query := make(url.Values)

	// add values
	query.Add("api", "SYNO.API.Info")
	query.Add("version", "1")
	query.Add("method", "query")
	query.Add("query", "SYNO.API.Auth,SYNO.DownloadStation2.Task")

	if response, err := request(conf.Synologyds, "GET", "webapi/query.cgi", nil, query, nil, ""); err != nil {
		return options, err
	} else {
		var jsonResponse dsResponseStruct
		if err := json.Unmarshal(response, &jsonResponse); err != nil {
			return options, err
		} else {
			if jsonResponse.Success {
				if _, ok := jsonResponse.Data["SYNO.DownloadStation2.Task"]; ok {
					options.path = jsonResponse.Data["SYNO.DownloadStation2.Task"].(map[string]interface{})["path"].(string)
					options.minVersion = int(jsonResponse.Data["SYNO.DownloadStation2.Task"].(map[string]interface{})["minVersion"].(float64))
					options.maxVersion = int(jsonResponse.Data["SYNO.DownloadStation2.Task"].(map[string]interface{})["maxVersion"].(float64))
					if _, ok := jsonResponse.Data["SYNO.API.Auth"]; ok {
						// set path
						path := jsonResponse.Data["SYNO.API.Auth"].(map[string]interface{})["path"].(string)

						// prepare query
						query := make(url.Values)

						// add values
						query.Add("api", "SYNO.API.Auth")
						query.Add("version", fmt.Sprintf("%d", int(jsonResponse.Data["SYNO.API.Auth"].(map[string]interface{})["maxVersion"].(float64))))
						query.Add("method", "login")
						query.Add("account", conf.Synologyds.Username)
						query.Add("passwd", conf.Synologyds.Password)
						query.Add("session", "DownloadStation")
						query.Add("format", "sid")

						if response, err := request(conf.Synologyds, "GET", "webapi/"+path, nil, query, nil, ""); err != nil {
							return options, err
						} else {
							var jsonResponse dsResponseStruct
							if err := json.Unmarshal(response, &jsonResponse); err != nil {
								return options, err
							} else {
								if jsonResponse.Success {
									if _, ok := jsonResponse.Data["sid"]; ok {
										options.sid = jsonResponse.Data["sid"].(string)
										return options, nil
									}
								}
							}
						}
					}
				}
			} else if jsonResponse.Error.Code > 0 {
				return options, synologyds_checkError(int(jsonResponse.Error.Code))
			}
		}
	}

	return options, fmt.Errorf("unknown response while authenticating")
}

func synologyds_checkError(errorCode int) error {

	// Synology error codes
	synologyErrorCodes := map[int]string{
		100: "Unknown error.",
		101: "No parameter of API, method or version.",
		102: "The requested API does not exist.",
		103: "The requested method does not exist.",
		104: "The requested version does not support the functionality.",
		105: "The logged in session does not have permission.",
		106: "Session timeout.",
		107: "Session interrupted by duplicated login.",
		108: "Failed to upload the file.",
		109: "The network connection is unstable or the system is busy.",
		110: "The network connection is unstable or the system is busy.",
		111: "The network connection is unstable or the system is busy.",
		112: "Preserve for other purpose.",
		113: "Preserve for other purpose.",
		114: "Lost parameters for this API.",
		115: "Not allowed to upload a file.",
		116: "Not allowed to perform for a demo site.",
		117: "The network connection is unstable or the system is busy.",
		118: "The network connection is unstable or the system is busy.",
		119: "Invalid session.",
		400: "No such account or incorrect password.",
		401: "Disabled account.",
		402: "Denied permission.",
		403: "2-factor authentication code required.",
		404: "Failed to authenticate 2-factor authentication code.",
		406: "Enforce to authenticate with 2-factor authentication code.",
		407: "Blocked IP source.",
		408: "Expired password cannot change.",
		409: "Expired password.",
		410: "Password must be changed.",
	}

	errorText := synologyErrorCodes[100]
	if _, ok := synologyErrorCodes[errorCode]; ok {
		errorText = synologyErrorCodes[errorCode]
	}
	return fmt.Errorf("%d - %s", errorCode, errorText)

}
