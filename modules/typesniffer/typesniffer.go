// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package typesniffer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"code.gitea.io/gitea/modules/util"
)

// Use at most this many bytes to determine Content Type.
const sniffLen = 1024

const (
	// SvgMimeType MIME type of SVG images.
	SvgMimeType = "image/svg+xml"
	// ApplicationOctetStream MIME type of binary files.
	ApplicationOctetStream = "application/octet-stream"
)

var (
	svgComment       = regexp.MustCompile(`(?s)<!--.*?-->`)
	svgTagRegex      = regexp.MustCompile(`(?si)\A\s*(?:(<!DOCTYPE\s+svg([\s:]+.*?>|>))\s*)*<svg\b`)
	svgTagInXMLRegex = regexp.MustCompile(`(?si)\A<\?xml\b.*?\?>\s*(?:(<!DOCTYPE\s+svg([\s:]+.*?>|>))\s*)*<svg\b`)
)

var audioFormats = map[string][][]byte{
	"audio/aac":        {[]byte("ADIF"), []byte("ADTS")},
	"audio/amr":        {[]byte("#!AMR\r\n")},
	"audio/3pg":        {[]byte("ftyp3gp")},
	"audio/m4a":        {[]byte("ftypmp41"), []byte("ftypmp42")},
	"audio/x-ms-wma":   {[]byte("ftypWMAV")},
	"audio/x-ape":      {[]byte("APE"), []byte("MAC")},
	"audio/x-flac":     {[]byte("fLaC")},
	"audio/alac":       {[]byte("alac")},
	"audio/x-wavpack":  {[]byte("wvpk")},
	"audio/silk-v3":    {[]byte("#!SILK_V3")},
	"audio/opus":       {[]byte("OpusHead")},
	"audio/x-musepack": {[]byte("MPCK")},
	"audio/ac3":        {[]byte("AC3")},
	"audio/dts":        {[]byte("DTS")},
}

var videoFormats = map[string][][]byte{
	"video/x-msvideo":       {[]byte("RIFFAVI ")},
	"video/x-flv":           {[]byte("FLV")},
	"video/mp4":             {[]byte("ftypmp41"), []byte("ftypmp42")},
	"video/mpeg":            {{0x00, 0x00, 0x01, 0xBA}, {0x00, 0x00, 0x01, 0xB3}},
	"video/x-ms-wmv":        {[]byte("WMV1"), []byte("WMV2")},
	"video/quicktime":       {{0x6D, 0x6F, 0x6F, 0x76}},
	"video/rmvb":            {[]byte(".RMF")},
	"application/x-mpegURL": {[]byte("#EXTM3U\n#EXT-X")},
}

var imageFormats = map[string][][]byte{
	"image/tiff":   {{0x49, 0x49, 0x2A, 0x00}, {0x4D, 0x4D, 0x00, 0x2A}},
	"image/heif":   {[]byte("ftypheic"), []byte("ftypmif1")},
	"image/x-icon": {{0x00, 0x00, 0x01, 0x00}},
	"image/x-tga":  {{0x00, 0x00, 0x02, 0x00}},
}

var documentFormats = map[string][][]byte{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {{0x50, 0x4B, 0x04, 0x04}},
	"application/msword":            {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},
	"application/vnd.ms-excel":      {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},
	"application/vnd.ms-powerpoint": {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {{0x50, 0x4B, 0x03,
		0x04, 0x14, 0x00, 0x06, 0x00}},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": {{0x50, 0x4B, 0x03,
		0x04, 0x14, 0x00, 0x06, 0x00}},
	"application/vnd.openxmlformats-officedocument.presentationml.slideshow": {{0x50, 0x4B, 0x03,
		0x04, 0x14, 0x00, 0x06, 0x00}},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.template": {{0x50, 0x4B, 0x03,
		0x04, 0x14, 0x00, 0x06, 0x00}},
	"application/vnd.ms-excel.template.macroEnabled.12": {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},
	"application/vnd.ms-excel.sheet.binary.macroEnabled.12": {{0x09, 0x08, 0x10, 0x00,
		0x00, 0x06, 0x05, 0x00}},
	"application/vnd.ms-excel.sheet.macroEnabled.12": {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1,
		0x1A, 0xE1}},
	"application/epub+zip": {[]byte("PK\x03\x04")},
}

// SniffedType contains information about a blobs type.
type SniffedType struct {
	contentType string
}

// IsText etects if content format is plain text.
func (ct SniffedType) IsText() bool {
	return strings.Contains(ct.contentType, "text/")
}

func (ct SniffedType) IsDocument() bool {
	if ct.IsText() {
		return true
	} else {
		if _, ok := documentFormats[ct.contentType]; ok {
			return true
		}
		return false
	}
}

// IsImage detects if data is an image format
func (ct SniffedType) IsImage() bool {
	return strings.Contains(ct.contentType, "image/")
}

// IsSvgImage detects if data is an SVG image format
func (ct SniffedType) IsSvgImage() bool {
	return strings.Contains(ct.contentType, SvgMimeType)
}

// IsPDF detects if data is a PDF format
func (ct SniffedType) IsPDF() bool {
	return strings.Contains(ct.contentType, "application/pdf")
}

// IsVideo detects if data is an video format
func (ct SniffedType) IsVideo() bool {
	return strings.Contains(ct.contentType, "video/") || ct.contentType == "application/x-mpegURL"
}

// IsAudio detects if data is an video format
func (ct SniffedType) IsAudio() bool {
	return strings.Contains(ct.contentType, "audio/")
}

// IsRepresentableAsText returns true if file content can be represented as
// plain text or is empty.
func (ct SniffedType) IsRepresentableAsText() bool {
	return ct.IsText() || ct.IsSvgImage()
}

// IsBrowsableBinaryType returns whether a non-text type can be displayed in a browser
func (ct SniffedType) IsBrowsableBinaryType() bool {
	return ct.IsImage() || ct.IsSvgImage() || ct.IsPDF() || ct.IsVideo() || ct.IsAudio()
}

// GetMimeType returns the mime type
func (ct SniffedType) GetMimeType() string {
	return strings.SplitN(ct.contentType, ";", 2)[0]
}

// DetectContentType extends http.DetectContentType with more content types. Defaults to text/unknown if input is empty.
func DetectContentType(data []byte) SniffedType {
	if len(data) == 0 {
		return SniffedType{"text/unknown"}
	}

	ct := http.DetectContentType(data)

	if len(data) > sniffLen {
		data = data[:sniffLen]
	}

	// SVG is unsupported by http.DetectContentType, https://github.com/golang/go/issues/15888

	detectByHTML := strings.Contains(ct, "text/plain") || strings.Contains(ct, "text/html")
	detectByXML := strings.Contains(ct, "text/xml")
	if detectByHTML || detectByXML {
		dataProcessed := svgComment.ReplaceAll(data, nil)
		dataProcessed = bytes.TrimSpace(dataProcessed)
		if detectByHTML && svgTagRegex.Match(dataProcessed) ||
			detectByXML && svgTagInXMLRegex.Match(dataProcessed) {
			ct = SvgMimeType
		}
	}

	if strings.HasPrefix(ct, "audio/") && bytes.HasPrefix(data, []byte("ID3")) {
		// The MP3 detection is quite inaccurate, any content with "ID3" prefix will result in "audio/mpeg".
		// So remove the "ID3" prefix and detect again, if result is text, then it must be text content.
		// This works especially because audio files contain many unprintable/invalid characters like `0x00`
		ct2 := http.DetectContentType(data[3:])
		if strings.HasPrefix(ct2, "text/") {
			ct = ct2
		}
	}

	if ct == "application/ogg" {
		dataHead := data
		if len(dataHead) > 256 {
			dataHead = dataHead[:256] // only need to do a quick check for the file header
		}
		if bytes.Contains(dataHead, []byte("theora")) || bytes.Contains(dataHead, []byte("dirac")) {
			ct = "video/ogg" // ogg is only used for some video formats, and it's not popular
		} else {
			ct = "audio/ogg" // for most cases, it is used as an audio container
		}
	}

	if ct == "application/octet-stream" {
		dataHead := data
		if len(dataHead) > 256 {
			dataHead = dataHead[:256] // only need to do a quick check for the file header
		}
		for ctName, headFormats := range documentFormats {
			for _, prefix := range headFormats {
				if bytes.HasPrefix(dataHead, prefix) {
					ct = ctName
				}
			}
		}

		for ctName, headFormats := range audioFormats {
			for _, prefix := range headFormats {
				if bytes.HasPrefix(dataHead, prefix) {
					ct = ctName
				}
			}
		}

		for ctName, headFormats := range videoFormats {
			for _, prefix := range headFormats {
				if bytes.HasPrefix(dataHead, prefix) {
					ct = ctName
				}
			}
		}

		for ctName, headFormats := range imageFormats {
			for _, prefix := range headFormats {
				if bytes.HasPrefix(dataHead, prefix) {
					ct = ctName
				}
			}
		}
	}

	return SniffedType{ct}
}

// DetectContentTypeFromReader guesses the content type contained in the reader.
func DetectContentTypeFromReader(r io.Reader) (SniffedType, error) {
	buf := make([]byte, sniffLen)
	n, err := util.ReadAtMost(r, buf)
	if err != nil {
		return SniffedType{}, fmt.Errorf("DetectContentTypeFromReader io error: %w", err)
	}
	buf = buf[:n]

	return DetectContentType(buf), nil
}
