package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"mime/multipart"
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
