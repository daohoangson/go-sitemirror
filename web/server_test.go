package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
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
			s := newServer()
			l, err := s.ListenAndServe("domain.com", 0)

			Expect(err).ToNot(HaveOccurred())
			l.Close()
		})

		It("should response", func() {
			s := newServer()
			l, _ := s.ListenAndServe("domain.com", 0)
			defer l.Close()

			addr := l.Addr().String()
			matches := regexp.MustCompile(`:(\d+)$`).FindStringSubmatch(addr)
			port, _ := strconv.ParseInt(matches[1], 10, 32)
			Expect(port).To(BeNumerically(">", 0))

			r, _ := http.Get(fmt.Sprintf("http://localhost:%d", port))
			Expect(r.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should not listen on privileged port", func() {
			s := newServer()
			_, err := s.ListenAndServe("domain.com", 80)

			Expect(err).To(HaveOccurred())
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
})
