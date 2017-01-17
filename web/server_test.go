package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sort"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	. "github.com/daohoangson/go-sitemirror/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	tmpDir := os.TempDir()
	rootPath := path.Join(tmpDir, "_TestServer_")

	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	c := cacher.NewHttpCacher(logger)
	c.SetPath(rootPath)

	var newServer = func() Server {
		return NewServer(c, logger)
	}

	BeforeEach(func() {
		os.Mkdir(rootPath, os.ModePerm)
	})

	AfterEach(func() {
		os.RemoveAll(rootPath)
	})

	Describe("ListenAndServe", func() {
		It("should listen and serve", func() {
			host := "listen.and.serve.com"
			s := newServer()

			l, err := s.ListenAndServe(host, 0)
			Expect(err).ToNot(HaveOccurred())
			l.Close()
		})

		It("should response", func() {
			host := "response.com"
			s := newServer()
			s.ListenAndServe(host, 0)
			defer s.StopAll()

			port, _ := s.GetListeningPort(host)
			r, _ := http.Get(fmt.Sprintf("http://localhost:%d", port))
			Expect(r.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should not listen on invalid port", func() {
			s := newServer()
			_, err := s.ListenAndServe("not.listen.invalid.port.com", -1)

			Expect(err).To(HaveOccurred())
		})

		It("should not listen on privileged port", func() {
			s := newServer()
			_, err := s.ListenAndServe("not.listen.privileged.port.com", 80)

			Expect(err).To(HaveOccurred())
		})

		It("should not listen twice for the same host", func() {
			host := "not.listen.twice.same.host.com"
			s := newServer()

			l1, err1 := s.ListenAndServe(host, 0)
			Expect(err1).ToNot(HaveOccurred())
			defer s.StopAll()

			l2, err2 := s.ListenAndServe(host, 0)
			Expect(err2).To(HaveOccurred())
			Expect(l2).To(Equal(l1))
		})

		Describe("GetListenerPort", func() {
			It("should return port", func() {
				host := "return.port.com"
				s := newServer()

				s.ListenAndServe(host, 0)
				defer s.StopAll()

				port, _ := s.GetListeningPort(host)
				Expect(port).To(BeNumerically(">", 0))
			})

			It("should return error for unknown host", func() {
				host := "return.error.unknown.com"
				s := newServer()

				_, err := s.GetListeningPort(host)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Serve", func() {
		It("should response with 404", func() {
			s := newServer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", "/Serve/404", nil)
			s.Serve("domain.com", w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("should response with 501 (empty file -> no first line)", func() {
			urlPath := "/Serve/501"
			url, _ := url.Parse("http://domain.com" + urlPath)
			cachePath := cacher.GenerateCachePath(rootPath, url)
			cacheDir, _ := path.Split(cachePath)
			os.MkdirAll(cacheDir, os.ModePerm)
			f, _ := os.Create(cachePath)
			f.Close()

			s := newServer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", urlPath, nil)
			s.Serve("domain.com", w, req)

			Expect(w.Code).To(Equal(http.StatusNotImplemented))
		})

		It("should response on New(nil, nil)", func() {
			s := NewServer(nil, nil)
			s.GetCacher().SetPath(rootPath)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", "/new/nil/nil", nil)
			s.Serve("domain.com", w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		Describe("SetOnCacheIssue", func() {
			It("should trigger func on cache not found", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/SetOnCacheIssue/cache/not/found", nil)

				var cacheNotFoundIssue *CacheIssue
				s.SetOnCacheIssue(func(issue CacheIssue) {
					switch issue.Type {
					case CacheNotFound:
						cacheNotFoundIssue = &issue
					}
				})

				s.Serve("domain.com", w, req)

				Expect(cacheNotFoundIssue).ToNot(BeNil())
			})

			It("should trigger func on cache error", func() {
				urlPath := "/SetOnCacheIssue/cache/error"
				url, _ := url.Parse("http://domain.com" + urlPath)
				cachePath := cacher.GenerateCachePath(rootPath, url)
				cacheDir, _ := path.Split(cachePath)
				os.MkdirAll(cacheDir, os.ModePerm)
				f, _ := os.Create(cachePath)
				f.Close()

				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", urlPath, nil)

				var cacheErrorIssue *CacheIssue
				s.SetOnCacheIssue(func(issue CacheIssue) {
					switch issue.Type {
					case CacheError:
						cacheErrorIssue = &issue
					}
				})

				s.Serve("domain.com", w, req)

				Expect(cacheErrorIssue).ToNot(BeNil())
			})

			It("should trigger func on cache expired", func() {
				urlPath := "/SetOnCacheIssue/cache/expired"
				url, _ := url.Parse("http://domain.com" + urlPath)
				cachePath := cacher.GenerateCachePath(rootPath, url)
				cacheDir, _ := path.Split(cachePath)
				os.MkdirAll(cacheDir, os.ModePerm)
				f, _ := os.Create(cachePath)
				f.WriteString(fmt.Sprintf(
					"HTTP 200\n%s: %d\n\n",
					cacher.HTTPHeaderExpires,
					time.Now().Add(-1*time.Hour).Unix(),
				))
				f.Close()

				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", urlPath, nil)

				var cacheExpiredIssue *CacheIssue
				s.SetOnCacheIssue(func(issue CacheIssue) {
					switch issue.Type {
					case CacheExpired:
						cacheExpiredIssue = &issue
					}
				})

				s.Serve("domain.com", w, req)

				Expect(cacheExpiredIssue).ToNot(BeNil())
			})
		})
	})

	Describe("Stop", func() {
		Describe("StopListening", func() {
			It("should stop listening", func() {
				host := "stop.listening.com"
				s := newServer()

				s.ListenAndServe(host, 0)

				time.Sleep(101 * time.Millisecond)
				err := s.StopListening(host)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not stop listening for unknown host", func() {
				host := "not.stop.unknown.com"
				s := newServer()

				err := s.StopListening(host)
				Expect(err).To(HaveOccurred())
			})

			It("should not stop listening twice for the same host", func() {
				host := "not.stop.twice.same.host.com"
				s := newServer()

				s.ListenAndServe(host, 0)
				err1 := s.StopListening(host)
				Expect(err1).ToNot(HaveOccurred())

				err2 := s.StopListening(host)
				Expect(err2).To(HaveOccurred())
			})
		})

		Describe("StopAll", func() {
			It("should stop all", func() {
				s := newServer()

				s.ListenAndServe("stop.all.1.com", 0)
				s.ListenAndServe("stop.all.2.com", 0)
				hosts := s.StopAll()
				Expect(len(hosts)).To(Equal(2))
			})

			It("should stop all except one", func() {
				hostsGood := []string{"stop.all.except.1.com", "stop.all.except.2.com"}
				hostBad := "stop.all.except.3.com"
				s := newServer()

				s.ListenAndServe(hostsGood[0], 0)
				s.ListenAndServe(hostsGood[1], 0)
				s.ListenAndServe(hostBad, 0)

				s.StopListening(hostBad)

				hosts := s.StopAll()
				sort.Strings(hosts)
				Expect(hosts).To(Equal(hostsGood))
			})
		})
	})
})
