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
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/web/internal"
)

type server struct {
	cacher cacher.Cacher
	logger *logrus.Logger

	onServerIssue *func(*ServerIssue)

	mutex     sync.Mutex
	listeners map[string]net.Listener
}

type listenerCloser struct {
	server *server
	host   string
}

var (
	regexpCrossHostPath = regexp.MustCompile(`^/(https?)/([^/]+)(/.*)?$`)
)

// NewServer returns a new server intance
func NewServer(cacher cacher.Cacher, logger *logrus.Logger) Server {
	s := &server{}
	s.init(cacher, logger)
	return s
}

func (s *server) init(httpCacher cacher.Cacher, logger *logrus.Logger) {
	if httpCacher == nil {
		httpCacher = cacher.NewHTTPCacher(nil, nil)
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

func (s *server) SetOnServerIssue(f func(*ServerIssue)) {
	s.onServerIssue = &f
}

func (s *server) ListenAndServe(root *url.URL, port int) (io.Closer, error) {
	if port < 0 {
		return nil, errors.New("Invalid port")
	}

	var host string
	if root != nil {
		host = root.Host
	}
	loggerContext := s.logger.WithFields(logrus.Fields{
		"root": root,
		"port": port,
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, existingFound := s.listeners[host]; existingFound {
		return nil, errors.New("Existing listener has been found for this host")
	}

	listener, listenError := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if listenError != nil {
		loggerContext.WithField("error", listenError).Errorf("Cannot listen")
		return nil, listenError
	}
	s.listeners[host] = listener

	if port == 0 {
		loggerContext = loggerContext.WithField("addr", listener.Addr().String())
	}

	s.setupListener(listener, host, root)

	closer := &listenerCloser{server: s, host: host}
	loggerContext.Info("Listening...")

	return closer, nil
}

func (s *server) GetListeningPort(host string) (int, error) {
	s.mutex.Lock()
	listener, ok := s.listeners[host]
	s.mutex.Unlock()

	if !ok {
		return 0, errors.New("Listener not found")
	}

	addr := listener.Addr().String()
	matches := regexp.MustCompile(`:(\d+)$`).FindStringSubmatch(addr)
	port, err := strconv.ParseInt(matches[1], 10, 64)

	return int(port), err
}

func (s *server) Serve(root *url.URL, w http.ResponseWriter, req *http.Request) internal.ServeInfo {
	if root != nil {
		return s.serveWithRoot(root.Scheme, root.Host, w, req)
	}

	return s.serveCrossHost(w, req)
}

func (s *server) Stop() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	hosts := make([]string, 0)

	i := 0
	l := len(s.listeners)
	for host, listener := range s.listeners {
		i++
		loggerContext := s.logger.WithFields(logrus.Fields{
			"host":  host,
			"index": i,
			"total": l,
		})
		loggerContext.Debug("Closing listener...")

		err := listener.Close()

		if err == nil {
			loggerContext.Info("Closed listener")
			hosts = append(hosts, host)
		} else {
			loggerContext.WithError(err).Error("Cannot close listener")
		}

		delete(s.listeners, host)
	}

	return hosts
}

func (s *server) setupListener(listener net.Listener, host string, root *url.URL) {

	go func() {
		var f http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
			s.Serve(root, w, req)
		}

		serveError := http.Serve(listener, f)
		if serveError != nil {
			loggerContext := s.logger.WithFields(logrus.Fields{
				"host":  host,
				"error": serveError,
			})

			s.mutex.Lock()
			_, found := s.listeners[host]
			s.mutex.Unlock()

			if !found {
				// no listener record found, probably closed properly
				loggerContext.Debug("Listener has been closed")
			} else {
				loggerContext.Error("Cannot serve")
			}
		}
	}()
}

func (s *server) serveWithRoot(scheme string, host string, w http.ResponseWriter, req *http.Request) internal.ServeInfo {
	si := internal.NewServeInfo(false, w)

	targetURL, _ := url.Parse(req.URL.String())
	targetURL.Scheme = scheme
	targetURL.Host = host

	return s.serveURL(targetURL, si, req)
}

func (s *server) serveCrossHost(w http.ResponseWriter, req *http.Request) internal.ServeInfo {
	si := internal.NewServeInfo(true, w)
	targetURL, _ := url.Parse(req.URL.String())

	matches := regexpCrossHostPath.FindStringSubmatch(targetURL.Path)
	if matches == nil {
		return s.serveServerIssue(&ServerIssue{
			Type: CrossHostInvalidPath,
			URL:  targetURL,
			Info: si.OnCrossHostInvalidPath(),
		})
	}

	targetURL.Scheme = matches[1]
	targetURL.Host = matches[2]
	targetURL.Path = matches[3]

	if len(targetURL.Path) == 0 {
		// relative urls do not work correctly if user is on http://localhost/https/domain.com
		// so we will take care of it here and redirect to ./
		si.SetStatusCode(http.StatusMovedPermanently)
		si.AddHeader(cacher.HeaderLocation, fmt.Sprintf("/%s/%s/", targetURL.Scheme, targetURL.Host))
		return si.Flush()
	}

	return s.serveURL(targetURL, si, req)
}

func (s *server) serveURL(url *url.URL, si internal.ServeInfo, req *http.Request) internal.ServeInfo {
	if len(url.Scheme) == 0 {
		url.Scheme = cacher.SchemeDefault
	}

	if len(req.Method) > 0 && req.Method != "GET" {
		return s.serveServerIssue(&ServerIssue{
			Type: MethodNotAllowed,
			URL:  url,
			Info: si.OnMethodNotAllowed(),
		})
	}

	if url.Path == "/robots.txt" {
		return s.serveRobotsTxt(si)
	}

	cache, err := s.cacher.Open(url)
	if err != nil {
		return s.serveServerIssue(&ServerIssue{
			Type: CacheNotFound,
			URL:  url,
			Info: si.OnCacheNotFound(err),
		})
	}
	defer cache.Close()

	ServeHTTPCache(cache, si)
	if si.HasError() {
		return s.serveServerIssue(&ServerIssue{
			Type: CacheError,
			URL:  url,
			Info: si,
		})
	}

	loggerContext := s.logger.WithField("url", url)
	siExpires := si.GetExpires()
	if siExpires != nil && siExpires.Before(time.Now()) {
		loggerContext = loggerContext.WithField("expired", siExpires)
		s.triggerOnServerIssue(&ServerIssue{
			Type: CacheExpired,
			URL:  url,
			Info: si,
		})
	}

	loggerContext.WithField("statusCode", si.GetStatusCode()).Debug("Served")
	return si.Flush()
}

func (s *server) serveRobotsTxt(si internal.ServeInfo) internal.ServeInfo {
	si.SetStatusCode(http.StatusOK)
	si.WriteBody([]byte("User-agent: *\nDisallow: /\n"))
	return si.Flush()
}

func (s *server) serveServerIssue(issue *ServerIssue) internal.ServeInfo {
	s.triggerOnServerIssue(issue)
	issue.Info.Flush()

	_, siError := issue.Info.GetError()
	s.logger.WithFields(logrus.Fields{
		"url":        issue.URL,
		"issue":      issue.Type,
		"error":      siError,
		"statusCode": issue.Info.GetStatusCode(),
	}).Debug("Served")

	return issue.Info
}

func (s *server) triggerOnServerIssue(issue *ServerIssue) {
	if s.onServerIssue == nil {
		return
	}

	(*s.onServerIssue)(issue)
}

func (closer *listenerCloser) Close() error {
	loggerContext := closer.server.logger.WithField("host", closer.host)
	loggerContext.Debug("Closing listener...")

	closer.server.mutex.Lock()
	defer closer.server.mutex.Unlock()

	listener, found := closer.server.listeners[closer.host]
	if !found {
		loggerContext.Debug("Listener has already been closed")
		return nil
	}

	delete(closer.server.listeners, closer.host)
	err := listener.Close()

	loggerContext.WithError(err).Info("Closed listener")

	return err
}
