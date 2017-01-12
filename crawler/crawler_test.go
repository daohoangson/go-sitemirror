package crawler_test

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"time"

	. "github.com/daohoangson/go-sitemirror/crawler"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crawler", func() {
	BeforeEach(func() {
		httpmock.Activate()
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	const sleepTime = 10 * time.Millisecond

	Describe("Crawl", func() {
		It("should process one url", func() {
			url := "http://domain.com/crawl"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			c := Crawl(http.DefaultClient, url)
			downloaded := c.Next()

			Expect(string(downloaded.BodyBytes)).To(Equal(body))

			Expect(c.GetQueuedCount()).To(Equal(1))
			Expect(c.GetDownloadedCount()).To(Equal(1))
			Expect(c.GetLinkFoundCount()).To(Equal(0))
		})

		It("should also download link", func() {
			url := "http://domain.com/crawl/download/link"
			targetUrl := "http://domain.com/crawl/download/link/target"
			html := newHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Text</a>", targetUrl))
			httpmock.RegisterResponder("GET", url, newHtmlResponder(html))
			httpmock.RegisterResponder("GET", targetUrl, httpmock.NewStringResponder(200, "foo/bar"))

			c := Crawl(http.DefaultClient, url)
			downloaded := c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded = c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(targetUrl))

			Expect(c.GetQueuedCount()).To(Equal(2))
			Expect(c.GetDownloadedCount()).To(Equal(2))
			Expect(c.GetLinkFoundCount()).To(Equal(1))
		})

		It("should only download link at depth 1", func() {
			url := "http://domain.com/crawl/depth/0"
			urlDepth1 := "http://domain.com/crawl/depth/1"
			urlDepth2 := "http://domain.com/crawl/depth/2"
			html := newHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Text</a>", urlDepth1))
			html1 := newHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Text</a>", urlDepth2))
			httpmock.RegisterResponder("GET", url, newHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDepth1, newHtmlResponder(html1))

			c := Crawl(http.DefaultClient, url)
			downloaded := c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			time.Sleep(sleepTime)
			downloaded = c.NextOrNil()
			Expect(downloaded.BaseURL.String()).To(Equal(urlDepth1))

			time.Sleep(sleepTime)
			downloaded = c.NextOrNil()
			Expect(downloaded).To(BeNil())

			Expect(c.GetQueuedCount()).To(Equal(2))
			Expect(c.GetDownloadedCount()).To(Equal(2))
			Expect(c.GetLinkFoundCount()).To(Equal(2))
		})
	})

	It("should work with init(nil)", func() {
		url := "http://domain.com/crawl/init/nil"
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

		c := New(nil)
		c.Download(url)
		downloaded := c.Next()
		Expect(downloaded.BaseURL.String()).To(Equal(url))
	})

	Describe("WorkerCount", func() {
		It("should not accept zero", func() {
			c := New(nil)
			err := c.SetWorkerCount(0)
			Expect(err).To(HaveOccurred())
		})

		It("should not accept negative value", func() {
			c := New(nil)
			err := c.SetWorkerCount(-1)
			Expect(err).To(HaveOccurred())
		})

		It("should not work after Start", func() {
			c := New(nil)
			c.Start()
			err := c.SetWorkerCount(1)
			Expect(err).To(HaveOccurred())
		})

		It("should work", func() {
			workerCount := 1

			c := New(nil)
			err := c.SetWorkerCount(workerCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.GetWorkerCount()).To(Equal(workerCount))
		})
	})

	Describe("SetOnDownloaded", func() {
		It("should trigger func directly", func() {
			url := "http://domain.com/SetOnDownloaded"
			urlTarget0 := "http://domain.com/SetOnDownloaded/target/0"
			urlTarget1 := "http://domain.com/SetOnDownloaded/target/1"
			html := newHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Text</a>"+
				"<a href=\"%s\">Text</a>", urlTarget0, urlTarget1))
			httpmock.RegisterResponder("GET", url, newHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlTarget0, httpmock.NewStringResponder(200, "foo/bar"))
			httpmock.RegisterResponder("GET", urlTarget1, httpmock.NewStringResponder(200, "foo/bar"))

			c := New(http.DefaultClient)
			urlFound := false
			urlTarget0Found := false
			urlTarget1Found := false
			c.SetOnDownloaded(func(d *Downloaded) {
				switch d.BaseURL.String() {
				case url:
					urlFound = true
				case urlTarget0:
					urlTarget0Found = true
				case urlTarget1:
					urlTarget1Found = true
				}
			})
			c.Download(url)

			time.Sleep(sleepTime)
			downloaded := c.NextOrNil()
			Expect(downloaded).To(BeNil())

			Expect(urlFound).To(BeTrue())
			Expect(urlTarget0Found).To(BeTrue())
			Expect(urlTarget1Found).To(BeTrue())
		})

		It("should trigger func on set", func() {
			url := "http://domain.com/SetOnDownloaded/trigger/on/set"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, "foo/bar"))

			c := New(http.DefaultClient)
			c.Download(url)

			time.Sleep(sleepTime)
			urlFound := false
			c.SetOnDownloaded(func(d *Downloaded) {
				if d.BaseURL.String() == url {
					urlFound = true
				}
			})
			time.Sleep(sleepTime)

			Expect(urlFound).To(BeTrue())
		})
	})

	Describe("SetOnURLShouldQueue", func() {
		It("should download link except one", func() {
			url := "http://domain.com/SetOnURLShouldQueue/download/except/one"
			urlDownload := "http://domain.com/SetOnURLShouldQueue/download"
			urlNotDownload := "http://domain.com/SetOnURLShouldQueue/not/download"
			html := newHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Text</a>"+
				"<a href=\"%s\">Text</a>", urlDownload, urlNotDownload))
			httpmock.RegisterResponder("GET", url, newHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDownload, httpmock.NewStringResponder(200, "foo/bar"))
			httpmock.RegisterResponder("GET", urlNotDownload, httpmock.NewStringResponder(200, "foo/bar"))

			c := New(http.DefaultClient)
			urlNotDownloadFound := false
			c.SetOnURLShouldQueue(func(u *neturl.URL) bool {
				if u.String() == urlNotDownload {
					urlNotDownloadFound = true
					return false
				}

				return true
			})
			c.Download(url)

			downloaded := c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			time.Sleep(sleepTime)
			downloaded = c.NextOrNil()
			Expect(downloaded.BaseURL.String()).To(Equal(urlDownload))

			time.Sleep(sleepTime)
			downloaded = c.NextOrNil()
			Expect(downloaded).To(BeNil())

			Expect(urlNotDownloadFound).To(BeTrue())
		})
	})

	Describe("WorkersRunning", func() {
		It("should work before start", func() {
			c := New(nil)
			Expect(c.IsWorkersRunning()).To(BeFalse())
		})

		It("should work after start", func() {
			c := New(nil)
			c.Start()
			Expect(c.IsWorkersRunning()).To(BeTrue())
		})
	})

})
