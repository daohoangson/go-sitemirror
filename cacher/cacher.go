package cacher

import (
	"errors"
	neturl "net/url"
	"os"
)

type cacher struct {
	mode cacherMode
	path string
}

func NewHttpCacher() Cacher {
	c := &cacher{}
	c.init(httpMode)
	return c
}

func (c *cacher) init(mode cacherMode) {
	c.mode = mode

	if wd, err := os.Getwd(); err == nil {
		c.path = wd
	}
}

func (c *cacher) SetPath(path string) {
	c.path = path
}

func (c *cacher) GetPath() string {
	return c.path
}

func (c *cacher) CheckCacheExists(url *neturl.URL) bool {
	cachePath := GenerateCachePath(c.path, url)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func (c *cacher) Write(input *Input) error {
	cachePath := GenerateCachePath(c.path, input.URL)
	f, err := CreateFile(cachePath)
	if err != nil {
		return err
	}
	defer f.Close()

	switch c.mode {
	case httpMode:
		return WriteHTTP(f, input)
	}

	return errors.New("Unknown mode")
}
