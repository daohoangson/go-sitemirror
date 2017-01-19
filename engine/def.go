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

type Engine interface {
	init(*http.Client, *logrus.Logger)

	GetCacher() cacher.Cacher
	GetCrawler() crawler.Crawler
	GetServer() web.Server

	AddHostRewrite(string, string)
	AddHostWhitelisted(string)
	SetBumpTTL(time.Duration)
	SetAutoEnqueueInterval(time.Duration)

	Mirror(*url.URL, int) error
	Stop()
}

var (
	ResponseBodyMethodNotAllowed = "Sorry, your request is not supported and cannot be processed."
)
