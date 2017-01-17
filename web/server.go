package web

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
)

type server struct {
	cacher cacher.Cacher
	logger *logrus.Logger

	onCacheIssue *func(CacheIssue)

	listeners map[string]net.Listener
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

	s.listeners = make(map[string]net.Listener)
}

func (s *server) GetCacher() cacher.Cacher {
	return s.cacher
}

func (s *server) SetOnCacheIssue(f func(CacheIssue)) {
	s.onCacheIssue = &f
}

func (s *server) ListenAndServe(host string, port int) (io.Closer, error) {
	if port < 0 {
		return nil, errors.New("Invalid port")
	}

	if existing, existingFound := s.listeners[host]; existingFound {
		return existing, errors.New("Existing listener has been found for this host")
	}

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

	start := time.Now()
	go func() {
		var f http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
			s.Serve(host, w, req)
		}

		loggerContext.Info("Serving")
		serveError := http.Serve(listener, f)
		if serveError != nil {
			elapsed := time.Since(start)
			errorContext := loggerContext.WithField("error", serveError)
			if elapsed > 100*time.Millisecond {
				// some time has passed, it's likely that it worked
				// but the listener has been asked to be closed
				errorContext.Debug("Listener has been closed")
			} else {
				errorContext.Errorf("Cannot serve")
			}
		}
	}()

	s.listeners[host] = listener
	return listener, nil
}

func (s *server) GetListeningPort(host string) (int, error) {
	listener, ok := s.listeners[host]
	if !ok {
		return 0, errors.New("Listener not found")
	}

	addr := listener.Addr().String()
	matches := regexp.MustCompile(`:(\d+)$`).FindStringSubmatch(addr)
	port, err := strconv.ParseInt(matches[1], 10, 64)

	return int(port), err
}

func (s *server) Serve(host string, w http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(req.URL.String())
	url.Host = host
	if len(url.Scheme) == 0 {
		url.Scheme = "http"
	}
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

	if info.Expires != nil && info.Expires.Before(time.Now()) {
		loggerContext = loggerContext.WithField("expired", info.Expires)
		s.triggerOnCacheIssue(CacheExpired, url, info)
	}

	loggerContext.WithField("statusCode", info.StatusCode).Debug("Served")
}

func (s *server) StopListening(host string) error {
	listener, ok := s.listeners[host]
	if !ok {
		return errors.New("Listener not found")
	}

	err := listener.Close()

	loggerContext := s.logger.WithField("host", host)
	if err == nil {
		loggerContext.Info("Stopped listening")
	} else {
		loggerContext.WithField("error", err).Error("Cannot stop listening")
	}

	return err
}

func (s *server) StopAll() []string {
	hosts := make([]string, 0)

	for host, listener := range s.listeners {
		err := listener.Close()
		loggerContext := s.logger.WithField("host", host)

		if err == nil {
			loggerContext.Info("Stopped listening")
			hosts = append(hosts, host)
		} else {
			loggerContext.WithField("error", err).Error("Cannot stop listening")
		}
	}

	return hosts
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
