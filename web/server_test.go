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
			root, _ := url.Parse("http://listen.and.serve.com")
			s := newServer()

			l, err := s.ListenAndServe(root, 0)
			Expect(err).ToNot(HaveOccurred())
			l.Close()
		})

		It("should response", func() {
			root, _ := url.Parse("http://response.com")
			s := newServer()
			s.ListenAndServe(root, 0)
			defer s.StopAll()

			port, _ := s.GetListeningPort(root.Host)
			r, _ := http.Get(fmt.Sprintf("http://localhost:%d", port))
			Expect(r.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should not listen on invalid port", func() {
			root, _ := url.Parse("http://not.listen.invalid.port.com")
			s := newServer()
			_, err := s.ListenAndServe(root, -1)

			Expect(err).To(HaveOccurred())
		})

		It("should not listen on privileged port", func() {
			root, _ := url.Parse("http://not.listen.privileged.port.com")
			s := newServer()
			_, err := s.ListenAndServe(root, 80)

			Expect(err).To(HaveOccurred())
		})

		It("should not listen twice for the same host", func() {
			root, _ := url.Parse("http://not.listen.twice.same.host.com")
			s := newServer()

			l1, err1 := s.ListenAndServe(root, 0)
			Expect(err1).ToNot(HaveOccurred())
			defer s.StopAll()

			l2, err2 := s.ListenAndServe(root, 0)
			Expect(err2).To(HaveOccurred())
			Expect(l2).To(Equal(l1))
		})

		Describe("GetListenerPort", func() {
			It("should return port", func() {
				root, _ := url.Parse("http://return.port.com")
				s := newServer()

				s.ListenAndServe(root, 0)
				defer s.StopAll()

				port, _ := s.GetListeningPort(root.Host)
				Expect(port).To(BeNumerically(">", 0))
			})

			It("should return error for unknown host", func() {
				s := newServer()
				_, err := s.GetListeningPort("return.error.unknown.com")

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Serve", func() {
		It("should response with 404", func() {
			root, _ := url.Parse("http://domain.com")
			s := newServer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", "/Serve/404", nil)
			s.Serve(root, w, req)

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
			s.Serve(url, w, req)

			Expect(w.Code).To(Equal(http.StatusNotImplemented))
		})

		It("should response on New(nil, nil)", func() {
			root, _ := url.Parse("http://domain.com")
			s := NewServer(nil, nil)
			s.GetCacher().SetPath(rootPath)
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", "/new/nil/nil", nil)
			s.Serve(root, w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		Describe("SetOnCacheIssue", func() {
			It("should trigger func on cache not found", func() {
				root, _ := url.Parse("http://domain.com")
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

				s.Serve(root, w, req)

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

				s.Serve(url, w, req)

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
					time.Now().Add(-1*time.Hour).UnixNano(),
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

				s.Serve(url, w, req)

				Expect(cacheExpiredIssue).ToNot(BeNil())
			})
		})
	})

	Describe("Stop", func() {
		Describe("StopListening", func() {
			It("should stop listening", func() {
				root, _ := url.Parse("http://stop.listening.com")
				s := newServer()

				s.ListenAndServe(root, 0)

				time.Sleep(101 * time.Millisecond)
				err := s.StopListening(root.Host)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not stop listening for unknown host", func() {
				s := newServer()
				err := s.StopListening("not.stop.unknown.com")

				Expect(err).To(HaveOccurred())
			})

			It("should not stop listening twice for the same host", func() {
				root, _ := url.Parse("http://not.stop.twice.same.host.com")
				s := newServer()

				s.ListenAndServe(root, 0)
				err1 := s.StopListening(root.Host)
				Expect(err1).ToNot(HaveOccurred())

				err2 := s.StopListening(root.Host)
				Expect(err2).To(HaveOccurred())
			})
		})

		Describe("StopAll", func() {
			It("should stop all", func() {
				root1, _ := url.Parse("http://stop.all.one.com")
				root2, _ := url.Parse("http://stop.all.two.com")
				s := newServer()

				s.ListenAndServe(root1, 0)
				s.ListenAndServe(root2, 0)

				hosts := s.StopAll()
				Expect(len(hosts)).To(Equal(2))
			})

			It("should stop all except one", func() {
				root1, _ := url.Parse("http://stop.all.except.one.com")
				root2, _ := url.Parse("http://stop.all.except.two.com")
				root3, _ := url.Parse("http://stop.all.except.three.com")
				s := newServer()

				s.ListenAndServe(root1, 0)
				s.ListenAndServe(root2, 0)
				s.ListenAndServe(root3, 0)

				s.StopListening(root2.Host)

				hosts := s.StopAll()
				sort.Strings(hosts)
				Expect(hosts).To(Equal([]string{root1.Host, root3.Host}))
			})
		})
	})
})
