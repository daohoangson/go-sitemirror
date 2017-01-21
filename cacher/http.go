package cacher

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

var (
	writeHTTPCachingHeadersMaxAgeRegexp = regexp.MustCompile(`max-age\s*=\s*(\d+)(\s|$)`)
	writeHTTPPlaceholderFirstLine       = fmt.Sprintf("HTTP %d\n", http.StatusNoContent)
)

// WriteHTTP writes cache data in http format
func WriteHTTP(w io.Writer, input *Input) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bw.WriteString(fmt.Sprintf("HTTP %d\n", input.StatusCode))

	if input.URL != nil {
		bw.WriteString(fmt.Sprintf("%s: %s\n", CustomHeaderURL, input.URL.String()))
	}

	WriteHTTPCachingHeaders(bw, input)
	writeHTTPHeader(bw, input)
	writeHTTPBody(bw, input)
}

// WriteHTTPCachingHeaders writes caching related headers
// like last modified, cache control, expires.
func WriteHTTPCachingHeaders(bw *bufio.Writer, input *Input) {
	var (
		now     = time.Now()
		expires *time.Time
	)

	bw.WriteString(fmt.Sprintf("%s: %s\n", HeaderLastModified, now.Format(http.TimeFormat)))

	if expires == nil {
		inputHeaderExpires := input.Header.Get(HeaderExpires)
		if len(inputHeaderExpires) > 0 {
			t, err := time.Parse(http.TimeFormat, inputHeaderExpires)
			if err == nil {
				expires = &t
			}
		}
	}

	if expires == nil {
		inputHeaderCacheControl := input.Header.Get(HeaderCacheControl)
		maxAgeSubmatch := writeHTTPCachingHeadersMaxAgeRegexp.FindStringSubmatch(inputHeaderCacheControl)
		if maxAgeSubmatch != nil {
			if maxAge, err := strconv.ParseInt(maxAgeSubmatch[1], 10, 64); err == nil {
				expires = &time.Time{}
				*expires = now.Add(time.Duration(maxAge) * time.Second)
			}
		}
	}

	if expires == nil && input.TTL > 0 {
		expires = &time.Time{}
		*expires = now.Add(input.TTL)
	}

	if expires != nil {
		bw.WriteString(fmt.Sprintf("%s: public, max-age=%d\n%s: %s\n",
			HeaderCacheControl, expires.Unix()-now.Unix(),
			HeaderExpires, expires.Format(http.TimeFormat),
		))
		bw.WriteString(formatExpiresHeader(*expires))
	}
}

func formatExpiresHeader(expires time.Time) string {
	return fmt.Sprintf("%s: %020d\n", CustomHeaderExpires, expires.UnixNano())
}

func writeHTTPHeader(bw *bufio.Writer, input *Input) {
	if input.Header == nil {
		return
	}

	for headerKey, headerValues := range input.Header {
		switch headerKey {
		case HeaderCacheControl:
		case HeaderExpires:
		default:
			for _, headerValue := range headerValues {
				bw.WriteString(fmt.Sprintf("%s: %s\n", headerKey, headerValue))
			}
		}
	}
}

func writeHTTPBody(bw *bufio.Writer, input *Input) {
	bodyLen := len(input.Body)
	if bodyLen > 0 {
		bw.WriteString(fmt.Sprintf("Content-Length: %d\n\n", bodyLen))
		bw.WriteString(input.Body)
	} else {
		bw.WriteString("\n")
	}
}

func writeHTTPPlaceholder(w io.Writer, url *url.URL, expires time.Time) error {
	_, writeError := w.Write([]byte(fmt.Sprintf(
		"%s%s: %s\n%s\n",
		writeHTTPPlaceholderFirstLine,
		CustomHeaderURL, url.String(),
		formatExpiresHeader(expires),
	)))

	return writeError
}
