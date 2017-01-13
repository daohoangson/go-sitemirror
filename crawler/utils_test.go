package crawler_test

import (
	neturl "net/url"

	. "github.com/daohoangson/go-sitemirror/crawler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("ReduceURL", func() {
		It("should keep absolute url", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/absolute")
			url2, _ := neturl.Parse("http://domain2.com/something/else")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal(url2.String()))
		})

		It("should do relative", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/hi")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("hi"))
		})

		It("should do relative with slash", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative/")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/hi")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../hi"))
		})

		It("should do relative multiple level", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/relative/")
			url2, _ := neturl.Parse("http://domain.com/root")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("../../../root"))
		})
	})

	Describe("LongestCommonPrefix", func() {
		It("should handle no common prefix", func() {
			path1 := "/a"
			path2 := "/b"
			lcp := LongestCommonPrefix(path1, path2)

			Expect(lcp).To(Equal("/"))
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
