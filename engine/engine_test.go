package engine_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	. "github.com/daohoangson/go-sitemirror/engine"
	t "github.com/daohoangson/go-sitemirror/testing"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Engine", func() {
	const rootPath = "/Engine/Tests"
	const sleepTime = 5 * time.Millisecond
	const uint64Zero = uint64(0)
	const uint64One = uint64(1)
	const uint64Two = uint64(2)
	const uint64Three = uint64(3)

	var fs cacher.Fs

	var newEngine = func() Engine {
		e := New(fs, http.DefaultClient, t.Logger())
		e.GetCacher().SetPath(rootPath)

		return e
	}

	var mirrorURL = func(e Engine, url string, port int) error {
		parsedURL, err := neturl.Parse(url)
		Expect(err).ToNot(HaveOccurred())

		return e.Mirror(parsedURL, port)
	}

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.RegisterNoResponder(httpmock.InitialTransport.RoundTrip)

		fs = t.NewFs()
		fs.MkdirAll(rootPath, 0777)
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	It("should work with init(nil, nil, nil)", func() {
		New(nil, nil, nil)
	})

	Describe("Mirror", func() {
		It("should download", func() {
			url := "http://domain.com/engine/mirror/download/url"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

			e := newEngine()

			mirrorURL(e, url, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
		})

		It("should listen and serve", func() {
			urlPath := "/engine/mirror/listen/and/serve"
			url := "http://domain.com" + urlPath
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

			e := newEngine()

			mirrorURL(e, url, 0)
			defer e.Stop()

			time.Sleep(sleepTime)
			port, _ := e.GetServer().GetListeningPort("domain.com")
			r, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
			Expect(r.StatusCode).To(Equal(http.StatusOK))
		})

		Context("Downloaded", func() {
			It("should write cache", func() {
				url := "http://domain.com/engine/mirror/download/downloaded/write"
				parsedURL, _ := neturl.Parse(url)
				httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(http.StatusOK, ""))

				e := newEngine()

				mirrorURL(e, url, -1)
				defer e.Stop()

				time.Sleep(sleepTime)
				f, err := e.GetCacher().Open(parsedURL)
				Expect(err).ToNot(HaveOccurred())
				f.Close()
			})

			It("should not overwrite cache with bad data", func() {
				url := "http://domain.com/engine/mirror/download/downloaded/not/overwrite"
				parsedURL, _ := neturl.Parse(url)
				httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(http.StatusInternalServerError, ""))
				cacherInput := &cacher.Input{
					URL:        parsedURL,
					StatusCode: http.StatusOK,
					Body:       "foo/bar",
				}

				e := newEngine()
				e.GetCacher().Write(cacherInput)
				e.GetCrawler().SetOnURLShouldDownload(func(_ *neturl.URL) bool {
					return true
				})

				e.GetCrawler().Download(crawler.QueueItem{URL: parsedURL})
				defer e.Stop()

				time.Sleep(sleepTime)
				f, _ := e.GetCacher().Open(parsedURL)
				defer f.Close()
				written, _ := ioutil.ReadAll(f)
				Expect(string(written)).To(HavePrefix("HTTP 200\n"))
			})
		})

		Context("Crawler cache exists", func() {
			It("should not download discovered link", func() {
				urlDo := "http://domain.com/engine/mirror/cache/exists/do/download"
				urlNo := "http://no.download.com"
				html := t.NewHTMLMarkup(fmt.Sprintf(`<a href="%s">Link</a>`, urlNo))
				httpmock.RegisterResponder("GET", urlDo, t.NewHTMLResponder(html))
				parsedUrlNo, _ := neturl.Parse(urlNo)
				cachePathNo := cacher.GenerateHTTPCachePath(rootPath, parsedUrlNo)
				f, _ := cacher.CreateFile(fs, cachePathNo)
				f.Write([]byte("HTTP 200\n\n"))
				f.Close()

				e := newEngine()
				mirrorURL(e, urlDo, -1)
				defer e.Stop()

				time.Sleep(sleepTime)
				Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
				Expect(e.GetCrawler().GetEnqueuedCount()).To(Equal(uint64Two))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
			})
		})

		Context("ServerIssue", func() {
			It("should response for method not allowed", func() {
				url := "http://domain.com/engine/mirror/method/not/allowed"
				httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

				e := newEngine()

				mirrorURL(e, url, 0)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("domain.com")
				resp, _ := http.Post(fmt.Sprintf("http://localhost:%d", port), "", bytes.NewReader([]byte{}))
				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

				respBody, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				Expect(string(respBody)).To(Equal(ResponseBodyMethodNotAllowed))
			})

			It("should download for cache not found", func() {
				urlRoot := "http://domain.com"
				urlPath := "/engine/mirror/cache/not/found/should/download"
				url := urlRoot + urlPath
				httpmock.RegisterResponder("GET", urlRoot+"/", httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", url, t.NewSlowResponder(sleepTime))

				e := newEngine()
				mirrorURL(e, urlRoot+"/", 0)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("domain.com")

				respStart := time.Now()
				resp, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp.StatusCode).To(Equal(200))
				Expect(time.Since(respStart)).To(BeNumerically(">", sleepTime))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			})

			It("should download for cache error", func() {
				urlRoot := "http://domain.com"
				urlPath := "/engine/mirror/cache/error/should/download"
				urlShouldQueue := urlRoot + urlPath
				httpmock.RegisterResponder("GET", urlRoot+"/", httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", urlShouldQueue, t.NewSlowResponder(sleepTime))
				parsedUrlShouldQueue, _ := neturl.Parse(urlShouldQueue)
				cachePath := cacher.GenerateHTTPCachePath(rootPath, parsedUrlShouldQueue)
				f, _ := cacher.CreateFile(fs, cachePath)
				f.Write([]byte(strings.Repeat("0", 100)))
				f.Close()

				e := newEngine()
				mirrorURL(e, urlRoot+"/", 0)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("domain.com")

				respStart := time.Now()
				resp, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp.StatusCode).To(Equal(200))
				Expect(time.Since(respStart)).To(BeNumerically(">", sleepTime))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			})

			It("should requeue for cache expired", func() {
				urlRoot := "http://domain.com"
				urlPath := "/engine/mirror/cache/expired/should/requeue"
				urlShouldQueue := urlRoot + urlPath
				httpmock.RegisterResponder("GET", urlRoot+"/", httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", urlShouldQueue, t.NewSlowResponder(sleepTime))

				e := newEngine()
				mirrorURL(e, urlRoot+"/", 0)
				defer e.Stop()

				e.GetCacher().SetDefaultTTL(time.Millisecond)

				port, _ := e.GetServer().GetListeningPort("domain.com")

				time.Sleep(sleepTime)
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))

				resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp1.StatusCode).To(Equal(http.StatusOK))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))

				time.Sleep(sleepTime)
				resp2, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp2.StatusCode).To(Equal(http.StatusOK))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))

				time.Sleep(sleepTime)
				resp3, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp3.StatusCode).To(Equal(http.StatusOK))
				Expect(e.GetCrawler().GetDownloadedCount()).To(BeNumerically(">=", 3))
			})
		})
	})

	Describe("hostRewrites", func() {
		It("should rewrite host", func() {
			url0 := "http://domain.com/engine/download/rewrite/host/0"
			url1Path := "/engine/download/rewrite/host/1"
			url1 := "http://domain.com" + url1Path
			url1OtherDomain := "http://other.domain.com" + url1Path
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1OtherDomain))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			url1Downloaded := false
			httpmock.RegisterResponder("GET", url1, func(req *http.Request) (*http.Response, error) {
				url1Downloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})
			url1OtherDomainDownloaded := false
			httpmock.RegisterResponder("GET", url1OtherDomain, func(req *http.Request) (*http.Response, error) {
				url1OtherDomainDownloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})

			e := newEngine()
			e.AddHostRewrite("other.domain.com", "domain.com")
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().IsBusy()).To(BeFalse())
			Expect(url1Downloaded).To(BeTrue())
			Expect(url1OtherDomainDownloaded).To(BeFalse())
		})

		It("should rewrite scheme", func() {
			url0 := "http://domain.com/engine/download/rewrite/scheme/0"
			url1Path := "/engine/download/rewrite/scheme/1"
			url1 := "http://domain.com" + url1Path
			url1OtherScheme := "https://other.domain.com" + url1Path
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1OtherScheme))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			url1Downloaded := false
			httpmock.RegisterResponder("GET", url1, func(req *http.Request) (*http.Response, error) {
				url1Downloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})
			url1OtherSchemeDownloaded := false
			httpmock.RegisterResponder("GET", url1OtherScheme, func(req *http.Request) (*http.Response, error) {
				url1OtherSchemeDownloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})

			e := newEngine()
			e.AddHostRewrite("other.domain.com", "http://domain.com")
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().IsBusy()).To(BeFalse())
			Expect(url1Downloaded).To(BeTrue())
			Expect(url1OtherSchemeDownloaded).To(BeFalse())
		})

		It("should rewrite path", func() {
			url0 := "http://domain.com/engine/download/rewrite/path/0"
			url1Path := "/engine/download/rewrite/path/1"
			url1Prefix := "/prefix"
			url1 := "http://domain.com" + url1Prefix + url1Path
			url1OtherPath := "http://other.domain.com" + url1Path
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1OtherPath))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			url1Downloaded := false
			httpmock.RegisterResponder("GET", url1, func(req *http.Request) (*http.Response, error) {
				url1Downloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})
			url1OtherPathDownloaded := false
			httpmock.RegisterResponder("GET", url1OtherPath, func(req *http.Request) (*http.Response, error) {
				url1OtherPathDownloaded = true
				return httpmock.NewStringResponse(http.StatusOK, ""), nil
			})

			e := newEngine()
			e.AddHostRewrite("other.domain.com", "http://domain.com"+url1Prefix)
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().IsBusy()).To(BeFalse())
			Expect(url1Downloaded).To(BeTrue())
			Expect(url1OtherPathDownloaded).To(BeFalse())
		})
	})

	Describe("hostsWhitelist", func() {
		It("should download from whitelisted host", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain.com/engine/download/whitelisted/1"
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain.com")
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
		})

		It("should download from whitelisted hosts", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain1.com/engine/download/whitelisted/1"
			url2 := "http://domain2.com/engine/download/whitelisted/2"
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", url1, url2))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))
			httpmock.RegisterResponder("GET", url2, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain1.com")
			e.AddHostWhitelisted("domain2.com")
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Three))
		})

		It("should not download from non-whitelisted host", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain1.com/engine/download/whitelisted/1"
			html0 := t.NewHTMLMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1))
			httpmock.RegisterResponder("GET", url0, t.NewHTMLResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain.com")
			e.AddHostWhitelisted("domain.com") // try to add it twice
			mirrorURL(e, url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
		})
	})

	Describe("SetBumpTTL", func() {

		testSetBumpTTLDuration := time.Millisecond

		expectServerServe := func(e Engine, url *neturl.URL, req *http.Request, statusCode int) {
			w := httptest.NewRecorder()
			e.GetServer().Serve(url, w, req)
			ExpectWithOffset(1, w.Code).To(Equal(statusCode))
		}

		testSetBumpTTL := func(bumpTTL time.Duration) Engine {
			urlPath := fmt.Sprintf("/engine/SetBumpTTL/%s", bumpTTL)
			url := "http://domain.com" + urlPath
			parsedURL, _ := neturl.Parse(url)
			httpmock.RegisterResponder("GET", url, t.NewSlowResponder(testSetBumpTTLDuration*10))
			req := httptest.NewRequest("GET", urlPath, nil)
			ch := make(chan interface{})

			e := newEngine()
			e.SetBumpTTL(bumpTTL)

			go func() {
				// trigger the 1st request
				expectServerServe(e, parsedURL, req, http.StatusOK)
				ch <- true
			}()

			time.Sleep(2 * testSetBumpTTLDuration)
			expectServerServe(e, parsedURL, req, http.StatusNoContent)

			time.Sleep(2 * testSetBumpTTLDuration)
			expectServerServe(e, parsedURL, req, http.StatusNoContent)

			<-ch
			e.Stop()

			return e
		}

		It("should set short ttl", func() {
			e := testSetBumpTTL(3 * testSetBumpTTLDuration)
			Expect(e.GetCrawler().GetDownloadedCount()).To(BeNumerically("~", 2, 1))
		})

		It("should set long ttl", func() {
			e := testSetBumpTTL(10 * testSetBumpTTLDuration)
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
		})
	})

	Describe("SetAutoEnqueueInterval", func() {
		intervalBase := time.Millisecond
		interval := 4 * intervalBase
		testTime := 10 * intervalBase

		It("should set interval", func() {
			url := "http://domain.com/engine/SetAutoEnqueueInterval/set"
			parsedURL, _ := neturl.Parse(url)
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(http.StatusOK, ""))

			e := newEngine()
			e.SetAutoEnqueueInterval(interval)
			e.Mirror(parsedURL, -1)

			time.Sleep(testTime)
			e.Stop()

			// .Mirror enqueues the first time
			// then .autoEnqueue does it 2 more times
			Expect(e.GetCrawler().GetEnqueuedCount()).To(BeNumerically("~", 3, 1))
		})

		It("should auto enqueue all urls", func() {
			url0 := "http://domain.com/engine/SetAutoEnqueueInterval/enqueue/all/0"
			parsedURL0, _ := neturl.Parse(url0)
			url1 := "http://domain.com/engine/SetAutoEnqueueInterval/enqueue/all/1"
			parsedURL1, _ := neturl.Parse(url1)
			httpmock.RegisterResponder("GET", url0, httpmock.NewStringResponder(http.StatusOK, ""))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(http.StatusOK, ""))

			e := newEngine()
			e.SetAutoEnqueueInterval(interval)
			e.Mirror(parsedURL0, -1)
			e.Mirror(parsedURL1, -1)

			time.Sleep(testTime)
			e.Stop()

			// .Mirror enqueues the two first times
			// then .autoEnqueue does it 2 more times for each url
			Expect(e.GetCrawler().GetEnqueuedCount()).To(Equal(uint64(6)))
		})
	})

	Describe("WaitAndStop", func() {
		It("should stop crawler", func() {
			url0 := "http://domain.com/engine/WaitAndStop/0"
			url1 := "http://domain.com/engine/WaitAndStop/1"
			url2 := "http://domain.com/engine/WaitAndStop/2"
			slowResponder := t.NewSlowResponder(sleepTime)
			httpmock.RegisterResponder("GET", url0, slowResponder)
			httpmock.RegisterResponder("GET", url1, slowResponder)
			httpmock.RegisterResponder("GET", url2, slowResponder)

			e := newEngine()
			e.GetCrawler().SetWorkerCount(uint64One)
			mirrorURL(e, url0, -1)
			mirrorURL(e, url1, -1)
			mirrorURL(e, url2, -1)

			e.Stop()
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Three))

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().HasStopped()).To(BeTrue())
		})

		It("should stop without downloaded something", func() {
			url := "http://domain.com/engine/should/stop/without/downloaded"
			parsedURL, _ := neturl.Parse(url)

			e := newEngine()
			e.GetCrawler().SetOnURLShouldDownload(func(url *neturl.URL) bool {
				time.Sleep(sleepTime)
				return false
			})

			e.GetCrawler().Enqueue(crawler.QueueItem{URL: parsedURL})
			e.Stop()
		})

		It("should run without panic if being called twice", func() {
			e := newEngine()
			e.Stop()
			e.Stop()
		})
	})
})
