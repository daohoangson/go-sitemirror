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

		Context("should keep url intact", func() {
			It("base is not absolute", func() {
				url1, _ := neturl.Parse("base/not/absolute")
				url2, _ := neturl.Parse("http://domain.com/url")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal(url2.String()))
			})

			It("url is not absolute", func() {
				url1, _ := neturl.Parse("http://domain.com/base")
				url2, _ := neturl.Parse("url/not/absolute")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal(url2.String()))
			})

			It("scheme mismatched", func() {
				url1, _ := neturl.Parse("http://domain.com/base")
				url2, _ := neturl.Parse("ftp://domain.com/url")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal(url2.String()))
			})

			It("host mismatched", func() {
				url1, _ := neturl.Parse("http://domain.com/base")
				url2, _ := neturl.Parse("http://domain2.com/url")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal(url2.String()))
				expectResolveOk(url1, reduced, url2)
			})
		})

		It("should do http->https", func() {
			url1, _ := neturl.Parse("http://domain.com/reduce/url/http")
			url2, _ := neturl.Parse("https://domain.com/reduce/url/https")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./https"))

			url2InHttp, _ := neturl.Parse(url2.String())
			url2InHttp.Scheme = "http"
			expectResolveOk(url1, reduced, url2InHttp)
		})

		It("should do https->http", func() {
			url1, _ := neturl.Parse("https://domain.com/reduce/url/https")
			url2, _ := neturl.Parse("http://domain.com/reduce/url/http")
			reduced := ReduceURL(url1, url2)

			Expect(reduced).To(Equal("./http"))

			url2InHttps, _ := neturl.Parse(url2.String())
			url2InHttps.Scheme = "https"
			expectResolveOk(url1, reduced, url2InHttps)
		})

		Context("self", func() {
			It("is file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/self")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/self")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./self"))
				expectResolveOk(url1, reduced, url2)
			})

			It("is dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/self/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/self/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./"))
				expectResolveOk(url1, reduced, url2)
			})
		})

		Context("siblings", func() {
			It("file to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/siblings")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./ok"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/siblings/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../ok"))
				expectResolveOk(url1, reduced, url2)
			})

			It("file to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/siblings")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./ok/"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/siblings/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../ok/"))
				expectResolveOk(url1, reduced, url2)
			})
		})

		Context("to grand child", func() {
			It("file to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/child")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./url/to/grand/child"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/child")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./to/grand/child"))
				expectResolveOk(url1, reduced, url2)
			})

			It("file to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/child/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./url/to/grand/child/"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/child/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./to/grand/child/"))
				expectResolveOk(url1, reduced, url2)
			})
		})

		Context("to grand parent", func() {
			It("file to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/parent")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../ok"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to file", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/parent/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../../ok"))
				expectResolveOk(url1, reduced, url2)
			})

			It("file to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/parent")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../ok/"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/grand/parent/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/ok/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../../ok/"))
				expectResolveOk(url1, reduced, url2)
			})
		})

		Context("from root", func() {
			It("file to file", func() {
				url1, _ := neturl.Parse("http://domain.com")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/from/root")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./reduce/url/from/root"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to file", func() {
				url1, _ := neturl.Parse("http://domain.com/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/from/root")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./reduce/url/from/root"))
				expectResolveOk(url1, reduced, url2)
			})

			It("file to dir", func() {
				url1, _ := neturl.Parse("http://domain.com")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/from/root/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./reduce/url/from/root/"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/")
				url2, _ := neturl.Parse("http://domain.com/reduce/url/from/root/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("./reduce/url/from/root/"))
				expectResolveOk(url1, reduced, url2)
			})
		})

		Context("to root", func() {
			It("file to file", func() {
				// TODO
			})

			It("dir to file", func() {
				// TODO
			})

			It("file to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/root")
				url2, _ := neturl.Parse("http://domain.com/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../../"))
				expectResolveOk(url1, reduced, url2)
			})

			It("dir to dir", func() {
				url1, _ := neturl.Parse("http://domain.com/reduce/url/to/root/")
				url2, _ := neturl.Parse("http://domain.com/")
				reduced := ReduceURL(url1, url2)

				Expect(reduced).To(Equal("../../../../"))
				expectResolveOk(url1, reduced, url2)
			})
		})
	})

	Describe("LongestCommonPrefix", func() {
		Context("has slash prefix", func() {
			It("should handle no common prefix", func() {
				path1 := "/one"
				path2 := "/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/"))
			})

			It("should handle no common prefix (uneven parts)", func() {
				path1 := "/one"
				path2 := "/three"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/"))
			})

			It("should handle no common prefix (one empty)", func() {
				path1 := "/"
				path2 := "/b"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/"))
			})

			It("should handle both empty", func() {
				path1 := "/"
				path2 := "/"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/"))
			})

			It("should handle common prefix but not whole part", func() {
				path1 := "/oneone"
				path2 := "/onetwo"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/"))
			})

			It("should handle matches", func() {
				path1 := "/one"
				path2 := "/one"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/one"))
			})

			It("should handle common prefix (uneven parts, without slash)", func() {
				path1 := "/one"
				path2 := "/one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/one"))
			})

			It("should handle common prefix (uneven parts, with slash)", func() {
				path1 := "/one/"
				path2 := "/one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/one/"))
			})

			It("should handle common prefix (same parts)", func() {
				path1 := "/one/one"
				path2 := "/one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("/one/"))
			})
		})

		Context("has no slash prefix", func() {
			It("should handle no common prefix", func() {
				path1 := "one"
				path2 := "two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal(""))
			})

			It("should handle no common prefix (uneven parts)", func() {
				path1 := "one"
				path2 := "three"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal(""))
			})

			It("should handle no common prefix (one empty)", func() {
				path1 := ""
				path2 := "b"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal(""))
			})

			It("should handle both empty", func() {
				path1 := ""
				path2 := ""
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal(""))
			})

			It("should handle common prefix but not whole part", func() {
				path1 := "oneone"
				path2 := "onetwo"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal(""))
			})

			It("should handle matches", func() {
				path1 := "one"
				path2 := "one"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("one"))
			})

			It("should handle common prefix (uneven parts, without slash)", func() {
				path1 := "one"
				path2 := "one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("one"))
			})

			It("should handle common prefix (uneven parts, with slash)", func() {
				path1 := "one/"
				path2 := "one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("one/"))
			})

			It("should handle common prefix (same parts)", func() {
				path1 := "one/one"
				path2 := "one/two"
				lcp := LongestCommonPrefix(path1, path2)

				Expect(lcp).To(Equal("one/"))
			})
		})
	})
})
