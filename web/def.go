package web

import (
	"io"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/web/internal"
)

type Server interface {
	init(cacher.Cacher, *logrus.Logger)

	GetCacher() cacher.Cacher
	SetOnServerIssue(func(*ServerIssue))

	ListenAndServe(*url.URL, int) (io.Closer, error)
	GetListeningPort(string) (int, error)
	Serve(*url.URL, http.ResponseWriter, *http.Request) internal.ServeInfo
	Stop() []string
}

type ServerIssue struct {
	URL  *url.URL
	Type serverIssueType
	Info internal.ServeInfo
}

const (
	MethodNotAllowed serverIssueType = 1 + iota
	CacheNotFound
	CacheError
	CacheExpired
)

type serverIssueType int
