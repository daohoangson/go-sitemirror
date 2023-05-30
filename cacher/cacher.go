package cacher

import (
	"bufio"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type httpCacher struct {
	fs     Fs
	logger *logrus.Logger
	mutex  sync.Mutex

	path string

	defaultTTL time.Duration
}

// NewHTTPCacher returns a new http cacher instance
func NewHTTPCacher(fs Fs, logger *logrus.Logger) Cacher {
	c := &httpCacher{}
	c.init(fs, logger)
	return c
}

func (c *httpCacher) init(fs Fs, logger *logrus.Logger) {
	if fs == nil {
		fs = NewFs()
	}
	c.fs = fs

	if logger == nil {
		logger = logrus.New()
	}
	c.logger = logger

	if wd, err := fs.Getwd(); err == nil {
		c.path = wd
	}

	c.defaultTTL = 10 * time.Minute
}

func (c *httpCacher) GetMode() cacherMode {
	return HTTPMode
}

func (c *httpCacher) SetPath(path string) {
	c.mutex.Lock()
	old := c.path
	c.path = path
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"old": old,
		"new": path,
	}).Info("Updated cacher path")
}

func (c *httpCacher) GetPath() string {
	c.mutex.Lock()
	path := c.path
	c.mutex.Unlock()

	return path
}

func (c *httpCacher) SetDefaultTTL(ttl time.Duration) {
	c.mutex.Lock()
	old := c.defaultTTL
	c.defaultTTL = ttl
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"old": old,
		"new": ttl,
	}).Info("Updated cacher default ttl")
}

func (c *httpCacher) GetDefaultTTL() time.Duration {
	c.mutex.Lock()
	ttl := c.defaultTTL
	c.mutex.Unlock()

	return ttl
}

func (c *httpCacher) CheckCacheExists(url *neturl.URL) bool {
	c.mutex.Lock()
	fs := c.fs
	c.mutex.Unlock()

	cachePath := c.generateCachePath(url)
	loggerContext := c.logger.WithFields(logrus.Fields{
		"url":  url,
		"path": cachePath,
	})

	f, openError := fs.OpenFile(cachePath, os.O_RDONLY, 0)
	if openError != nil {
		loggerContext.WithError(openError).Debug("Cannot open file -> cache not exists")
		return false
	}
	defer func() { _ = f.Close() }()

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
	c.mutex.Lock()
	if input.TTL == 0 {
		input.TTL = c.defaultTTL
	}
	fs := c.fs
	c.mutex.Unlock()

	cachePath := c.generateCachePath(input.URL)
	f, err := CreateFile(fs, cachePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	writeError := WriteHTTP(f, input)
	if writeError != nil {
		return fmt.Errorf("WriteHTTP: %s", writeError)
	}

	c.logger.WithFields(logrus.Fields{
		"url":  input.URL,
		"path": cachePath,
	}).Debug("Written HTTP cache")

	return nil
}

func (c *httpCacher) Bump(url *neturl.URL, ttl time.Duration) error {
	c.mutex.Lock()
	fs := c.fs
	c.mutex.Unlock()

	cachePath := c.generateCachePath(url)
	newExpires := time.Now().Add(ttl)
	loggerContext := c.logger.WithFields(logrus.Fields{
		"url":  url,
		"path": cachePath,
		"time": newExpires,
	})

	f, openError := OpenFile(fs, cachePath)
	if openError != nil {
		return openError
	}
	defer func() { _ = f.Close() }()

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

		if strings.HasPrefix(line, CustomHeaderExpires) {
			newLine := formatExpiresHeader(newExpires)
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
			_, writeError := f.WriteAt(bytes, position)
			if writeError != nil {
				return writeError
			}

			loggerContext.Info("Bumped")
			return nil
		}
	}

	// invalid file or data, just write the placeholder
	truncateError := f.Truncate(0)
	if truncateError != nil {
		return fmt.Errorf("f.Truncate(0): %w", truncateError)
	}
	_, seekError := f.Seek(0, 0)
	if seekError != nil {
		return fmt.Errorf("f.Seek(0, 0): %w", seekError)
	}
	writeError := writeHTTPPlaceholder(f, url, newExpires)

	if writeError == nil {
		loggerContext.Info("Written placeholder instead of bump")
	}

	return writeError
}

func (c *httpCacher) WritePlaceholder(url *neturl.URL, ttl time.Duration) error {
	c.mutex.Lock()
	fs := c.fs
	c.mutex.Unlock()

	cachePath := c.generateCachePath(url)
	f, err := CreateFile(fs, cachePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

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
	c.mutex.Lock()
	fs := c.fs
	c.mutex.Unlock()

	cachePath := c.generateCachePath(url)
	f, err := fs.OpenFile(cachePath, os.O_RDONLY, 0)

	if err == nil {
		c.logger.WithFields(logrus.Fields{
			"url":  url,
			"path": cachePath,
		}).Debug("Opened cache")
	}

	return f, err
}

func (c *httpCacher) generateCachePath(url *neturl.URL) string {
	c.mutex.Lock()
	path := c.path
	c.mutex.Unlock()

	return GenerateHTTPCachePath(path, url)
}
