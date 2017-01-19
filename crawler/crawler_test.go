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
	const sleepTime = 20 * time.Millisecond
	const uint64Zero = uint64(0)
	const uint64One = uint64(1)
	const uint64Two = uint64(2)

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
		c.EnqueueURL(url)
		defer c.Stop()

		downloaded, _ := c.Downloaded()
		Expect(downloaded.BaseURL.String()).To(Equal(url))
	})

	It("should set auto download depth", func() {
		autoDownloadDepth := uint64(2)

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

		It("should not work after Start", func() {
			c := newCrawler()
			c.Start()
			defer c.Stop()

			time.Sleep(sleepTime)
			err := c.SetWorkerCount(1)
			Expect(err).To(HaveOccurred())
		})

		It("should work", func() {
			workerCount := uint64One

			c := newCrawler()
			err := c.SetWorkerCount(workerCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(c.GetWorkerCount()).To(Equal(workerCount))
		})
	})

	Describe("RequestHeader", func() {
		var (
			requestHeaderKey  string
			requestHeaderVal1 string
			requestHeaderVal2 string
		)

		BeforeEach(func() {
			now := time.Now()
			requestHeaderKey = "Now"
			requestHeaderVal1 = fmt.Sprintf("%s", now)
			requestHeaderVal2 = fmt.Sprintf("%d", now.Unix())
		})

		It("should add", func() {
			c := newCrawler()
			c.AddRequestHeader(requestHeaderKey, requestHeaderVal1)
			c.AddRequestHeader(requestHeaderKey, requestHeaderVal2)

			Expect(c.GetRequestHeaderValues(requestHeaderKey)).To(Equal([]string{
				requestHeaderVal1,
				requestHeaderVal2,
			}))
		})

		It("should set", func() {
			c := newCrawler()
			c.SetRequestHeader(requestHeaderKey, requestHeaderVal1)
			c.SetRequestHeader(requestHeaderKey, requestHeaderVal2)

			Expect(c.GetRequestHeaderValues(requestHeaderKey)).To(Equal([]string{
				requestHeaderVal2,
			}))
		})

		It("should return on no header values", func() {
			c := newCrawler()
			Expect(c.GetRequestHeaderValues(requestHeaderKey)).To(BeNil())
		})

		It("should download with header", func() {
			url := "http://domain.com/RequestHeader/download/with/header"
			httpmock.RegisterResponder("GET", url, func(req *http.Request) (*http.Response, error) {
				resp := httpmock.NewStringResponse(200, req.Header.Get(requestHeaderKey))
				return resp, nil
			})

			c := newCrawler()
			c.AddRequestHeader(requestHeaderKey, requestHeaderVal1)
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(string(downloaded.BodyBytes)).To(Equal(requestHeaderVal1))
		})
	})

	Describe("SetOnURLShouldQueue", func() {
		It("should enqueue link except one", func() {
			url := "http://domain.com/SetOnURLShouldQueue/enqueue/except/one"
			urlShouldQueue := "http://domain.com/SetOnURLShouldQueue/should/queue"
			urlNotQueue := "http://domain.com/SetOnURLShouldQueue/not/queue"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", urlShouldQueue, urlNotQueue))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlShouldQueue, httpmock.NewStringResponder(200, ""))

			c := newCrawler()
			urlNotQueueFound := false
			c.SetOnURLShouldQueue(func(u *neturl.URL) bool {
				if u.String() == urlNotQueue {
					urlNotQueueFound = true
					return false
				}

				return true
			})

			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded, _ = c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(urlShouldQueue))

			time.Sleep(sleepTime)
			Expect(c.IsBusy()).To(BeFalse())

			Expect(urlNotQueueFound).To(BeTrue())
		})
	})

	Describe("SetOnURLShouldDownload", func() {
		It("should download link except one", func() {
			url := "http://domain.com/SetOnURLShouldDownload/download/except/one"
			urlDownload := "http://domain.com/SetOnURLShouldDownload/download"
			urlNotDownload := "http://domain.com/SetOnURLShouldDownload/not/download"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", urlDownload, urlNotDownload))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDownload, httpmock.NewStringResponder(200, ""))

			c := newCrawler()
			urlNotDownloadFound := false
			c.SetOnURLShouldDownload(func(u *neturl.URL) bool {
				if u.String() == urlNotDownload {
					urlNotDownloadFound = true
					return false
				}

				return true
			})

			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded, _ = c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(urlDownload))

			time.Sleep(sleepTime)
			Expect(c.IsBusy()).To(BeFalse())

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

			c.EnqueueURL(url)
			defer c.Stop()

			c.Downloaded()

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

			c.EnqueueURL(url)
			defer c.Stop()

			time.Sleep(sleepTime)
			downloaded := c.DownloadedNotBlocking()
			Expect(downloaded).To(BeNil())

			Expect(urlFound).To(BeTrue())
			Expect(urlTarget0Found).To(BeTrue())
			Expect(urlTarget1Found).To(BeTrue())
		})

		It("should trigger func on set", func() {
			url := "http://domain.com/SetOnDownloaded/trigger/on/set"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()
			c.EnqueueURL(url)
			defer c.Stop()

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

	Describe("Life Cycle", func() {
		It("should work before start", func() {
			c := newCrawler()
			Expect(c.HasStarted()).To(BeFalse())
			Expect(c.HasStopped()).To(BeFalse())
			Expect(c.IsRunning()).To(BeFalse())
			Expect(c.IsBusy()).To(BeFalse())
		})

		It("should work after start", func() {
			c := newCrawler()
			c.Start()
			defer c.Stop()

			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeTrue())
			Expect(c.HasStopped()).To(BeFalse())
			Expect(c.IsRunning()).To(BeTrue())
			Expect(c.IsBusy()).To(BeFalse())
		})

		It("should report being busy", func() {
			url1 := "http://domain.com/crawler/IsBusy/queuing/1"
			url2 := "http://domain.com/crawler/IsBusy/queuing/2"
			slowResponder := t.NewSlowResponder(sleepTime)
			httpmock.RegisterResponder("GET", url1, slowResponder)
			httpmock.RegisterResponder("GET", url2, slowResponder)

			c := newCrawler()
			c.SetWorkerCount(uint64One)

			c.EnqueueURL(url1)
			c.EnqueueURL(url2)
			defer c.Stop()
			time.Sleep(sleepTime)

			// should be busy queuing because url1 request is slow
			// therefore url2 is still in the queue
			Expect(c.IsBusy()).To(BeTrue())

			// wait for url1 request to complete, consume its result
			// in order for url2 request to start
			c.Downloaded()
			// should be busy downloading...
			Expect(c.IsBusy()).To(BeTrue())

			// consume url2 result
			c.Downloaded()
			// should no longer be busy
			Expect(c.IsBusy()).To(BeFalse())
		})

		It("should work after stop", func() {
			c := newCrawler()
			c.Start()

			time.Sleep(sleepTime)
			c.Stop()

			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeTrue())
			Expect(c.HasStopped()).To(BeTrue())
			Expect(c.IsRunning()).To(BeFalse())
			Expect(c.IsBusy()).To(BeFalse())
		})

		It("should not auto-start on stop being called", func() {
			c := newCrawler()
			c.Stop()

			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeFalse())
			Expect(c.HasStopped()).To(BeFalse())
		})

		It("should do no op on stop being called twice", func() {
			c := newCrawler()

			c.Start()
			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeTrue())
			Expect(c.HasStopped()).To(BeFalse())

			c.Stop()
			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeTrue())
			Expect(c.HasStopped()).To(BeTrue())

			c.Stop()
			time.Sleep(sleepTime)
			Expect(c.HasStarted()).To(BeTrue())
			Expect(c.HasStopped()).To(BeTrue())
		})
	})

	Describe("Enqueue", func() {
		It("should enqueue one url", func() {
			url := "http://domain.com/crawler/enqueue/one"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			c := newCrawler()
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(string(downloaded.BodyBytes)).To(Equal(body))

			Expect(c.GetEnqueuedCount()).To(Equal(uint64One))
			Expect(c.GetDownloadedCount()).To(Equal(uint64One))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64Zero))
		})

		It("should enqueue url + found link", func() {
			url := "http://domain.com/crawler/enqueue/link"
			targetUrl := "http://domain.com/crawler/enqueue/link/target"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", targetUrl, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded, _ = c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(targetUrl))

			Expect(c.GetEnqueuedCount()).To(Equal(uint64Two))
			Expect(c.GetDownloadedCount()).To(Equal(uint64Two))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64One))
		})

		It("should enqueue url + found link at depth 1", func() {
			url := "http://domain.com/crawl/enqueue/depth/1/only"
			urlDepth1 := "http://domain.com/crawl/enqueue/depth/1/first"
			urlDepth2 := "http://domain.com/crawl/enqueue/depth/1/second"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link depth=1</a>", urlDepth1))
			html1 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link depth=2</a>", urlDepth2))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlDepth1, t.NewHtmlResponder(html1))

			c := newCrawler()
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded, _ = c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(urlDepth1))

			time.Sleep(sleepTime)
			Expect(c.IsBusy()).To(BeFalse())

			Expect(c.GetEnqueuedCount()).To(Equal(uint64Two))
			Expect(c.GetDownloadedCount()).To(Equal(uint64Two))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64Two))
		})

		It("should enqueue url without found link (auto download depth = 0)", func() {
			url := "http://domain.com/crawler/enqueue/no/link"
			targetUrl := "http://domain.com/crawler/enqueue/no/link/target"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			c := newCrawler()
			c.SetAutoDownloadDepth(uint64Zero)
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			time.Sleep(sleepTime)
			Expect(c.IsBusy()).To(BeFalse())

			Expect(c.GetEnqueuedCount()).To(Equal(uint64One))
			Expect(c.GetDownloadedCount()).To(Equal(uint64One))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64One))
		})

		It("should enqueue url + found asset, but not link (auto download depth = 0)", func() {
			url := "http://domain.com/crawl/enqueue/asset/not/link"
			urlAsset := "http://domain.com/crawl/enqueue/asset"
			urlLink := "http://domain.com/crawl/enqueue/link"
			html := t.NewHtmlMarkup(fmt.Sprintf("<script src=\"%s\">"+
				"</script><a href=\"%s\">Link</a>", urlAsset, urlLink))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))
			httpmock.RegisterResponder("GET", urlAsset, httpmock.NewStringResponder(200, "foo/bar"))

			c := newCrawler()
			c.SetAutoDownloadDepth(uint64Zero)
			c.EnqueueURL(url)
			defer c.Stop()

			downloaded, _ := c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(url))

			downloaded, _ = c.Downloaded()
			Expect(downloaded.BaseURL.String()).To(Equal(urlAsset))

			time.Sleep(sleepTime)
			Expect(c.IsBusy()).To(BeFalse())

			Expect(c.GetEnqueuedCount()).To(Equal(uint64Two))
			Expect(c.GetDownloadedCount()).To(Equal(uint64Two))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64Two))
		})

		It("should not enqueue invalid url", func() {
			c := newCrawler()
			err := c.EnqueueURL(t.InvalidURL)
			defer c.Stop()

			time.Sleep(sleepTime)
			Expect(c.GetEnqueuedCount()).To(Equal(uint64Zero))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("should download", func() {
			url := "http://domain.com/crawler/download"
			parsedURL, _ := neturl.Parse(url)
			statusCode := http.StatusOK
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(statusCode, ""))

			c := newCrawler()
			downloaded := c.Download(QueueItem{URL: parsedURL})

			Expect(downloaded.StatusCode).To(Equal(statusCode))
			Expect(c.HasStarted()).To(BeFalse())
			Expect(c.GetEnqueuedCount()).To(Equal(uint64Zero))
			Expect(c.GetDownloadedCount()).To(Equal(uint64One))
			Expect(c.GetLinkFoundCount()).To(Equal(uint64Zero))
		})

		It("should skip downloading", func() {
			url := "http://domain.com/crawler/download"
			parsedURL, _ := neturl.Parse(url)
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(http.StatusOK, ""))

			c := newCrawler()
			c.SetOnURLShouldDownload(func(_ *neturl.URL) bool {
				return false
			})
			downloaded := c.Download(QueueItem{URL: parsedURL})

			Expect(downloaded).To(BeNil())
		})

		It("should force downloading", func() {
			url := "http://domain.com/crawler/download"
			parsedURL, _ := neturl.Parse(url)
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(http.StatusOK, ""))

			c := newCrawler()
			c.SetOnURLShouldDownload(func(_ *neturl.URL) bool {
				return false
			})
			downloaded := c.Download(QueueItem{
				URL:           parsedURL,
				ForceDownload: true,
			})

			Expect(downloaded).ToNot(BeNil())
		})
	})
})
