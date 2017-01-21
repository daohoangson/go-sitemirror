package web

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	"github.com/daohoangson/go-sitemirror/web/internal"
)

var (
	regexHTTPStatusCode = regexp.MustCompile(`^HTTP (\d+)\n$`)
	regexHTTPHeader     = regexp.MustCompile(`^([^:]+): (.+)\n$`)
)

// ServeDownloaded streams data directly from downloaded struct to user
func ServeDownloaded(downloaded *crawler.Downloaded, info internal.ServeInfo) {
	info.SetStatusCode(downloaded.StatusCode)

	headerKeys := downloaded.GetHeaderKeys()
	for _, headerKey := range headerKeys {
		for _, headerValue := range downloaded.GetHeaderValues(headerKey) {
			info.AddHeader(headerKey, headerValue)
		}
	}

	info.WriteBody([]byte(downloaded.Body))
}

// ServeHTTPCache seves user request with content from cached data
func ServeHTTPCache(input io.Reader, info internal.ServeInfo) {
	r := bufio.NewReader(input)

	ServeHTTPGetStatusCode(r, info)
	if info.HasError() {
		return
	}

	ServeHTTPAddHeaders(r, info)
	if info.HasError() {
		return
	}

	info.CopyBody(r)
	return
}

// ServeHTTPGetStatusCode serves user request with status code from cached data
func ServeHTTPGetStatusCode(r *bufio.Reader, info internal.ServeInfo) {
	line, err := r.ReadString('\n')
	if err != nil {
		info.OnNoStatusCode(internal.ErrorReadLine, "Cannot read first line: %v", err)
		return
	}

	matches := regexHTTPStatusCode.FindStringSubmatch(line)
	if matches == nil {
		info.OnNoStatusCode(internal.ErrorParseLine, "Unexpected first line: %s", line)
		return
	}

	statusCodeString := matches[1]
	statusCode, err := strconv.ParseUint(statusCodeString, 10, 32)
	if err != nil {
		info.OnNoStatusCode(internal.ErrorParseInt, "Cannot convert status code from %s", statusCodeString)
		return
	}

	info.SetStatusCode(int(statusCode))
}

// ServeHTTPAddHeaders serves user request with headers from cached data
func ServeHTTPAddHeaders(r *bufio.Reader, info internal.ServeInfo) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			info.OnBrokenHeader(internal.ErrorReadLine, "Cannot read header line: %v", err)
			return
		}

		if done := serveHTTPAddHeader(line, info); done {
			return
		}
	}
}

func serveHTTPAddHeader(line string, info internal.ServeInfo) bool {
	if line == "\n" {
		return true
	}

	matches := regexHTTPHeader.FindStringSubmatch(line)
	if matches == nil {
		info.OnBrokenHeader(internal.ErrorParseLine, "Unexpected header line: %s", line)
		return true
	}

	headerKey := matches[1]
	headerValue := matches[2]

	switch headerKey {
	case "Content-Length":
		contentLength, err := strconv.ParseInt(headerValue, 10, 64)
		if err != nil {
			info.OnBrokenHeader(internal.ErrorParseInt, "Cannot convert content length from %s", headerValue)
			return true
		}

		info.SetContentLength(contentLength)
		return false
	case cacher.HTTPHeaderCrossHostRef:
		info.OnCrossHostRef()
		if info.HasError() {
			return true
		}
		return false
	case cacher.HTTPHeaderExpires:
		if expires, err := strconv.ParseInt(headerValue, 10, 64); err == nil {
			t := time.Unix(0, expires)
			info.SetExpires(t)
		}

		return false
	default:
		if strings.HasPrefix(headerKey, cacher.HTTPHeaderPrefix) {
			// do not output internal headers
			return false
		}
	}

	info.AddHeader(headerKey, headerValue)
	return false
}
