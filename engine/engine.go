package engine

import (
	"net/http"
	neturl "net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	"github.com/daohoangson/go-sitemirror/web"
	"github.com/tevino/abool"
)

type engine struct {
	logger *logrus.Logger

	cacher  cacher.Cacher
	crawler crawler.Crawler
	server  web.Server

	hostRewrites   map[string]string
	hostsWhitelist []string
	bumpTTL        time.Duration

	stopped             *abool.AtomicBool
	downloadedSomething chan interface{}
}

func New(httpClient *http.Client, logger *logrus.Logger) Engine {
	e := &engine{}
	e.init(httpClient, logger)
	return e
}

func (e *engine) init(httpClient *http.Client, logger *logrus.Logger) {
	if logger == nil {
		logger = logrus.New()
	}
	e.logger = logger

	e.cacher = cacher.NewHttpCacher(logger)
	e.crawler = crawler.New(httpClient, logger)
	e.server = web.NewServer(e.cacher, logger)

	e.bumpTTL = time.Minute

	e.stopped = abool.New()
	e.downloadedSomething = make(chan interface{})

	e.crawler.SetURLRewriter(func(u *neturl.URL) {
		e.rewriteURLHost(u)
	})

	e.crawler.SetOnURLShouldQueue(func(u *neturl.URL) bool {
		if !e.checkHostWhitelisted(u.Host) {
			e.logger.WithFields(logrus.Fields{
				"host": u.Host,
				"list": e.hostsWhitelist,
			}).Info("Host is not whitelisted")
			return false
		}

		return true
	})

	e.crawler.SetOnURLShouldDownload(func(u *neturl.URL) bool {
		if e.cacher.CheckCacheExists(u) {
			e.logger.WithField("url", u).Info("Cache exists for url")
			return false
		}

		return true
	})

	e.crawler.SetOnDownloaded(func(downloaded *crawler.Downloaded) {
		input := BuildCacherInputFromCrawlerDownloaded(downloaded)
		e.cacher.Write(input)

		if !e.stopped.IsSet() {
			select {
			case e.downloadedSomething <- true:
			default:
			}
		}
	})

	e.server.SetOnServerIssue(func(issue *web.ServerIssue) {
		switch issue.Type {
		case web.MethodNotAllowed:
			issue.Info.WriteBody([]byte(ResponseBodyMethodNotAllowed))
		case web.CacheNotFound:
			e.cacher.WritePlaceholder(issue.URL, e.bumpTTL)

			downloaded := e.crawler.Download(crawler.QueueItem{
				URL:           issue.URL,
				ForceDownload: true,
			})

			web.ServeDownloaded(downloaded, issue.Info)
		case web.CacheError:
			e.cacher.WritePlaceholder(issue.URL, e.bumpTTL)
			e.crawler.Enqueue(crawler.QueueItem{URL: issue.URL})
		case web.CacheExpired:
			e.cacher.Bump(issue.URL, e.bumpTTL)
			e.crawler.Enqueue(crawler.QueueItem{
				URL:           issue.URL,
				ForceDownload: true,
			})
		}
	})
}

func (e *engine) GetCacher() cacher.Cacher {
	return e.cacher
}

func (e *engine) GetCrawler() crawler.Crawler {
	return e.crawler
}

func (e *engine) GetServer() web.Server {
	return e.server
}

func (e *engine) AddHostRewrite(from string, to string) {
	if e.hostRewrites == nil {
		e.hostRewrites = make(map[string]string)
	}

	e.hostRewrites[from] = to
	e.logger.WithFields(logrus.Fields{
		"from":    from,
		"to":      to,
		"mapping": e.hostRewrites,
	}).Info("Added host rewrite")
}

func (e *engine) AddHostWhitelisted(host string) {
	if e.hostsWhitelist == nil {
		e.hostsWhitelist = make([]string, 1)
		e.hostsWhitelist[0] = host
	} else {
		for _, hostWhitelist := range e.hostsWhitelist {
			if hostWhitelist == host {
				e.logger.WithFields(logrus.Fields{
					"host": host,
					"list": e.hostsWhitelist,
				}).Debug("Cannot add host: already in whitelist")

				return
			}
		}

		e.hostsWhitelist = append(e.hostsWhitelist, host)
	}

	e.logger.WithFields(logrus.Fields{
		"host": host,
		"list": e.hostsWhitelist,
	}).Info("Added host into whitelist")
}

func (e *engine) SetBumpTTL(ttl time.Duration) {
	e.bumpTTL = ttl
}

func (e *engine) Mirror(url *neturl.URL, port int) error {
	e.crawler.Enqueue(crawler.QueueItem{URL: url})

	if port < 0 {
		return nil
	}

	_, err := e.server.ListenAndServe(url, port)

	return err
}

func (e *engine) MirrorURL(url string, port int) error {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		e.logger.WithField("url", url).Error("Cannot mirror invalid url")

		return err
	}

	return e.Mirror(parsedURL, port)
}

func (e *engine) Stop() {
	if e.stopped.IsSet() {
		return
	}

	for {
		if !e.crawler.IsBusy() {
			e.cleanUp()
			break
		}
		select {
		case <-e.downloadedSomething:
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (e *engine) rewriteURLHost(url *neturl.URL) {
	if e.hostRewrites == nil {
		return
	}

	for from, to := range e.hostRewrites {
		if url.Host == from {
			urlBefore := url.String()
			url.Host = to

			e.logger.WithFields(logrus.Fields{
				"before": urlBefore,
				"after":  url.String(),
			}).Debug("Rewritten url host")

			return
		}
	}
}

func (e *engine) checkHostWhitelisted(host string) bool {
	if e.hostsWhitelist == nil {
		return true
	}

	for _, hostWhitelist := range e.hostsWhitelist {
		if host == hostWhitelist {
			return true
		}
	}

	return false
}

func (e *engine) cleanUp() {
	stoppedAtomicChange := e.stopped.SetToIf(false, true)
	if stoppedAtomicChange {
		e.crawler.Stop()
		e.server.Stop()

		close(e.downloadedSomething)
	}
}
