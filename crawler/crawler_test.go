package crawler_test

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"time"

	"github.com/Sirupsen/logrus"
	. "github.com/daohoangson/go-sitemirror/crawler"
	t "github.com/daohoangson/go-sitemirror/testing"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crawler", func() {
	const sleepTime = 10 * time.Millisecond

	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	var newCrawler = func() Crawler {
		c := New(http.DefaultClient, logger)

		return c
	}

	BeforeEach(func() {
		httpmock.Activate()
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	It("should work with init(nil, nil)", func() {
		url := "http://domain.com/crawl/init/nil"
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

		c := New(nil, nil)
		c.QueueURL(url)
		downloaded := c.Next()
		Expect(downloaded.BaseURL.String()).To(Equal(url))
	})

	It("should set auto download depth", func() {
		autoDownloadDepth := 2

		c := newCrawler()
		c.SetAutoDownloadDepth(autoDownloadDepth)

		Expect(c.GetAutoDownloadDepth()).To(Equal(autoDownloadDepth))
	})

	Describe("WorkerCount", func() {
		It("should not accept zero", func() {
			c := newCrawler()
			err := c.SetWorkerCount(0)
			Expect(err).To(HaveOccurred())
		})

		It("should not accept negative value", func() {
			c := newCrawler()
			err := c.SetWorkerCount(-1)
			Expect(err).To(HaveOccurred())
		})

		It("should not work after Start", func() {
			c := newCrawler()
			c.Start()
			err := c.SetWorkerCount(1)
			Expect(err).To(HaveOccurred())
		})

		It("should work", func() {
			workerCount := 1

			c := newCrawler()
			err := c.SetWorkerCount(workerCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.GetWorkerCount()).To(Equal(workerCount))
		})
	})

	Describe("SetOnURLShouldQueue", func() {
		It("should download link except one", func() {
			url := "http://domain.com/SetOnURLShouldQueue/download/except/one"
			urlDownload := "http://domain.com/SetOnURLShouldQueue/download"
			urlNotDownload := "http://domain.com/SetOnURLShouldQueue/not/download"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", urlDownload, urlNotDownload))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDownload, httpmock.NewStringResponder(200, "foo/bar"))
			httpmock.RegisterResponder("GET", urlNotDownload, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()
			urlNotDownloadFound := false
			c.SetOnURLShouldQueue(func(u *neturl.URL) bool {
				if u.String() == urlNotDownload {
					urlNotDownloadFound = true
					return false
				}

				return true
			})
			c.QueueURL(url)

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

	Describe("SetOnDownload", func() {
		It("should trigger func", func() {
			url := "http://domain.com/crawl/SetOnDownload"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()

			urlFound := false
			c.SetOnDownload(func(downloadURL *neturl.URL) {
				if downloadURL.String() == url {
					urlFound = true
				}
			})

			c.QueueURL(url)
			c.Next()

			Expect(urlFound).To(BeTrue())
		})
	})

	Describe("SetOnDownloaded", func() {
		It("should trigger func directly", func() {
			url := "http://domain.com/SetOnDownloaded"
			urlTarget0 := "http://domain.com/SetOnDownloaded/target/0"
			urlTarget1 := "http://domain.com/SetOnDownloaded/target/1"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", urlTarget0, urlTarget1))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlTarget0, httpmock.NewStringResponder(200, "foo/bar"))
			httpmock.RegisterResponder("GET", urlTarget1, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()

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

			c.QueueURL(url)

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

			c := newCrawler()
			c.QueueURL(url)

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

	Describe("WorkersRunning", func() {
		It("should work before start", func() {
			c := newCrawler()
			Expect(c.IsWorkersRunning()).To(BeFalse())
		})

		It("should work after start", func() {
			c := newCrawler()
			c.Start()
			Expect(c.IsWorkersRunning()).To(BeTrue())
		})
	})

	Describe("Queue", func() {
		It("should queue one url", func() {
			url := "http://domain.com/crawler/download/one"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			c := newCrawler()
			c.QueueURL(url)
			downloaded := c.Next()

			Expect(string(downloaded.BodyBytes)).To(Equal(body))

			Expect(c.GetQueuedCount()).To(Equal(1))
			Expect(c.GetDownloadedCount()).To(Equal(1))
			Expect(c.GetLinkFoundCount()).To(Equal(0))
		})

		It("should queue url + found link", func() {
			url := "http://domain.com/crawler/download/link"
			targetUrl := "http://domain.com/crawler/download/link/target"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", targetUrl, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()
			c.QueueURL(url)
			downloaded := c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded = c.Next()
			Expect(downloaded.BaseURL.String()).To(Equal(targetUrl))

			Expect(c.GetQueuedCount()).To(Equal(2))
			Expect(c.GetDownloadedCount()).To(Equal(2))
			Expect(c.GetLinkFoundCount()).To(Equal(1))
		})

		It("should queue url + found link at depth 1", func() {
			url := "http://domain.com/crawl/download/depth/1/only"
			urlDepth1 := "http://domain.com/crawl/download/depth/1/first"
			urlDepth2 := "http://domain.com/crawl/download/depth/1/second"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link depth=1</a>", urlDepth1))
			html1 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link depth=2</a>", urlDepth2))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDepth1, t.NewHtmlResponder(html1))

			c := newCrawler()
			c.QueueURL(url)
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

		It("should not queue invalid url", func() {
			c := newCrawler()
			err := c.QueueURL(t.InvalidURL)

			time.Sleep(sleepTime)
			Expect(c.GetQueuedCount()).To(Equal(0))
			Expect(err).To(HaveOccurred())
		})
	})
})
