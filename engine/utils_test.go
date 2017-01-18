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
			d := &crawler.Downloaded{URL: url}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.URL).To(Equal(url))
		})

		It("should sync content type", func() {
			d := &crawler.Downloaded{ContentType: "text/html"}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.ContentType).To(Equal(d.ContentType))
		})

		It("should sync body string", func() {
			d := &crawler.Downloaded{BodyString: "foo/bar"}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Body).To(Equal(d.BodyString))
		})

		It("should sync body bytes", func() {
			d := &crawler.Downloaded{BodyBytes: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Body).To(Equal(string(d.BodyBytes)))
		})

		It("should sync body string, ignoring body bytes", func() {
			d := &crawler.Downloaded{
				BodyString: "foo/bar",
				BodyBytes:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
			}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Body).To(Equal(d.BodyString))
		})

		It("should sync header location", func() {
			targetUrl, _ := url.Parse("http://domain.com/engine/utils/target/url")
			d := &crawler.Downloaded{HeaderLocation: targetUrl}
			i := BuildCacherInputFromCrawlerDownloaded(d)
			Expect(i.Redirection).To(Equal(targetUrl))
		})
	})
})
