package cacher

import (
	"bufio"
	"io"
	neturl "net/url"
	"os"
	"strings"
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
	loggerContext := c.logger.WithFields(logrus.Fields{
		"url":  url,
		"path": cachePath,
	})

	f, openError := os.Open(cachePath)
	if openError != nil {
		loggerContext.WithError(openError).Debug("Cannot open file -> cache not exists")
		return false
	}
	defer f.Close()

	buffer := make([]byte, len(writeHTTPPlaceholderFirstLine))
	n, readError := f.Read(buffer)
	if readError != nil {
		loggerContext.WithError(readError).Error("Cannot read file -> cache not exists")
		return false
	}
	if n == len(writeHTTPPlaceholderFirstLine) &&
		string(buffer) == writeHTTPPlaceholderFirstLine {
		loggerContext.WithError(openError).Debug("Placeholder first line found -> cache not exists")
		return false
	}

	return true
}

func (c *httpCacher) Write(input *Input) error {
	if input.TTL == 0 {
		input.TTL = c.defaultTTL
	}

	cachePath := GenerateCachePath(c.path, input.URL)
	f, err := CreateFile(cachePath)
	if err != nil {
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

func (c *httpCacher) Bump(url *neturl.URL, ttl time.Duration) error {
	cachePath := GenerateCachePath(c.path, url)
	newExpires := time.Now().Add(ttl)
	loggerContext := c.logger.WithFields(logrus.Fields{
		"url":  url,
		"path": cachePath,
		"time": newExpires,
	})

	f, openError := OpenFile(cachePath)
	if openError != nil {
		return openError
	}
	defer f.Close()

	// try to replace the line
	r := bufio.NewReader(f)
	for {
		line, readError := r.ReadString('\n')
		if readError != nil {
			loggerContext.WithField("error", readError).Error("Cannot read line to bump")
			break
		}

		if line == "\n" {
			// reached end of header without expires line found, fallback to placeholder
			break
		}

		if strings.HasPrefix(line, HTTPHeaderExpires) {
			newLine := writeHTTPFormatExpiresHeader(newExpires)
			if len(newLine) != len(line) {
				loggerContext.WithFields(logrus.Fields{
					"existing": line,
					"new":      newLine,
				}).Error("Cannot bump")
				break
			}

			bytes := []byte(newLine)
			position, _ := f.Seek(0, 1)
			position -= int64(r.Buffered()) + int64(len(bytes))
			f.WriteAt(bytes, position)
			loggerContext.Info("Bumped")
			return nil
		}
	}

	// invalid file or data, just write the placeholder
	f.Truncate(0)
	f.Seek(0, 0)
	writeError := writeHTTPPlaceholder(f, url, newExpires)

	if writeError == nil {
		loggerContext.Info("Written placeholder instead of bump")
	}

	return writeError
}

func (c *httpCacher) WritePlaceholder(url *neturl.URL, ttl time.Duration) error {
	cachePath := GenerateCachePath(c.path, url)
	f, err := CreateFile(cachePath)
	if err != nil {
		return err
	}
	defer f.Close()

	writeError := writeHTTPPlaceholder(f, url, time.Now().Add(ttl))

	if writeError == nil {
		c.logger.WithFields(logrus.Fields{
			"url":  url,
			"path": cachePath,
			"ttl":  ttl,
		}).Info("Written placeholder")
	}

	return writeError
}

func (c *httpCacher) Open(url *neturl.URL) (io.ReadCloser, error) {
	cachePath := GenerateCachePath(c.path, url)
	f, err := os.Open(cachePath)

	if err == nil {
		c.logger.WithFields(logrus.Fields{
			"url":  url,
			"path": cachePath,
		}).Debug("Opened cache")
	}

	return f, err
}
