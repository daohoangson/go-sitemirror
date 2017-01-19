package engine

import (
	"net/http"

	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
)

func BuildCacherInputFromCrawlerDownloaded(d *crawler.Downloaded) *cacher.Input {
	i := &cacher.Input{}

	if d.StatusCode > 0 {
		i.StatusCode = d.StatusCode
	}

	if d.Input != nil && d.Input.URL != nil {
		i.URL = d.Input.URL
	}

	i.Body = d.Body

	i.Header = make(http.Header)
	for _, headerKey := range d.GetHeaderKeys() {
		for _, headerValue := range d.GetHeaderValues(headerKey) {
			i.Header.Add(headerKey, headerValue)
		}
	}

	return i
}
