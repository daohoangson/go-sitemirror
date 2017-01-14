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
			BaseURL:         baseUrl,
			LinksAssets:     make(map[string]Link),
			LinksDiscovered: make(map[string]Link),
		}
	})

	Describe("GetAssetURLs", func() {
		It("should return empty slice", func() {
			urls := downloaded.GetAssetURLs()
			Expect(len(urls)).To(Equal(0))
		})

		It("should resolve relative url", func() {
			linkUrl, _ := neturl.Parse("relative/assets")
			link := Link{URL: linkUrl}
			downloaded.LinksAssets[linkUrl.String()] = link

			urls := downloaded.GetAssetURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0].String()).To(Equal("http://domain.com/downloaded/relative/assets"))
		})
	})

	Describe("GetDiscoveredURLs", func() {
		It("should return empty slice", func() {
			urls := downloaded.GetDiscoveredURLs()
			Expect(len(urls)).To(Equal(0))
		})

		It("should resolve relative url", func() {
			linkUrl, _ := neturl.Parse("relative/discovered")
			link := Link{URL: linkUrl}
			downloaded.LinksDiscovered[linkUrl.String()] = link

			urls := downloaded.GetDiscoveredURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0].String()).To(Equal("http://domain.com/downloaded/relative/discovered"))
		})
	})
})
