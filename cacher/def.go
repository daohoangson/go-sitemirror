package cacher

import "net/url"

type Cacher interface {
	init(cacherMode)

	SetPath(string)
	GetPath() string
	CheckCacheExists(*url.URL) bool
	Write(*Input) error
}

// Input struct to be used with cacher func
type Input struct {
	StatusCode int
	URL        *url.URL

	ContentType string
	Body        string

	Redirection *url.URL
}

const (
	httpMode cacherMode = 1 + iota
)

type cacherMode int
