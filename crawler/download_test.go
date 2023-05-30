package crawler_test

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"time"

	"github.com/jarcoal/httpmock"

	"github.com/daohoangson/go-sitemirror/cacher"
	. "github.com/daohoangson/go-sitemirror/crawler"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Download", func() {

	downloadWithDefaultClient := func(url string) *Downloaded {
		parsedURL, err := neturl.Parse(url)
		Expect(err).ToNot(HaveOccurred())

		return Download(&Input{
			Client: http.DefaultClient,
			URL:    parsedURL,
		})
	}

	BeforeEach(func() {
		httpmock.Activate()
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	It("should not work with nil http.Client", func() {
		url := "https://domain.com/client/nil"

		parsedURL, _ := neturl.Parse(url)
		downloaded := Download(&Input{URL: parsedURL})

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with nil url.URL", func() {
		downloaded := Download(&Input{Client: http.DefaultClient})

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should set request header", func() {
		url := "https://domain.com/request/header"
		header := make(http.Header)
		headerKey := "Key"
		headerValue := "Value"
		header.Add(headerKey, headerValue)
		httpmock.RegisterResponder("GET", url, func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, req.Header.Get(headerKey))
			return resp, nil
		})
		parsedURL, _ := neturl.Parse(url)

		downloaded := Download(&Input{
			Client: http.DefaultClient,
			Header: header,
			URL:    parsedURL,
		})

		Expect(downloaded.Body).To(Equal(headerValue))
	})

	It("should not work with relative url", func() {
		url := "relative/url/"
		downloaded := downloadWithDefaultClient(url)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with non http/https url", func() {
		url := "ftp://domain.com/non/http/url"
		downloaded := downloadWithDefaultClient(url)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should fix root", func() {
		host := "download.fix-root.com"
		url := "https://" + host
		httpmock.RegisterResponder("GET", url+"/", httpmock.NewStringResponder(http.StatusOK, ""))

		downloaded := downloadWithDefaultClient(url)
		Expect(downloaded.StatusCode).To(Equal(http.StatusOK))
	})

	It("should relay request error", func() {
		url := "https://domain.com/Download/request/error"
		parsedURL, _ := neturl.Parse(url)
		parsedURL.Host = "/"
		downloaded := Download(&Input{
			Client: http.DefaultClient,
			URL:    parsedURL,
		})

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should relay client error", func() {
		url := "http://a.b.c"
		downloaded := downloadWithDefaultClient(url)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	Describe("BaseURL", func() {
		It("should match url", func() {
			url := "https://domain.com/download/url/base"
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(""))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.BaseURL.String()).To(Equal(url))
		})

		It("should match base href", func() {
			url := "https://domain.com/download/url/base/href"
			baseHref := "/some/where/else"
			htmlText := "<p>Text</p>"
			htmlTemplate := "<base href=\"%s\" />" + htmlText
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, baseHref))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(htmlText)))
			Expect(downloaded.BaseURL.String()).To(Equal("https://domain.com/some/where/else"))
		})

		It("should match url on empty base href", func() {
			url := "https://domain.com/download/url/base/href/empty"
			html := t.NewHTMLMarkup("<base />")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
			Expect(downloaded.BaseURL.String()).To(Equal(url))
		})
	})

	Describe("Body", func() {
		It("should match generic response body", func() {
			url := "https://domain.com/download/body"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(body))
		})

		It("should match css", func() {
			url := "https://domain.com/download/body/css/valid"
			css := "body{background:#fff}"
			httpmock.RegisterResponder("GET", url, t.NewCSSResponder(css))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(css))
		})

		It("should match valid html", func() {
			url := "https://domain.com/download/body/html/valid"
			html := t.NewHTMLMarkup("<p>Hello&nbsp;World!</p>")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
		})

		It("should keep invalid html intact", func() {
			url := "https://domain.com/download/body/html/invalid"
			html := t.NewHTMLMarkup("<p>Oops</p")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
		})

		It("should keep complicated html intact", func() {
			url := "https://domain.com/download/body/html/complicated"
			html := t.NewHTMLMarkup(`<div data-html="&lt;p class=&#34;something-else&#34;&gt;HTML&lt;/p&gt;"` +
				` class="something"` +
				` style="font-family:'Noto Sans',sans-serif;"` +
				`>Text</div>`)
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
		})
	})

	Describe("Header", func() {
		Context(cacher.HeaderContentType, func() {
			It("should pick up header value", func() {
				url := "https://domain.com/download/header/content/type"
				httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(""))

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.GetHeaderValues(cacher.HeaderContentType)).To(Equal([]string{"text/html"}))
			})
		})

		Context(cacher.HeaderCacheControl, func() {
			It("should pick up header value", func() {
				url := "https://domain.com/download/header/caching/expires"
				cacheControl := "public"
				httpmock.RegisterResponder("GET", url, func(req *http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(http.StatusOK, "")
					resp.Header.Add(cacher.HeaderCacheControl, cacheControl)
					return resp, nil
				})

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.GetHeaderValues(cacher.HeaderCacheControl)).To(Equal([]string{cacheControl}))
			})

			It("should not pick up value (none given)", func() {
				url := "https://domain.com/download/header/caching/expires/non"
				httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(""))

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.GetHeaderValues(cacher.HeaderCacheControl)).To(BeNil())
			})
		})

		Context(cacher.HeaderExpires, func() {
			It("should pick up header value", func() {
				url := "https://domain.com/download/header/caching/expires"
				expires := time.Now().Add(time.Hour).Format(http.TimeFormat)
				httpmock.RegisterResponder("GET", url, func(req *http.Request) (*http.Response, error) {
					resp := httpmock.NewStringResponse(http.StatusOK, "")
					resp.Header.Add(cacher.HeaderExpires, expires)
					return resp, nil
				})

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.GetHeaderValues(cacher.HeaderExpires)).To(Equal([]string{expires}))
			})

			It("should not pick up value (none given)", func() {
				url := "https://domain.com/download/header/caching/expires/non"
				httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(""))

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.GetHeaderValues(cacher.HeaderExpires)).To(BeNil())
			})
		})

		Context(cacher.HeaderLocation, func() {
			It("should pick up header value", func() {
				status := http.StatusMovedPermanently
				url := fmt.Sprintf("https://domain.com/download/header/location/%d", status)
				targetUrl := "https://domain.com/download/header/location/target"
				httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(status, targetUrl))

				downloaded := downloadWithDefaultClient(url)

				Expect(downloaded.StatusCode).To(Equal(status))
				Expect(downloaded.GetHeaderValues(cacher.HeaderLocation)).To(Equal([]string{"./target"}))
			})
		})
	})

	Describe("Links", func() {
		It("should pick up css url() value", func() {
			url := "https://domain.com/download/urls/css/url"
			targetUrl := "https://domain.com/download/urls/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCSSResponder(css))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up css url() value, double quote", func() {
			url := "https://domain.com/download/urls/css/url/double/quote"
			targetUrl := "https://domain.com/download/urls/css/url/double/target"
			cssTemplate := "body{background:url(\"%s\")}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCSSResponder(css))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up css url() value, no quote", func() {
			url := "https://domain.com/download/urls/css/url/no/quote"
			targetUrl := "https://domain.com/download/urls/css/url/no/target"
			cssTemplate := "body{background:url(%s)}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCSSResponder(css))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up a href", func() {
			url := "https://domain.com/download/urls/a"
			targetUrl := "https://domain.com/download/urls/target"
			htmlTemplate := "<a href=\"%s\">Link</a><a>Anchor</a>"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagA))
			}
		})

		It("should pick up form action", func() {
			url := "https://domain.com/download/urls/form"
			targetUrl := "https://domain.com/download/urls/target"
			htmlTemplate := `<form action="%s"></form><form></form>`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagForm))
			}
		})

		It("should pick up img src, using start tag", func() {
			url := "https://domain.com/download/urls/img/start"
			targetUrl := "https://domain.com/download/urls/img/target"
			htmlTemplate := `<img src="%s"></img>`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up link[rel=stylesheet] href, using start tag", func() {
			url := "https://domain.com/download/urls/link/stylesheet/start"
			targetUrl := "https://domain.com/download/urls/link/stylesheet/target"
			htmlTemplate := "<link rel=\"stylesheet\" href=\"%s\"></link>"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagLinkStylesheet))
			}
		})

		It("should pick up script src", func() {
			url := "https://domain.com/download/urls/script"
			targetUrl := "https://domain.com/download/urls/target"
			htmlTemplate := "<script src=\"%s\"></script><script>alert('hello world');</script>"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagScript))
			}
		})

		It("should remove inline script with base", func() {
			url := "https://domain.com/download/urls/script"
			html := t.NewHTMLMarkup("<script>document.getElementsByTagName('base').something();</script>")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup("<script></script>")))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should pick up internal css url() value", func() {
			url := "https://domain.com/download/urls/internal/css/url"
			targetUrl := "https://domain.com/download/urls/internal/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			htmlTemplate := "<style>%s</style>"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, css))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			cssNew := fmt.Sprintf(cssTemplate, "./target")
			htmlNew := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, cssNew))
			Expect(downloaded.Body).To(Equal(htmlNew))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up img src", func() {
			url := "https://domain.com/download/urls/img"
			targetUrl := "https://domain.com/download/urls/target"
			htmlTemplate := `<img src="%s" /><img class="friend" data-hello="world" data-invalid="%s" />`

			//noinspection GoPlaceholderCount
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl, t.InvalidURL))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			//noinspection GoPlaceholderCount
			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target", t.InvalidURL))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up img data-src", func() {
			url := "https://domain.com/download/urls/img/data-src"
			targetUrl := "https://domain.com/download/urls/img/target"
			htmlTemplate := `<img data-src="%s" />`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up img src inside a", func() {
			url := "https://domain.com/download/urls/img/inside/a"
			targetUrl0 := "https://domain.com/download/urls/img/inside/target/0"
			targetUrl1 := "https://domain.com/download/urls/img/inside/target/1"
			htmlTemplate := `<a href="%s"><img src="%s" /></a>`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl0, targetUrl1))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target/0", "./target/1"))))

			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))
			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl0))
				Expect(link.Context).To(Equal(HTMLTagA))
			}

			Expect(len(downloaded.LinksAssets)).To(Equal(1))
			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl1))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up link[rel=stylesheet] href", func() {
			url := "https://domain.com/download/urls/link/stylesheet"
			targetUrl := "https://domain.com/download/urls/link/target"
			htmlTemplate := "<link rel=\"stylesheet\" href=\"%s\" /><link />"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagLinkStylesheet))
			}
		})

		It("should pick up inline css url() value", func() {
			url := "https://domain.com/download/urls/inline/css/url"
			targetUrl := "https://domain.com/download/urls/inline/css/target"
			cssTemplate := "background:url('%s')"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			htmlTemplate := `<div style="%s"></style>`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, css))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			cssNew := fmt.Sprintf(cssTemplate, "./target")
			htmlNew := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, cssNew))
			Expect(downloaded.Body).To(Equal(htmlNew))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up 3xx response Location header", func() {
			url := "https://domain.com/download/urls/3xx"
			targetUrl := "https://domain.com/download/target/url"
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(301, targetUrl))

			downloaded := downloadWithDefaultClient(url)

			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTTP3xxLocation))
			}
		})

		It("should pick up multiple urls", func() {
			url := "https://domain.com/download/urls/multiple"
			targetUrlA := "https://domain.com/download/target/url/a"
			targetUrlAHttps := "https://domain.com/download/target/url/a/https"
			targetUrlAProtocolRelative := "//domain.com/download/target/url/a/protocol/relative"
			targetUrlScript := "https://domain.com/download/target/url/script"
			targetUrlCssUri := "https://domain.com/download/target/url/css/uri"
			targetUrlImg := "https://domain.com/download/target/url/img"
			targetUrlLink := "https://domain.com/download/target/url/link"
			css := fmt.Sprintf("body{background:url('%s')}", targetUrlCssUri)
			html := t.NewHTMLMarkup(
				fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrlA) +
					fmt.Sprintf("<a href=\"%s\">Link HTTPS</a>", targetUrlAHttps) +
					fmt.Sprintf("<a href=\"%s\">Link protocol relative</a>", targetUrlAProtocolRelative) +
					fmt.Sprintf("<script src=\"%s\"></script>", targetUrlScript) +
					fmt.Sprintf("<style>%s</style>", css) +
					fmt.Sprintf("<img src=\"%s\" />", targetUrlImg) +
					fmt.Sprintf("<link rel=\"stylesheet\" href=\"%s\" />", targetUrlLink))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(len(downloaded.LinksAssets)).To(Equal(4))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(3))

			found := make(map[string]bool)

			for url, link := range downloaded.LinksAssets {
				found[url] = true

				switch url {
				case targetUrlScript:
					Expect(link.Context).To(Equal(HTMLTagScript))
				case targetUrlCssUri:
					Expect(link.Context).To(Equal(CSSUri))
				case targetUrlImg:
					Expect(link.Context).To(Equal(HTMLTagImg))
				case targetUrlLink:
					Expect(link.Context).To(Equal(HTMLTagLinkStylesheet))
				}
			}

			for url, link := range downloaded.LinksDiscovered {
				found[url] = true

				switch url {
				case targetUrlA:
					Expect(link.Context).To(Equal(HTMLTagA))
				case targetUrlAHttps:
					Expect(link.Context).To(Equal(HTMLTagA))
				case "http:" + targetUrlAProtocolRelative:
					Expect(link.Context).To(Equal(HTMLTagA))
				}
			}

			foundCount := 0
			for range found {
				foundCount++
			}

			Expect(foundCount).To(Equal(7))
		})

		It("should not pick up empty url", func() {
			url := "https://domain.com/download/urls/empty/url"
			html := t.NewHTMLMarkup("<a href=\"\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up non http/https url", func() {
			url := "https://domain.com/download/urls/non/http/url"
			html := t.NewHTMLMarkup("<a href=\"ftp://domain.com/non/http/url\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up data uri", func() {
			url := "https://domain.com/download/urls/data/uri"
			html := t.NewHTMLMarkup(fmt.Sprintf("<img src=\"%s\" />", t.TransparentDataURI))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up url #fragment part", func() {
			url := "https://domain.com/download/urls/fragment"
			targetUrlBase := "https://domain.com/download/urls/target"
			fragment := "#foo=bar"
			targetUrl := targetUrlBase + fragment
			htmlTemplate := "<a href=\"%s\">Link</a>"
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./target"+fragment))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrlBase))
			}
		})

		It("should not pick up #fragment only url", func() {
			url := "https://domain.com/download/urls/fragment/only"
			htmlTemplate := `<a href="%s">Link</a>`
			html := t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "#"))
			httpmock.RegisterResponder("GET", url, t.NewHTMLResponder(html))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.Body).To(Equal(t.NewHTMLMarkup(fmt.Sprintf(htmlTemplate, "./only"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})
	})

	Describe("StatusCode", func() {
		It("should match response status code", func() {
			url := "https://domain.com/download/status/code"
			statusCode := 200
			httpmock.RegisterResponder("GET", url,
				httpmock.NewStringResponder(statusCode, ""))

			downloaded := downloadWithDefaultClient(url)

			Expect(downloaded.StatusCode).To(Equal(statusCode))
		})
	})
})
