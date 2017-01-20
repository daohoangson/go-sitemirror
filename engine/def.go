package engine

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	"github.com/daohoangson/go-sitemirror/web"
)

// Engine represents an object that can mirror urls
type Engine interface {
	init(*http.Client, *logrus.Logger)

	GetCacher() cacher.Cacher
	GetCrawler() crawler.Crawler
	GetServer() web.Server

	AddHostRewrite(string, string)
	GetHostRewrites() map[string]string
	AddHostWhitelisted(string)
	GetHostsWhitelist() []string
	SetBumpTTL(time.Duration)
	GetBumpTTL() time.Duration
	SetAutoEnqueueInterval(time.Duration)
	GetAutoEnqueueInterval() time.Duration

	Mirror(*url.URL, int) error
	Stop()
}

var (
	// ResponseBodyMethodNotAllowed the text to respond when user request method is not allowed
	ResponseBodyMethodNotAllowed = "Sorry, your request is not supported and cannot be processed."
)
