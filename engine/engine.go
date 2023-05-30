package engine

import (
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	"github.com/daohoangson/go-sitemirror/web"
	"github.com/tevino/abool"
)

type engine struct {
	logger *logrus.Logger
	mutex  sync.Mutex

	cacher  cacher.Cacher
	crawler crawler.Crawler
	server  web.Server

	hostRewrites        map[string]engineHostRewrite
	hostsWhitelist      []string
	bumpTTL             time.Duration
	autoEnqueueInterval time.Duration

	autoEnqueueOnce     sync.Once
	autoEnqueueUrls     []*neturl.URL
	autoEnqueueMutex    sync.Mutex
	stopped             *abool.AtomicBool
	downloadedSomething chan interface{}
}

type engineHostRewrite func(*neturl.URL) string

// New returns a new Engine instance
func New(fs cacher.Fs, httpClient *http.Client, logger *logrus.Logger) Engine {
	e := &engine{}
	e.init(fs, httpClient, logger)
	return e
}

func (e *engine) init(fs cacher.Fs, httpClient *http.Client, logger *logrus.Logger) {
	if logger == nil {
		logger = logrus.New()
	}
	e.logger = logger

	e.cacher = cacher.NewHTTPCacher(fs, logger)
	e.crawler = crawler.New(httpClient, logger)
	e.server = web.NewServer(e.cacher, logger)

	e.bumpTTL = time.Minute

	e.stopped = abool.New()
	e.downloadedSomething = make(chan interface{})

	e.crawler.SetURLRewriter(func(u *neturl.URL) {
		e.rewriteURL(u)
	})

	e.crawler.SetOnURLShouldQueue(func(u *neturl.URL) bool {
		if !e.checkHostWhitelisted(u.Host) {
			e.logger.WithFields(logrus.Fields{
				"host": u.Host,
				"list": e.hostsWhitelist,
			}).Debug("Host is not whitelisted")
			return false
		}

		return true
	})

	e.crawler.SetOnURLShouldDownload(func(u *neturl.URL) bool {
		if e.cacher.CheckCacheExists(u) {
			e.logger.WithField("url", u).Debug("Cache exists for url")
			return false
		}

		return true
	})

	e.crawler.SetOnDownloaded(func(downloaded *crawler.Downloaded) {
		if (downloaded.StatusCode == 0 || downloaded.StatusCode >= 500) &&
			e.cacher.CheckCacheExists(downloaded.Input.URL) {
			e.logger.WithFields(logrus.Fields{
				"url":        downloaded.Input.URL,
				"statusCode": downloaded.StatusCode,
			}).Debug("Skipped writing cache")
			return
		}

		input := BuildCacherInputFromCrawlerDownloaded(downloaded)
		cacheError := e.cacher.Write(input)
		if cacheError != nil {
			e.logger.WithFields(logrus.Fields{
				"url":        downloaded.Input.URL,
				"statusCode": downloaded.StatusCode,
				"cacheError": cacheError,
			}).Error("Failed to write cache")
		}

		e.mutex.Lock()
		if !e.stopped.IsSet() {
			select {
			case e.downloadedSomething <- true:
			default:
			}
		}
		e.mutex.Unlock()
	})

	downloadAndServe := func(issue *web.ServerIssue) {
		placeholderError := e.cacher.WritePlaceholder(issue.URL, e.bumpTTL)
		if placeholderError != nil {
			e.logger.WithFields(logrus.Fields{
				"url":              issue.URL,
				"placeholderError": placeholderError,
			}).Error("Failed to write placeholder")
		} else {
			downloaded := e.crawler.Download(crawler.QueueItem{
				URL:           issue.URL,
				ForceDownload: true,
			})
			web.ServeDownloaded(downloaded, issue.Info)
		}
	}
	e.server.SetOnServerIssue(func(issue *web.ServerIssue) {
		switch issue.Type {
		case web.MethodNotAllowed:
			issue.Info.WriteBody([]byte(ResponseBodyMethodNotAllowed))
		case web.CacheNotFound:
			downloadAndServe(issue)
		case web.CacheError:
			downloadAndServe(issue)
		case web.CacheExpired:
			_ = e.cacher.Bump(issue.URL, e.bumpTTL)
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
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.hostRewrites == nil {
		e.hostRewrites = make(map[string]engineHostRewrite)
	}

	var parsedTo *neturl.URL
	if strings.HasPrefix(to, "http") {
		parsedTo, _ = neturl.Parse(to)
	}

	e.hostRewrites[from] = func(url *neturl.URL) string {
		if url != nil {
			if parsedTo != nil {
				url.Scheme = parsedTo.Scheme
				url.Host = parsedTo.Host
				url.Path = strings.TrimRight(parsedTo.Path, "/") + url.Path
			} else {
				url.Host = to
			}
		}

		return to
	}

	e.logger.WithFields(logrus.Fields{
		"from":     from,
		"to":       to,
		"mappings": len(e.hostRewrites),
	}).Info("Added host rewrite")
}

func (e *engine) GetHostRewrites() map[string]string {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	hostRewrites := make(map[string]string)

	if e.hostRewrites != nil {
		for from, f := range e.hostRewrites {
			hostRewrites[from] = f(nil)
		}
	}

	return hostRewrites
}

func (e *engine) AddHostWhitelisted(host string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

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

func (e *engine) GetHostsWhitelist() []string {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	hostsWhitelist := make([]string, len(e.hostsWhitelist))
	if e.hostsWhitelist != nil {
		i := 0
		for _, host := range e.hostsWhitelist {
			hostsWhitelist[i] = host
		}
	}

	return hostsWhitelist
}

func (e *engine) SetBumpTTL(ttl time.Duration) {
	e.mutex.Lock()
	e.bumpTTL = ttl
	e.mutex.Unlock()
}

func (e *engine) GetBumpTTL() time.Duration {
	e.mutex.Lock()
	ttl := e.bumpTTL
	e.mutex.Unlock()

	return ttl
}

func (e *engine) SetAutoEnqueueInterval(interval time.Duration) {
	e.mutex.Lock()
	e.autoEnqueueInterval = interval
	e.mutex.Unlock()
}

func (e *engine) GetAutoEnqueueInterval() time.Duration {
	e.mutex.Lock()
	interval := e.autoEnqueueInterval
	e.mutex.Unlock()

	return interval
}

func (e *engine) Mirror(url *neturl.URL, port int) error {
	var root *neturl.URL

	if url != nil {
		root, _ = neturl.Parse(url.String())
		if len(root.Path) == 0 {
			root.Path = "/"
		}

		e.autoEnqueue(root)
		e.crawler.Enqueue(crawler.QueueItem{URL: root})
	}

	if port < 0 {
		return nil
	}

	_, err := e.server.ListenAndServe(root, port)

	loggerContext := e.logger.WithFields(logrus.Fields{
		"url":  url,
		"port": port,
		"root": root,
	})
	if err != nil {
		loggerContext.Error("Mirror cannot be setup")
	} else {
		loggerContext.Info("Mirror is up")
	}

	return err
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

func (e *engine) autoEnqueue(url *neturl.URL) {
	e.autoEnqueueMutex.Lock()
	interval := e.autoEnqueueInterval
	e.autoEnqueueMutex.Unlock()

	if interval == 0 {
		e.logger.Debug("Engine.autoEnqueue skipped")
		return
	}

	e.autoEnqueueMutex.Lock()
	if e.autoEnqueueUrls == nil {
		e.autoEnqueueUrls = []*neturl.URL{url}
	} else {
		e.autoEnqueueUrls = append(e.autoEnqueueUrls, url)
	}
	e.autoEnqueueMutex.Unlock()

	e.autoEnqueueOnce.Do(func() {
		go func() {
			for {
				<-time.After(interval)

				if e.stopped.IsSet() {
					e.logger.Info("Engine.autoEnqueue stopped")
					return
				}

				e.autoEnqueueMutex.Lock()
				for _, url := range e.autoEnqueueUrls {
					e.GetCrawler().Enqueue(crawler.QueueItem{
						URL:           url,
						ForceDownload: true,
					})
					e.logger.WithField("url", url).Debug("Engine.autoEnqueue enqueued")
				}
				e.autoEnqueueMutex.Unlock()
			}
		}()
	})
}

func (e *engine) rewriteURL(url *neturl.URL) {
	e.mutex.Lock()
	hostRewrites := e.hostRewrites
	e.mutex.Unlock()

	if hostRewrites == nil {
		return
	}

	for from, f := range hostRewrites {
		if url.Host == from {
			urlBefore := url.String()
			f(url)

			e.logger.WithFields(logrus.Fields{
				"before": urlBefore,
				"after":  url.String(),
			}).Debug("Rewritten url")

			return
		}
	}
}

func (e *engine) checkHostWhitelisted(host string) bool {
	e.mutex.Lock()
	hostsWhitelist := e.hostsWhitelist
	e.mutex.Unlock()

	if hostsWhitelist == nil {
		return true
	}

	for _, hostWhitelist := range hostsWhitelist {
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

		e.mutex.Lock()
		close(e.downloadedSomething)
		e.mutex.Unlock()
	}
}
