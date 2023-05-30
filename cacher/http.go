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
func WriteHTTP(w io.Writer, input *Input) error {
	bw := bufio.NewWriter(w)
	defer func() { _ = bw.Flush() }()

	_, statusCodeError := bw.WriteString(fmt.Sprintf("HTTP %d\n", input.StatusCode))
	if statusCodeError != nil {
		return fmt.Errorf("bw.WriteString(StatusCode): %w", statusCodeError)
	}

	if input.URL != nil {
		_, urlError := bw.WriteString(fmt.Sprintf("%s: %s\n", CustomHeaderURL, input.URL.String()))
		if urlError != nil {
			return fmt.Errorf("bw.WriteString(URL): %w", urlError)
		}
	}

	cachingHeadersError := WriteHTTPCachingHeaders(bw, input)
	if cachingHeadersError != nil {
		return fmt.Errorf("WriteHTTPCachingHeaders: %w", cachingHeadersError)
	}

	httpHeaderError := writeHTTPHeader(bw, input)
	if httpHeaderError != nil {
		return fmt.Errorf("writeHTTPHeader: %w", httpHeaderError)
	}

	_, bodyError := writeHTTPBody(bw, input)
	return fmt.Errorf("writeHTTPBody: %w", bodyError)
}

// WriteHTTPCachingHeaders writes caching related headers
// like last modified, cache control, expires.
func WriteHTTPCachingHeaders(bw *bufio.Writer, input *Input) error {
	var (
		now     = time.Now()
		expires *time.Time
	)

	_, lastModifiedError := bw.WriteString(fmt.Sprintf("%s: %s\n", HeaderLastModified, now.Format(http.TimeFormat)))
	if lastModifiedError != nil {
		return fmt.Errorf("bw.WriteString(LastModified): %w", lastModifiedError)
	}

	if expires == nil {
		inputHeaderExpires := input.Header.Get(HeaderExpires)
		if len(inputHeaderExpires) > 0 {
			t, err := time.Parse(http.TimeFormat, inputHeaderExpires)
			if err == nil && t.After(now) {
				expires = &t
			}
		}
	}

	if expires == nil {
		inputHeaderCacheControl := input.Header.Get(HeaderCacheControl)
		maxAgeSubmatch := writeHTTPCachingHeadersMaxAgeRegexp.FindStringSubmatch(inputHeaderCacheControl)
		if maxAgeSubmatch != nil {
			if maxAge, err := strconv.ParseInt(maxAgeSubmatch[1], 10, 64); err == nil && maxAge > 0 {
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
		_, cacheControlError := bw.WriteString(fmt.Sprintf("%s: public, max-age=%d\n%s: %s\n",
			HeaderCacheControl, expires.Unix()-now.Unix(),
			HeaderExpires, expires.Format(http.TimeFormat),
		))
		if cacheControlError != nil {
			return fmt.Errorf("bw.WriteString(CacheControl): %w", cacheControlError)
		}

		_, expiresError := bw.WriteString(formatExpiresHeader(*expires))
		if expiresError != nil {
			return fmt.Errorf("bw.WriteString(Expires): %w", expiresError)
		}
	}

	return nil
}

func formatExpiresHeader(expires time.Time) string {
	return fmt.Sprintf("%s: %020d\n", CustomHeaderExpires, expires.UnixNano())
}

func writeHTTPHeader(bw *bufio.Writer, input *Input) error {
	if input.Header == nil {
		return nil
	}

	for headerKey, headerValues := range input.Header {
		switch headerKey {
		case HeaderCacheControl:
			continue
		case HeaderExpires:
			continue
		default:
			for _, headerValue := range headerValues {
				_, writeError := bw.WriteString(fmt.Sprintf("%s: %s\n", headerKey, headerValue))
				if writeError != nil {
					return fmt.Errorf("bw.WriteString(%s): %w", headerKey, writeError)
				}
			}
		}
	}

	return nil
}

func writeHTTPBody(bw *bufio.Writer, input *Input) (int, error) {
	bodyLen := len(input.Body)
	if bodyLen > 0 {
		_, contentLengthError := bw.WriteString(fmt.Sprintf("Content-Length: %d\n\n", bodyLen))
		if contentLengthError != nil {
			return 0, fmt.Errorf("bw.WriteString(Content-Length): %w", contentLengthError)
		}

		return bw.WriteString(input.Body)
	} else {
		return bw.WriteString("\n")
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
