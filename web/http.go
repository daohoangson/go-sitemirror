package web

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
)

var regexHTTPStatusCode = regexp.MustCompile(`^HTTP (\d+)\n$`)
var regexHTTPHeader = regexp.MustCompile(`^([^:]+): (.+)\n$`)

func ServeDownloaded(downloaded *crawler.Downloaded, w http.ResponseWriter) *CacheInfo {
	info := &CacheInfo{ResponseWriter: w}

	info.StatusCode = downloaded.StatusCode
	w.WriteHeader(info.StatusCode)

	if downloaded.HeaderLocation != nil {
		w.Header().Add("Location", downloaded.HeaderLocation.String())
		return info
	}

	if len(downloaded.ContentType) > 0 {
		w.Header().Add("Content-Type", downloaded.ContentType)
	}

	info.ContentLength = int64(len(downloaded.BodyString))
	var bytes []byte
	if info.ContentLength > 0 {
		bytes = []byte(downloaded.BodyString)
	} else if downloaded.BodyBytes != nil {
		bytes = downloaded.BodyBytes
		info.ContentLength = int64(len(downloaded.BodyBytes))
	}
	if info.ContentLength > 0 {
		w.Header().Add("Content-Length", fmt.Sprintf("%d", info.ContentLength))
		written, err := w.Write(bytes)
		info.ContentWritten = int64(written)
		info.Error = err
	}

	return info
}

func ServeHTTPCache(input io.Reader, w http.ResponseWriter) *CacheInfo {
	r := bufio.NewReader(input)
	info := &CacheInfo{ResponseWriter: w}

	ServeHTTPGetStatusCode(r, info)
	if info.Error != nil {
		w.WriteHeader(http.StatusNotImplemented)
		return info
	}

	ServeHTTPAddHeaders(r, info)
	if info.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return info
	}

	w.WriteHeader(info.StatusCode)
	ServeHTTPCopyContent(r, info, w)

	return info
}

func ServeHTTPGetStatusCode(r *bufio.Reader, info *CacheInfo) {
	line, err := r.ReadString('\n')
	if err != nil {
		info.ErrorType = ErrorReadLine
		info.Error = fmt.Errorf("Cannot read first line: %v", err)
		return
	}

	matches := regexHTTPStatusCode.FindStringSubmatch(line)
	if matches == nil {
		info.ErrorType = ErrorParseLine
		info.Error = fmt.Errorf("Unexpected first line: %s", line)
		return
	}

	statusCodeString := matches[1]
	statusCode, err := strconv.ParseUint(statusCodeString, 10, 32)
	if err != nil {
		info.ErrorType = ErrorParseInt
		info.Error = fmt.Errorf("Cannot convert status code from %s", statusCodeString)
		return
	}

	info.StatusCode = int(statusCode)
}

func ServeHTTPAddHeaders(r *bufio.Reader, info *CacheInfo) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			info.ErrorType = ErrorReadLine
			info.Error = fmt.Errorf("Cannot read header line: %v", err)
			return
		}

		if done := serveHTTPAddHeader(line, info); done {
			return
		}
	}
}

func ServeHTTPCopyContent(r *bufio.Reader, info *CacheInfo, w io.Writer) {
	if info.ContentLength == 0 {
		return
	}

	written, err := io.CopyN(w, r, info.ContentLength)
	info.ContentWritten = written
	if err != nil {
		info.ErrorType = ErrorWriteContent
		info.Error = err
	}
}

func serveHTTPAddHeader(line string, info *CacheInfo) bool {
	if line == "\n" {
		return true
	}

	matches := regexHTTPHeader.FindStringSubmatch(line)
	if matches == nil {
		info.ErrorType = ErrorParseLine
		info.Error = fmt.Errorf("Unexpected header line: %s", line)
		return true
	}

	headerKey := matches[1]
	headerValue := matches[2]

	switch headerKey {
	case "Content-Length":
		contentLength, err := strconv.ParseInt(headerValue, 10, 64)
		if err != nil {
			info.ErrorType = ErrorParseInt
			info.Error = fmt.Errorf("Cannot convert content length from %s", headerValue)
			return true
		}

		info.ContentLength = contentLength
	case cacher.HTTPHeaderExpires:
		if expires, err := strconv.ParseInt(headerValue, 10, 64); err == nil {
			t := time.Unix(0, expires)
			info.Expires = &t
		}

		return false
	default:
		if strings.HasPrefix(headerKey, cacher.HTTPHeaderPrefix) {
			// do not output internal headers
			return false
		}
	}

	info.ResponseWriter.Header().Add(headerKey, headerValue)
	return false
}
