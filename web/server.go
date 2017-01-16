package web

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
)

type server struct {
	cacher cacher.Cacher
	logger *logrus.Logger

	onCacheIssue *func(CacheIssue)
}

func NewServer(cacher cacher.Cacher, logger *logrus.Logger) Server {
	s := &server{}
	s.init(cacher, logger)
	return s
}

func (s *server) init(httpCacher cacher.Cacher, logger *logrus.Logger) {
	if httpCacher == nil {
		httpCacher = cacher.NewHttpCacher(nil)
	}
	s.cacher = httpCacher

	if logger == nil {
		logger = logrus.New()
	}
	s.logger = logger
}

func (s *server) GetCacher() cacher.Cacher {
	return s.cacher
}

func (s *server) SetOnCacheIssue(f func(CacheIssue)) {
	s.onCacheIssue = &f
}

func (s *server) ListenAndServe(host string, port int) (net.Listener, error) {
	loggerContext := s.logger.WithFields(logrus.Fields{
		"host": host,
		"port": port,
	})

	listener, listenError := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if listenError != nil {
		loggerContext.WithField("error", listenError).Errorf("Cannot listen")
		return nil, listenError
	}

	if port == 0 {
		loggerContext = loggerContext.WithField("addr", listener.Addr().String())
	}

	go func() {
		var f http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
			s.Serve(host, w, req)
		}

		loggerContext.Info("Serving")
		serveError := http.Serve(listener, f)
		if serveError != nil {
			loggerContext.WithField("error", serveError).Errorf("Cannot serve")
		}
	}()

	return listener, nil
}

func (s *server) Serve(host string, w http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(req.URL.String())
	url.Host = host
	loggerContext := s.logger.WithField("url", url)

	cache, err := s.cacher.Open(url)
	if err != nil {
		loggerContext.Debug("Cache not found")

		infoOnError := &CacheInfo{
			ResponseWriter: w,
			Error:          err,
		}
		if !s.triggerOnCacheIssue(CacheNotFound, url, infoOnError) {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	}
	defer cache.Close()

	info := ServeHTTPCache(cache, w)
	if info.Error != nil {
		loggerContext.WithField("error", info.Error).Error("Cannot serve")
		s.triggerOnCacheIssue(CacheError, url, info)
		return
	}

	loggerContext.WithField("statusCode", info.StatusCode).Debug("Served")
	if info.Expires != nil && info.Expires.Before(time.Now()) {
		s.triggerOnCacheIssue(CacheExpired, url, info)
	}
}

func (s *server) triggerOnCacheIssue(t cacheIssueType, url *url.URL, info *CacheInfo) bool {
	if s.onCacheIssue == nil {
		return false
	}

	(*s.onCacheIssue)(CacheIssue{
		Type: t,
		URL:  url,
		Info: info,
	})

	return true
}
