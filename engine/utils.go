package engine

import (
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
)

func BuildCacherInputFromCrawlerDownloaded(d *crawler.Downloaded) *cacher.Input {
	i := &cacher.Input{}

	if d.StatusCode > 0 {
		i.StatusCode = d.StatusCode
	}

	if d.URL != nil {
		i.URL = d.URL
	}

	if len(d.ContentType) > 0 {
		i.ContentType = d.ContentType
	}

	if len(d.BodyString) > 0 {
		i.Body = d.BodyString
	} else if d.BodyBytes != nil {
		i.Body = string(d.BodyBytes)
	}

	if d.HeaderLocation != nil {
		i.Redirection = d.HeaderLocation
	}

	return i
}
