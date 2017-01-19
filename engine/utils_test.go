package engine_test

import (
	"net/url"

	"github.com/daohoangson/go-sitemirror/crawler"
	. "github.com/daohoangson/go-sitemirror/engine"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("BuildCacherInputFromCrawlerDownloaded", func() {
		It("should sync status code", func() {
			d := &crawler.Downloaded{StatusCode: 200}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.StatusCode).To(Equal(d.StatusCode))
		})

		It("should sync url", func() {
			url, _ := url.Parse("http://domain.com/engine/utils")
			d := &crawler.Downloaded{Input: &crawler.Input{URL: url}}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.URL).To(Equal(url))
		})

		It("should sync body", func() {
			d := &crawler.Downloaded{Body: "foo/bar"}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Body).To(Equal(d.Body))
		})

		It("should sync header content type", func() {
			headerKey := "Content-Type"
			headerValue := "plain/text"
			d := &crawler.Downloaded{}
			d.AddHeader(headerKey, headerValue)

			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Header.Get(headerKey)).To(Equal(headerValue))
		})

		It("should sync header location", func() {
			headerKey := "Location"
			headerValue := "http://domain.com/engine/utils/sync/header/location"
			d := &crawler.Downloaded{}
			d.AddHeader(headerKey, headerValue)

			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Header.Get(headerKey)).To(Equal(headerValue))
		})
	})
})
