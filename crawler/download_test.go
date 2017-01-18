package crawler_test

import (
	"fmt"
	"net/http"
	neturl "net/url"

	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/daohoangson/go-sitemirror/crawler"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Download", func() {
	BeforeEach(func() {
		httpmock.Activate()
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	It("should not work with nil http.Client", func() {
		url := "http://domain.com/client/nil"

		parsedURL, _ := neturl.Parse(url)
		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(nil, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with nil url.URL", func() {
		downloaded := Download(http.DefaultClient, nil)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with relative url", func() {
		url := "relative/url/"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with non http/https url", func() {
		url := "ftp://domain.com/non/http/url"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should passthrough client error", func() {
		url := "http://a.b.c"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	Describe("BaseURL", func() {
		It("should match url", func() {
			url := "http://domain.com/download/url/base"
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BaseURL).To(Equal(parsedURL))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})

		It("should match base href", func() {
			url := "http://domain.com/download/url/base/href"
			baseHref := "/some/where/else"
			htmlTemplate := "<base href=\"%s\" />"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, baseHref))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "."))))
			Expect(downloaded.BaseURL.String()).To(Equal("http://domain.com/some/where/else"))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})

		It("should match url on empty base href", func() {
			url := "http://domain.com/download/url/base/href/empty"
			html := t.NewHtmlMarkup("<base />")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(downloaded.BaseURL).To(Equal(parsedURL))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})
	})

	Describe("Body", func() {
		It("should match generic response body", func() {
			url := "http://domain.com/download/body"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(len(downloaded.BodyString)).To(Equal(0))
			Expect(string(downloaded.BodyBytes)).To(Equal(body))
		})

		It("should match css", func() {
			url := "http://domain.com/download/body/css/valid"
			css := "body{background:#fff}"
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(css))
		})

		It("should match valid html", func() {
			url := "http://domain.com/download/body/html/valid"
			html := t.NewHtmlMarkup("<p>Hello&nbsp;World!</p>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
		})

		It("should keep invalid html intact", func() {
			url := "http://domain.com/download/body/html/invalid"
			html := t.NewHtmlMarkup("<p>Oops</p")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
		})

		It("should keep complicated html intact", func() {
			url := "http://domain.com/download/body/html/complicated"
			html := t.NewHtmlMarkup(`<div data-html="&lt;p class=&#34;something-else&#34;&gt;HTML&lt;/p&gt;"` +
				` class="something"` +
				` style="font-family:'Noto Sans',sans-serif;"` +
				`>Text</div>`)
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
		})
	})

	Describe("ContentType", func() {
		It("should match response header value", func() {
			url := "http://domain.com/download/content/type"
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.ContentType).To(Equal("text/html"))
		})
	})

	Describe("HeaderLocation", func() {
		It("should work with 301 response status", func() {
			status := 301
			url := fmt.Sprintf("http://domain.com/download/header/location/%d", status)
			targetUrl := "http://domain.com/download/target/url"
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(status, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(status))
			Expect(downloaded.HeaderLocation.String()).To(Equal(targetUrl))
		})

		It("should not work with invalid url", func() {
			// have to use 399 status code otherwise http.Client will parse
			// the location header itself and trigger error too soon
			status := 399
			url := "http://domain.com/download/header/location/invalid"
			targetUrl := t.InvalidURL
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(status, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(status))
			Expect(downloaded.HeaderLocation).To(BeNil())
		})
	})

	Describe("Links", func() {
		It("should pick up css url() value", func() {
			url := "http://domain.com/download/urls/css/url"
			targetUrl := "http://domain.com/download/urls/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up css url() value, double quote", func() {
			url := "http://domain.com/download/urls/css/url/double/quote"
			targetUrl := "http://domain.com/download/urls/css/url/double/target"
			cssTemplate := "body{background:url(\"%s\")}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up css url() value, no quote", func() {
			url := "http://domain.com/download/urls/css/url/no/quote"
			targetUrl := "http://domain.com/download/urls/css/url/no/target"
			cssTemplate := "body{background:url(%s)}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "./target")))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up a href", func() {
			url := "http://domain.com/download/urls/a"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := "<a href=\"%s\">Link</a><a>Anchor</a>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagA))
			}
		})

		It("should pick up form action", func() {
			url := "http://domain.com/download/urls/form"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := `<form action="%s"></form><form></form>`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagForm))
			}
		})

		It("should pick up img src, using start tag", func() {
			url := "http://domain.com/download/urls/img/start"
			targetUrl := "http://domain.com/download/urls/img/target"
			htmlTemplate := `<img src="%s"></img>`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up link[rel=stylesheet] href, using start tag", func() {
			url := "http://domain.com/download/urls/link/stylesheet/start"
			targetUrl := "http://domain.com/download/urls/link/stylesheet/target"
			htmlTemplate := "<link rel=\"stylesheet\" href=\"%s\"></link>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagLinkStylesheet))
			}
		})

		It("should pick up script src", func() {
			url := "http://domain.com/download/urls/script"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := "<script src=\"%s\"></script><script>alert('hello world');</script>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagScript))
			}
		})

		It("should remove inline script with base", func() {
			url := "http://domain.com/download/urls/script"
			html := t.NewHtmlMarkup("<script>document.getElementsByTagName('base').something();</script>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup("<script></script>")))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should pick up internal css url() value", func() {
			url := "http://domain.com/download/urls/internal/css/url"
			targetUrl := "http://domain.com/download/urls/internal/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			htmlTemplate := "<style>%s</style>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, css))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			cssNew := fmt.Sprintf(cssTemplate, "./target")
			htmlNew := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, cssNew))
			Expect(downloaded.BodyString).To(Equal(htmlNew))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up img src", func() {
			url := "http://domain.com/download/urls/img"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := `<img src="%s" /><img class="friend" data-hello="world" data-invalid="` + t.InvalidURL + `" />`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up img data-src", func() {
			url := "http://domain.com/download/urls/img/data-src"
			targetUrl := "http://domain.com/download/urls/img/target"
			htmlTemplate := `<img data-src="%s" />`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagImg))
			}
		})

		It("should pick up img src inside a", func() {
			url := "http://domain.com/download/urls/img/inside/a"
			targetUrl0 := "http://domain.com/download/urls/img/inside/target/0"
			targetUrl1 := "http://domain.com/download/urls/img/inside/target/1"
			htmlTemplate := `<a href="%s"><img src="%s" /></a>`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl0, targetUrl1))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target/0", "./target/1"))))

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
			url := "http://domain.com/download/urls/link/stylesheet"
			targetUrl := "http://domain.com/download/urls/link/target"
			htmlTemplate := "<link rel=\"stylesheet\" href=\"%s\" /><link />"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"))))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTMLTagLinkStylesheet))
			}
		})

		It("should pick up inline css url() value", func() {
			url := "http://domain.com/download/urls/inline/css/url"
			targetUrl := "http://domain.com/download/urls/inline/css/target"
			cssTemplate := "background:url('%s')"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			htmlTemplate := `<div style="%s"></style>`
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, css))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			cssNew := fmt.Sprintf(cssTemplate, "./target")
			htmlNew := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, cssNew))
			Expect(downloaded.BodyString).To(Equal(htmlNew))
			Expect(len(downloaded.LinksAssets)).To(Equal(1))

			for _, link := range downloaded.LinksAssets {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(CSSUri))
			}
		})

		It("should pick up 3xx response Location header", func() {
			url := "http://domain.com/download/urls/3xx"
			targetUrl := "http://domain.com/download/target/url"
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(301, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrl))
				Expect(link.Context).To(Equal(HTTP3xxLocation))
			}
		})

		It("should pick up multiple urls", func() {
			url := "http://domain.com/download/urls/multiple"
			targetUrlA := "http://domain.com/download/target/url/a"
			targetUrlAHttps := "https://domain.com/download/target/url/a/https"
			targetUrlAProtocolRelative := "//domain.com/download/target/url/a/protocol/relative"
			targetUrlScript := "http://domain.com/download/target/url/script"
			targetUrlCssUri := "http://domain.com/download/target/url/css/uri"
			targetUrlImg := "http://domain.com/download/target/url/img"
			targetUrlLink := "http://domain.com/download/target/url/link"
			css := fmt.Sprintf("body{background:url('%s')}", targetUrlCssUri)
			html := t.NewHtmlMarkup(
				fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrlA) +
					fmt.Sprintf("<a href=\"%s\">Link HTTPS</a>", targetUrlAHttps) +
					fmt.Sprintf("<a href=\"%s\">Link protocol relative</a>", targetUrlAProtocolRelative) +
					fmt.Sprintf("<script src=\"%s\"></script>", targetUrlScript) +
					fmt.Sprintf("<style>%s</style>", css) +
					fmt.Sprintf("<img src=\"%s\" />", targetUrlImg) +
					fmt.Sprintf("<link rel=\"stylesheet\" href=\"%s\" />", targetUrlLink))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

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
			for _, _ = range found {
				foundCount++
			}

			Expect(foundCount).To(Equal(7))
		})

		It("should not pick up empty url", func() {
			url := "http://domain.com/download/urls/empty/url"
			html := t.NewHtmlMarkup("<a href=\"\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up invalid url", func() {
			url := "http://domain.com/download/urls/invalid/url"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", t.InvalidURL))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up non http/https url", func() {
			url := "http://domain.com/download/urls/non/http/url"
			html := t.NewHtmlMarkup("<a href=\"ftp://domain.com/non/http/url\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up data uri", func() {
			url := "http://domain.com/download/urls/data/uri"
			html := t.NewHtmlMarkup(fmt.Sprintf("<img src=\"%s\" />", t.TransparentDataURI))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})

		It("should not pick up url #fragment part", func() {
			url := "http://domain.com/download/urls/fragment"
			targetUrlBase := "http://domain.com/download/urls/target"
			fragment := "#foo=bar"
			targetUrl := targetUrlBase + fragment
			htmlTemplate := "<a href=\"%s\">Link</a>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "./target"+fragment))))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(1))

			for _, link := range downloaded.LinksDiscovered {
				Expect(link.URL.String()).To(Equal(targetUrlBase))
			}
		})

		It("should not pick up #fragment only url", func() {
			url := "http://domain.com/download/urls/fragment/only"
			html := t.NewHtmlMarkup("<a href=\"#\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.LinksAssets)).To(Equal(0))
			Expect(len(downloaded.LinksDiscovered)).To(Equal(0))
		})
	})

	Describe("StatusCode", func() {
		It("should match response status code", func() {
			url := "http://domain.com/download/status/code"
			statusCode := 200
			httpmock.RegisterResponder("GET", url,
				httpmock.NewStringResponder(statusCode, ""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(statusCode))
		})
	})
})
