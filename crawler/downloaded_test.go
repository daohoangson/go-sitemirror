package crawler_test

import (
	. "github.com/daohoangson/go-sitemirror/crawler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	neturl "net/url"
)

var _ = Describe("Downloaded", func() {
	baseUrl, _ := neturl.Parse("http://domain.com/downloaded/base")
	var downloaded *Downloaded

	BeforeEach(func() {
		downloaded = &Downloaded{
			BaseURL: baseUrl,
			Links:   make(map[string]Link),
		}
	})

	Describe("GetResolvedURLs", func() {
		It("should return empty slice", func() {
			urls := downloaded.GetResolvedURLs()
			Expect(len(urls)).To(Equal(0))
		})

		It("should resolve relative url", func() {
			linkUrl, _ := neturl.Parse("relative")
			link := Link{URL: linkUrl}
			downloaded.Links[linkUrl.String()] = link

			urls := downloaded.GetResolvedURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0].String()).To(Equal("http://domain.com/downloaded/relative"))
		})

		It("should resolve root relative url", func() {
			linkUrl, _ := neturl.Parse("/root/relative")
			link := Link{URL: linkUrl}
			downloaded.Links[linkUrl.String()] = link

			urls := downloaded.GetResolvedURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0].String()).To(Equal("http://domain.com/root/relative"))
		})

		It("should keep absolute url intact", func() {
			linkUrl, _ := neturl.Parse("http://domain2.com")
			link := Link{URL: linkUrl}
			downloaded.Links[linkUrl.String()] = link

			urls := downloaded.GetResolvedURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0]).To(Equal(linkUrl))
		})
	})
})
