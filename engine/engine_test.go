package engine_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	. "github.com/daohoangson/go-sitemirror/engine"
	t "github.com/daohoangson/go-sitemirror/testing"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Engine", func() {
	const sleepTime = 20 * time.Millisecond
	const uint64Zero = uint64(0)
	const uint64One = uint64(1)
	const uint64Two = uint64(2)
	const uint64Three = uint64(3)

	tmpDir := os.TempDir()
	rootPath := path.Join(tmpDir, "_TestEngine_")

	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	var newEngine = func() Engine {
		e := New(http.DefaultClient, logger)
		e.GetCacher().SetPath(rootPath)

		return e
	}

	BeforeEach(func() {
		httpmock.Activate()
		httpmock.RegisterNoResponder(httpmock.InitialTransport.RoundTrip)
		os.Mkdir(rootPath, os.ModePerm)
	})

	AfterEach(func() {
		os.RemoveAll(rootPath)
		httpmock.DeactivateAndReset()
	})

	It("should work with init(nil, nil)", func() {
		url := "http://domain.com/engine/init/nil"
		httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

		e := New(nil, nil)
		e.GetCacher().SetPath(rootPath)
		e.MirrorURL(url, -1)
		defer e.Stop()

		time.Sleep(sleepTime)
		Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
	})

	Describe("Mirror", func() {
		It("should download", func() {
			url := "http://domain.com/engine/mirror/download/url"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

			e := newEngine()

			e.MirrorURL(url, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
		})

		It("should download url.URL", func() {
			url := "http://domain.com/engine/mirror/download/url"
			body := "body1"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			e := newEngine()

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			e.Mirror(parsedURL, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
		})

		It("should not queue invalid url", func() {
			e := newEngine()
			err := e.MirrorURL(t.InvalidURL, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetEnqueuedCount()).To(Equal(uint64Zero))
			Expect(err).To(HaveOccurred())
		})

		It("should listen and serve", func() {
			urlPath := "/engine/mirror/listen/and/serve"
			url := "http://domain.com" + urlPath
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

			e := newEngine()

			e.MirrorURL(url, 0)
			defer e.Stop()

			time.Sleep(sleepTime)
			port, _ := e.GetServer().GetListeningPort("domain.com")
			r, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
			Expect(r.StatusCode).To(Equal(http.StatusOK))
		})

		Context("Crawler cache exists", func() {
			It("should not download discovered link", func() {
				urlDo := "http://domain.com/engine/mirror/cache/exists/do/download"
				urlNo := "http://no.download.com"
				html := t.NewHtmlMarkup(fmt.Sprintf(`<a href="%s">Link</a>`, urlNo))
				httpmock.RegisterResponder("GET", urlDo, t.NewHtmlResponder(html))
				parsedUrlNo, _ := neturl.Parse(urlNo)
				cachePathNo := cacher.GenerateCachePath(rootPath, parsedUrlNo)
				f, _ := cacher.CreateFile(cachePathNo)
				f.WriteString("HTTP 200\n\n")
				f.Close()

				e := newEngine()
				e.MirrorURL(urlDo, -1)
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

				e.MirrorURL(url, 0)
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
				e.MirrorURL(urlRoot+"/", 0)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("domain.com")

				time.Sleep(sleepTime)
				respRootStart := time.Now()
				respRoot, _ := http.Get(fmt.Sprintf("http://localhost:%d", port))
				Expect(respRoot.StatusCode).To(Equal(200))
				Expect(time.Since(respRootStart)).To(BeNumerically("<", 5*time.Millisecond))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))

				respStart := time.Now()
				resp, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp.StatusCode).To(Equal(200))
				Expect(time.Since(respStart)).To(BeNumerically(">", sleepTime))
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			})

			It("should queue for cache error", func() {
				urlRoot := "http://domain.com"
				urlPath := "/engine/mirror/cache/error/should/queue"
				urlShouldQueue := urlRoot + urlPath
				httpmock.RegisterResponder("GET", urlRoot+"/", httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", urlShouldQueue, t.NewSlowResponder(sleepTime))
				parsedUrlShouldQueue, _ := neturl.Parse(urlShouldQueue)
				cachePath := cacher.GenerateCachePath(rootPath, parsedUrlShouldQueue)
				f, _ := cacher.CreateFile(cachePath)
				f.WriteString(strings.Repeat("0", 100))
				f.Close()

				e := newEngine()
				e.MirrorURL(urlRoot+"/", 0)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("domain.com")

				resp1, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp1.StatusCode).To(Equal(http.StatusNotImplemented))

				resp2, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp2.StatusCode).To(Equal(http.StatusNoContent))

				time.Sleep(sleepTime * 2)
				resp3, _ := http.Get(fmt.Sprintf("http://localhost:%d"+urlPath, port))
				Expect(resp3.StatusCode).To(Equal(http.StatusOK))
			})

			It("should requeue for cache expired", func() {
				urlRoot := "http://domain.com"
				urlPath := "/engine/mirror/cache/expired/should/requeue"
				urlShouldQueue := urlRoot + urlPath
				httpmock.RegisterResponder("GET", urlRoot+"/", httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", urlShouldQueue, t.NewSlowResponder(sleepTime))

				e := newEngine()
				e.MirrorURL(urlRoot+"/", 0)
				defer e.Stop()

				e.GetCacher().SetDefaultTTL(time.Millisecond)
				e.SetBumpTTL(time.Millisecond)

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
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Three))
			})
		})
	})

	Describe("hostRewrites", func() {
		It("should rewrite", func() {
			url0 := "http://domain.com/engine/download/rewrite/0"
			url1Path := "/engine/download/rewrite/1"
			url1 := "http://domain.com" + url1Path
			url1OtherDomain := "http://other.domain.com" + url1Path
			html0 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1OtherDomain))
			httpmock.RegisterResponder("GET", url0, t.NewHtmlResponder(html0))
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
			e.MirrorURL(url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().IsBusy()).To(BeFalse())
			Expect(url1Downloaded).To(BeTrue())
			Expect(url1OtherDomainDownloaded).To(BeFalse())
		})
	})

	Describe("hostsWhitelist", func() {
		It("should download from whitelisted host", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain.com/engine/download/whitelisted/1"
			html0 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1))
			httpmock.RegisterResponder("GET", url0, t.NewHtmlResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain.com")
			e.MirrorURL(url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))
		})

		It("should download from whitelisted hosts", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain1.com/engine/download/whitelisted/1"
			url2 := "http://domain2.com/engine/download/whitelisted/2"
			html0 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>"+
				"<a href=\"%s\">Link</a>", url1, url2))
			httpmock.RegisterResponder("GET", url0, t.NewHtmlResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))
			httpmock.RegisterResponder("GET", url2, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain1.com")
			e.AddHostWhitelisted("domain2.com")
			e.MirrorURL(url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64Two))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Three))
		})

		It("should not download from non-whitelisted host", func() {
			url0 := "http://domain.com/engine/download/whitelisted/0"
			url1 := "http://domain1.com/engine/download/whitelisted/1"
			html0 := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", url1))
			httpmock.RegisterResponder("GET", url0, t.NewHtmlResponder(html0))
			httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))

			e := newEngine()
			e.AddHostWhitelisted("domain.com")
			e.AddHostWhitelisted("domain.com") // try to add it twice
			e.MirrorURL(url0, -1)
			defer e.Stop()

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().GetLinkFoundCount()).To(Equal(uint64One))
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))
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
			e.MirrorURL(url0, -1)
			e.MirrorURL(url1, -1)
			e.MirrorURL(url2, -1)

			e.Stop()
			Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Three))

			time.Sleep(sleepTime)
			Expect(e.GetCrawler().HasStopped()).To(BeTrue())
		})

		It("should stop without downloaded something", func() {
			e := newEngine()
			e.GetCrawler().SetOnURLShouldDownload(func(url *neturl.URL) bool {
				time.Sleep(sleepTime)
				return false
			})
			e.GetCrawler().EnqueueURL("http://domain.com/engine/should/stop/without/downloaded")
			e.Stop()
		})

		It("should run without panic if being called twice", func() {
			e := newEngine()
			e.Stop()
			e.Stop()
		})
	})
})