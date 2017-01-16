package web

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
)

type Server interface {
	init(cacher.Cacher, *logrus.Logger)

	GetCacher() cacher.Cacher
	SetOnCacheIssue(func(CacheIssue))

	ListenAndServe(string, int) (net.Listener, error)
	Serve(string, http.ResponseWriter, *http.Request)
}

type CacheIssue struct {
	URL  *url.URL
	Type cacheIssueType
	Info *CacheInfo
}

type CacheInfo struct {
	ResponseWriter http.ResponseWriter
	ErrorType      errorType
	Error          error

	StatusCode     int
	ContentLength  int64
	ContentWritten int64
	Expires        *time.Time
}

const (
	CacheNotFound cacheIssueType = 1 + iota
	CacheError
	CacheExpired
)

const (
	ErrorReadLine errorType = 1 + iota
	ErrorParseLine
	ErrorParseInt
	ErrorWriteContent
)

type cacheIssueType int
type errorType int
