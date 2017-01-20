package web

import (
	"io"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/web/internal"
)

// Server represents an object that can serve user request with cached data
type Server interface {
	init(cacher.Cacher, *logrus.Logger)

	GetCacher() cacher.Cacher
	SetOnServerIssue(func(*ServerIssue))

	ListenAndServe(*url.URL, int) (io.Closer, error)
	GetListeningPort(string) (int, error)
	Serve(*url.URL, http.ResponseWriter, *http.Request) internal.ServeInfo
	Stop() []string
}

// ServerIssue represents an issue that cannot be handled by the server itself
type ServerIssue struct {
	URL  *url.URL
	Type serverIssueType
	Info internal.ServeInfo
}

const (
	// MethodNotAllowed server issue type when user request is made with restricted method
	MethodNotAllowed serverIssueType = 1 + iota
	// CacheNotFound server issue type when existing cache cannot be found
	CacheNotFound
	// CacheError server issue type when cache cannot be read
	CacheError
	// CacheExpired server issue type when cache has expired
	CacheExpired
)

type serverIssueType int
