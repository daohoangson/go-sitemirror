package cacher

import (
	"io"
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

	CheckCacheExists(*url.URL) bool
	Write(*Input) error
	WritePlaceholder(*url.URL) error
	Open(*url.URL) (io.ReadCloser, error)
}

// Input struct to be used with cacher func
type Input struct {
	StatusCode int
	URL        *url.URL
	TTL        time.Duration

	ContentType string
	Body        string

	Redirection *url.URL
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
