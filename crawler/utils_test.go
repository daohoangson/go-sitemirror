package crawler_test

import (
	neturl "net/url"

	. "github.com/daohoangson/go-sitemirror/crawler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("ReduceURL", func() {

		expectResolveOk := func(base *neturl.URL, reduced string, target *neturl.URL) {
			parsedReduced, err := neturl.Parse(reduced)
			Expect(err).ToNot(HaveOccurred())

			test := base.ResolveReference(parsedReduced)
			Expect(test).To(Equal(target))
		}

		It("should keep url intact if base is not absolute", func() {
			url1, _ := neturl.Parse("reduce/base/not/absolute")
			url2, _ := neturl.Parse("http://domain.com/other")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal(url2.String()))
		})

		It("should keep url intact if url is not absolute", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/not/absolute")
			url2, _ := neturl.Parse("other")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal(url2.String()))
		})

		It("should keep url intact if scheme mismatched", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/not/absolute")
			url2, _ := neturl.Parse("ftp://domain.com/other")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal(url2.String()))
		})

		It("should keep url intact if host mismatched", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/not/absolute")
			url2, _ := neturl.Parse("http://domain2.com/other")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal(url2.String()))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./ok"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative http->https", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative")
			url2, _ := neturl.Parse("https://domain.com/reduce/url/https")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./https"))

			url2InHttp, _ := neturl.Parse(url2.String())
			url2InHttp.Scheme = "http"
			expectResolveOk(url1, reduced, url2InHttp)
		})

		It("should do relative https->http", func() {
			url1, _ := neturl.Parse("https://domain.com/reduce/url/relative")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/http")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./http"))

			url2InHttps, _ := neturl.Parse(url2.String())
			url2InHttps.Scheme = "https"
			expectResolveOk(url1, reduced, url2InHttps)
		})

		It("should do relative with slash", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative/")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../ok"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative multiple level", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative")
			url2, _ := neturl.Parse("http://domain.com/multiple")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../../multiple"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative multiple level with slash", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative/")
			url2, _ := neturl.Parse("http://domain.com/multiple/with/slash")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../../../multiple/with/slash"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative from root", func() {
			url1, _ := neturl.Parse("http://domain.com")
			url2, _ := neturl.Parse("http://domain.com/relative/from/root")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./relative/from/root"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative from root with slash", func() {
			url1, _ := neturl.Parse("http://domain.com/")
			url2, _ := neturl.Parse("http://domain.com/relative/from/root/with/slash")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./relative/from/root/with/slash"))
			expectResolveOk(url1, reduced, url2)
		})

		It("should do relative to root", func() {
			url1, _ := neturl.Parse("http://domain.com/relative/to/root/with/slash")
			url2, _ := neturl.Parse("http://domain.com/")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../../../.."))
			expectResolveOk(url1, reduced, url2)
		})
	})

	Describe("LongestCommonPrefix", func() {
		It("should handle no common prefix", func() {
			path1 := "/a"
			path2 := "/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("/"))
		})

		It("should handle no common prefix (uneven parts)", func() {
			path1 := "/"
			path2 := "/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("/"))
		})

		It("should handle no common prefix (one empty)", func() {
			path1 := ""
			path2 := "/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal(""))
		})

		It("should handle no common prefix (both empty)", func() {
			path1 := ""
			path2 := ""
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal(""))
		})

		It("should handle common prefix but not whole part", func() {
			path1 := "/aa"
			path2 := "/ab"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("/"))
		})

		It("should handle common prefix", func() {
			path1 := "/a/a"
			path2 := "/a/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("/a/"))
		})

		It("should handle common prefix without slash at the beginning", func() {
			path1 := "a/a"
			path2 := "a/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("a/"))
		})
	})
})
