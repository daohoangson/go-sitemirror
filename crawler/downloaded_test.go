package crawler_test

import (
	"fmt"
	"sort"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	. "github.com/daohoangson/go-sitemirror/crawler"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	neturl "net/url"
)

var _ = Describe("Downloaded", func() {
	baseUrl, _ := neturl.Parse("https://domain.com/downloaded/base")
	var downloaded *Downloaded

	BeforeEach(func() {
		downloaded = &Downloaded{
			BaseURL:         baseUrl,
			LinksAssets:     make(map[string]Link),
			LinksDiscovered: make(map[string]Link),
		}
	})

	Describe("Header", func() {
		var (
			headerKey  string
			headerVal1 string
			headerVal2 string
		)

		BeforeEach(func() {
			now := time.Now()
			headerKey = "Now"
			headerVal1 = fmt.Sprintf("%s", now)
			headerVal2 = fmt.Sprintf("%d", now.Unix())
		})

		It("should add", func() {
			downloaded.AddHeader(headerKey, headerVal1)
			downloaded.AddHeader(headerKey, headerVal2)

			Expect(downloaded.GetHeaderValues(headerKey)).To(Equal([]string{
				headerVal1,
				headerVal2,
			}))
		})

		It("should return key", func() {
			downloaded.AddHeader(headerKey, headerVal1)

			Expect(downloaded.GetHeaderKeys()).To(Equal([]string{
				headerKey,
			}))
		})

		It("should return all keys", func() {
			headerKey2 := headerKey + "2"
			downloaded.AddHeader(headerKey, headerVal1)
			downloaded.AddHeader(headerKey2, headerVal2)

			keys := downloaded.GetHeaderKeys()
			sort.Strings(keys)
			Expect(keys).To(Equal([]string{
				headerKey,
				headerKey2,
			}))
		})

		It("should return no keys", func() {
			Expect(downloaded.GetHeaderKeys()).To(BeNil())
		})

		It("should return on no header", func() {
			Expect(downloaded.GetHeaderValues(headerKey)).To(BeNil())
		})

		It("should return no values", func() {
			downloaded.AddHeader("Other", "Value")

			Expect(downloaded.GetHeaderValues(headerKey)).To(BeNil())
		})
	})

	Describe("ProcessURL", func() {
		BeforeEach(func() {
			defaultURL := "https://domain.com/ProcessURL/default"
			parsedURL, _ := neturl.Parse(defaultURL)
			downloaded.Input = &Input{URL: parsedURL}
		})

		It("should rewrite url", func() {
			urlPath := "/ProcessURL/rewriter"
			url := "https://domain2.com" + urlPath
			parsedURL, _ := neturl.Parse("https://domain.com")
			rewriter := func(url *neturl.URL) {
				url.Host = "domain.com"
			}
			downloaded.Input = &Input{URL: parsedURL, Rewriter: &rewriter}

			processedURL, _ := downloaded.ProcessURL(HTMLTagA, url)

			Expect(processedURL).To(Equal("." + urlPath))
		})

		It("should keep non-http url intact", func() {
			url := "ftp://domain.com/ProcessURL/non/http"
			processedURL, _ := downloaded.ProcessURL(HTMLTagA, url)

			Expect(processedURL).To(Equal(url))
		})

		It("should remove #fragment", func() {
			fragment := "#foo=bar"
			url := "https://domain.com/ProcessURL/remove/fragment"
			_, err := downloaded.ProcessURL(HTMLTagA, url+fragment)

			Expect(err).ToNot(HaveOccurred())

			urls := downloaded.GetDiscoveredURLs()
			Expect(len(urls)).To(Equal(1))
			Expect(urls[0].String()).To(Equal(url))
		})

		It("should not process empty url", func() {
			_, err := downloaded.ProcessURL(HTMLTagA, "")

			Expect(err).To(HaveOccurred())
		})

		It("should not process with missing .Input", func() {
			url := "https://domain.com/ProcessURL/not"
			downloaded.Input = nil
			_, err := downloaded.ProcessURL(HTMLTagA, url)

			Expect(err).To(HaveOccurred())
		})

		It("should not process with missing .Input.URL", func() {
			url := "https://domain.com/ProcessURL/not"
			downloaded.Input = &Input{}
			_, err := downloaded.ProcessURL(HTMLTagA, url)

			Expect(err).To(HaveOccurred())
		})

		It("should not process invalid url", func() {
			_, err := downloaded.ProcessURL(HTMLTagA, t.InvalidURL)

			Expect(err).To(HaveOccurred())
		})

		It("should not save #fragment url", func() {
			downloaded.BaseURL = downloaded.Input.URL
			_, _ = downloaded.ProcessURL(HTMLTagA, "#fragment")

			Expect(len(downloaded.GetDiscoveredURLs())).To(Equal(0))
		})

		It("should not save .Input.URL with #fragment", func() {
			_, _ = downloaded.ProcessURL(HTMLTagA, fmt.Sprintf("%s#fragment", downloaded.Input.URL))

			Expect(len(downloaded.GetDiscoveredURLs())).To(Equal(0))
		})

		Describe("Reduce", func() {
			It("should reduce url", func() {
				url := "https://domain.com/Reduce/reduce"
				parsedURL, _ := neturl.Parse(url)
				reduced := downloaded.Reduce(parsedURL)

				Expect(reduced).To(Equal("../Reduce/reduce"))
			})

			It("should reduce .Input.URL", func() {
				url := "https://domain.com/Reduce/reduce/self"
				parsedURL, _ := neturl.Parse(url)
				downloaded.Input.URL = parsedURL
				reduced := downloaded.Reduce(parsedURL)

				Expect(reduced).To(Equal("./self"))
			})

			Context("cross-host", func() {
				It("should reduce", func() {
					url := "https://domain2.com/Reduce/cross/host"
					parsedURL, _ := neturl.Parse(url)
					reduced := downloaded.Reduce(parsedURL)

					Expect(reduced).To(Equal("../../domain2.com/Reduce/cross/host"))
				})

				It("should reduce without scheme", func() {
					url := "https://domain2.com/Reduce/cross/host/without/scheme"
					parsedURL, _ := neturl.Parse(url)
					parsedURL.Scheme = ""
					reduced := downloaded.Reduce(parsedURL)

					Expect(reduced).To(Equal("../../domain2.com/Reduce/cross/host/without/scheme"))
				})

				It("should reduce cross scheme", func() {
					url := "ftp://domain2.com/Reduce/cross/scheme"
					parsedURL, _ := neturl.Parse(url)
					reduced := downloaded.Reduce(parsedURL)

					Expect(reduced).To(Equal("../../../ftp/domain2.com/Reduce/cross/scheme"))
				})

				It("should set ref header", func() {
					url := "https://domain2.com/Reduce/cross/host/ref/header"
					parsedURL, _ := neturl.Parse(url)
					downloaded.Reduce(parsedURL)

					Expect(len(downloaded.GetHeaderValues(cacher.CustomHeaderCrossHostRef))).To(Equal(1))
				})

				It("should set ref header once", func() {
					url := "https://domain2.com/Reduce/cross/host/ref/header/once"
					parsedURL, _ := neturl.Parse(url)
					downloaded.Reduce(parsedURL)
					downloaded.Reduce(parsedURL)

					Expect(len(downloaded.GetHeaderValues(cacher.CustomHeaderCrossHostRef))).To(Equal(1))
				})

				It("should not reduce", func() {
					url := "https://domain2.com/Reduce/cross/host/not"
					parsedURL, _ := neturl.Parse(url)
					downloaded.Input.NoCrossHost = true
					reduced := downloaded.Reduce(parsedURL)

					Expect(reduced).To(Equal(url))
				})
			})
		})
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
			Expect(urls[0].String()).To(Equal("https://domain.com/downloaded/relative/assets"))
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
			Expect(urls[0].String()).To(Equal("https://domain.com/downloaded/relative/discovered"))
		})
	})
})
