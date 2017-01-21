package cacher

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
)

// Cacher represents an object that can write/open cached data
type Cacher interface {
	init(*logrus.Logger)

	GetMode() cacherMode
	SetPath(string)
	GetPath() string
	SetDefaultTTL(time.Duration)
	GetDefaultTTL() time.Duration

	CheckCacheExists(*url.URL) bool
	Write(*Input) error
	Bump(*url.URL, time.Duration) error
	WritePlaceholder(*url.URL, time.Duration) error
	Open(*url.URL) (io.ReadCloser, error)
}

// Input struct to be used with cacher func
type Input struct {
	StatusCode int
	URL        *url.URL
	TTL        time.Duration

	Body   string
	Header http.Header
}

const (
	// SchemeDefault the default scheme if none specified
	SchemeDefault = "http"
)

const (
	// CustomHeaderPrefix prefix for internal headers
	CustomHeaderPrefix = "X-Mirror-"
	// CustomHeaderURL header key for cache url
	CustomHeaderURL = "X-Mirror-Url"
	// CustomHeaderCrossHostRef header key for cross host reference flag
	CustomHeaderCrossHostRef = "X-Mirror-Cross-Host-Ref"
	// CustomHeaderExpires header key for cache expire time in nano second
	CustomHeaderExpires = "X-Mirror-Expires"
)

const (
	// HeaderCacheControl http cache control header key
	HeaderCacheControl = "Cache-Control"
	// HeaderContentLength http content length header key
	HeaderContentLength = "Content-Length"
	// HeaderContentType http content type header key
	HeaderContentType = "Content-Type"
	// HeaderExpires http expires header key
	HeaderExpires = "Expires"
	// HeaderLastModified http last modified header key
	HeaderLastModified = "Last-Modified"
	// HeaderLocation http location header key
	HeaderLocation = "Location"
)

const (
	// HTTPMode cacher mode http
	HTTPMode cacherMode = 1 + iota
)

type cacherMode int
