package cacher

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

var (
	writeHTTPPlaceholderFirstLine = fmt.Sprintf("HTTP %d\n", http.StatusNoContent)
)

// WriteHTTP writes cache data in http format
func WriteHTTP(w io.Writer, input *Input) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bw.WriteString(fmt.Sprintf("HTTP %d\n", input.StatusCode))

	if input.URL != nil {
		bw.WriteString(fmt.Sprintf("%s: %s\n", HTTPHeaderURL, input.URL.String()))
	}

	now := time.Now()
	bw.WriteString(fmt.Sprintf("Last-Modified: %s\n", now.Format(http.TimeFormat)))
	if input.TTL > 0 {
		expires := now.Add(input.TTL)
		bw.WriteString(fmt.Sprintf("Cache-Control: public\nExpires: %s\n", expires.Format(http.TimeFormat)))
		bw.WriteString(writeHTTPFormatExpiresHeader(expires))
	}

	if input.StatusCode >= 200 && input.StatusCode <= 299 {
		writeHTTP2xx(bw, input)
	} else if input.StatusCode >= 300 && input.StatusCode <= 399 {
		writeHTTP3xx(bw, input)
	} else {
		bw.WriteString("\n")
	}
}

func writeHTTPFormatExpiresHeader(expires time.Time) string {
	return fmt.Sprintf("%s: %020d\n", HTTPHeaderExpires, expires.UnixNano())
}

func writeHTTP2xx(bw *bufio.Writer, input *Input) {
	if len(input.ContentType) > 0 {
		bw.WriteString(fmt.Sprintf("Content-Type: %s\n", input.ContentType))
	}

	bodyLen := len(input.Body)
	if bodyLen > 0 {
		bw.WriteString(fmt.Sprintf("Content-Length: %d\n\n", bodyLen))
		bw.WriteString(input.Body)
	} else {
		bw.WriteString("\n")
	}
}

func writeHTTP3xx(bw *bufio.Writer, input *Input) {
	if input.Redirection != nil {
		bw.WriteString(fmt.Sprintf("Location: %s\n\n", input.Redirection.String()))
		return
	}
}

func writeHTTPPlaceholder(w io.Writer, url *url.URL, expires time.Time) error {
	_, writeError := w.Write([]byte(fmt.Sprintf(
		"%s%s: %s\n%s\n",
		writeHTTPPlaceholderFirstLine,
		HTTPHeaderURL, url.String(),
		writeHTTPFormatExpiresHeader(expires),
	)))

	return writeError
}
