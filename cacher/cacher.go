package cacher

import (
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
)

type httpCacher struct {
	logger *logrus.Logger

	path string

	defaultTTL time.Duration
}

func NewHttpCacher(logger *logrus.Logger) Cacher {
	c := &httpCacher{}
	c.init(logger)
	return c
}

func (c *httpCacher) init(logger *logrus.Logger) {
	if logger == nil {
		logger = logrus.New()
	}
	c.logger = logger

	if wd, err := os.Getwd(); err == nil {
		c.path = wd
	}

	c.defaultTTL = 10 * time.Minute
}

func (c *httpCacher) GetMode() cacherMode {
	return HttpMode
}

func (c *httpCacher) SetPath(path string) {
	c.logger.WithFields(logrus.Fields{
		"old": c.path,
		"new": path,
	}).Info("Updating cacher path")

	c.path = path
}

func (c *httpCacher) GetPath() string {
	return c.path
}

func (c *httpCacher) SetDefaultTTL(ttl time.Duration) {
	c.defaultTTL = ttl
}

func (c *httpCacher) CheckCacheExists(url *neturl.URL) bool {
	cachePath := GenerateCachePath(c.path, url)
	_, err := os.Stat(cachePath)

	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"url":   url,
			"path":  cachePath,
			"error": err,
		}).Error("Error checking for cache existence")
	}

	return err == nil
}

func (c *httpCacher) Write(input *Input) error {
	if input.TTL == 0 {
		input.TTL = c.defaultTTL
	}

	cachePath := GenerateCachePath(c.path, input.URL)
	f, err := CreateFile(cachePath)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"url":   input.URL,
			"path":  cachePath,
			"error": err,
		}).Error("Cannot write HTTP cache")

		return err
	}
	defer f.Close()

	WriteHTTP(f, input)

	c.logger.WithFields(logrus.Fields{
		"url":  input.URL,
		"path": cachePath,
	}).Debug("Written HTTP cache")

	return nil
}

func (c *httpCacher) WritePlaceholder(url *neturl.URL) error {
	input := &Input{
		URL:        url,
		StatusCode: http.StatusNoContent,
	}
	return c.Write(input)
}

func (c *httpCacher) Open(url *neturl.URL) (io.ReadCloser, error) {
	cachePath := GenerateCachePath(c.path, url)
	f, err := os.Open(cachePath)

	loggerContext := c.logger.WithFields(logrus.Fields{
		"url":  url,
		"path": cachePath,
	})

	if err == nil {
		loggerContext.Debug("Opened cache")
	} else {
		loggerContext.WithField("error", err).Error("Cannot open cache")
	}

	return f, err
}
