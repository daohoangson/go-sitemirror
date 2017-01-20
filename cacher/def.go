package cacher

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
)

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
	HTTPHeaderPrefix  = "X-Mirror-"
	HTTPHeaderURL     = "X-Mirror-Url"
	HTTPHeaderExpires = "X-Mirror-Expires"
)

const (
	HttpMode cacherMode = 1 + iota
)

type cacherMode int
