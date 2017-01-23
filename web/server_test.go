package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sort"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	t "github.com/daohoangson/go-sitemirror/testing"
	. "github.com/daohoangson/go-sitemirror/web"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	const rootPath = "/Server/Tests"
	var fs cacher.Fs
	var c cacher.Cacher

	var newServer = func() Server {
		c = cacher.NewHTTPCacher(fs, t.Logger())
		c.SetPath(rootPath)

		return NewServer(c, t.Logger())
	}

	BeforeEach(func() {
		fs = t.NewFs()
		fs.MkdirAll(rootPath, 0777)
	})

	It("should work with init(nil, nil)", func() {
		e := NewServer(nil, nil)

		Expect(e.GetCacher()).ToNot(BeNil())
	})

	Describe("ListenAndServe", func() {
		It("should listen and serve", func() {
			root, _ := url.Parse("http://listen.and.serve.com")
			s := newServer()

			l, err := s.ListenAndServe(root, 0)
			Expect(err).ToNot(HaveOccurred())
			l.Close()
		})

		It("should listen and serve cross-host", func() {
			s := newServer()

			l, err := s.ListenAndServe(nil, 0)
			Expect(err).ToNot(HaveOccurred())
			l.Close()
		})

		It("should response", func() {
			root, _ := url.Parse("http://response.com")
			s := newServer()
			s.ListenAndServe(root, 0)
			defer s.Stop()

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

			_, err1 := s.ListenAndServe(root, 0)
			Expect(err1).ToNot(HaveOccurred())
			defer s.Stop()

			_, err2 := s.ListenAndServe(root, 0)
			Expect(err2).To(HaveOccurred())
		})

		Describe("GetListenerPort", func() {
			It("should return port", func() {
				root, _ := url.Parse("http://return.port.com")
				s := newServer()

				s.ListenAndServe(root, 0)
				defer s.Stop()

				port, _ := s.GetListeningPort(root.Host)
				Expect(port).To(BeNumerically(">", 0))
			})

			It("should return error for unknown host", func() {
				s := newServer()
				_, err := s.GetListeningPort("return.error.unknown.com")

				Expect(err).To(HaveOccurred())
			})
		})

		Describe("listenerCloser", func() {
			It("should close", func() {
				root, _ := url.Parse("http://listenerClose.should.close/")
				s := newServer()
				l, _ := s.ListenAndServe(root, 0)

				err := l.Close()

				Expect(err).ToNot(HaveOccurred())
			})

			It("should not panic on Close() being called twice", func() {
				root, _ := url.Parse("http://listenerClose.close.twice/")
				s := newServer()
				l, _ := s.ListenAndServe(root, 0)

				err1 := l.Close()
				Expect(err1).ToNot(HaveOccurred())

				err2 := l.Close()
				Expect(err2).ToNot(HaveOccurred())
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
			cachePath := cacher.GenerateHTTPCachePath(rootPath, url)
			cacheDir, _ := path.Split(cachePath)
			fs.MkdirAll(cacheDir, 0777)
			f, _ := t.FsCreate(fs, cachePath)
			f.Close()

			s := newServer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", urlPath, nil)
			s.Serve(url, w, req)

			Expect(w.Code).To(Equal(http.StatusNotImplemented))
		})

		It("should default http scheme", func() {
			root, _ := url.Parse("//domain.com")
			s := newServer()
			w := httptest.NewRecorder()
			req := httptest.NewRequest("", "/Serve/default/http", nil)
			s.Serve(root, w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		Context("cross-host", func() {
			It("should response", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/http/domain.com/Serve/cross/host/response", nil)
				s.Serve(nil, w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})

			It("should redirect domain root request", func() {
				path := "/http/domain.com"
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", path, nil)
				s.Serve(nil, w, req)

				Expect(w.Code).To(Equal(http.StatusMovedPermanently))
				Expect(w.Header().Get(cacher.HeaderLocation)).To(Equal(path + "/"))
			})

			It("should response error (no scheme, no host)", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/", nil)
				s.Serve(nil, w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("should response error (no host)", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/http/", nil)
				s.Serve(nil, w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("should response error (bad scheme)", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/ftp/domain.com/bad/scheme", nil)
				s.Serve(nil, w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Describe("SetOnServerIssue", func() {
			It("should trigger func on method not allowed", func() {
				root, _ := url.Parse("http://domain.com")
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/SetOnServerIssue/method/not/allowed", nil)

				var methodNotAllowedIssue *ServerIssue
				s.SetOnServerIssue(func(issue *ServerIssue) {
					switch issue.Type {
					case MethodNotAllowed:
						methodNotAllowedIssue = issue
					}
				})

				s.Serve(root, w, req)

				Expect(methodNotAllowedIssue).ToNot(BeNil())
			})

			It("should trigger func on cache not found", func() {
				root, _ := url.Parse("http://domain.com")
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", "/SetOnServerIssue/cache/not/found", nil)

				var cacheNotFoundIssue *ServerIssue
				s.SetOnServerIssue(func(issue *ServerIssue) {
					switch issue.Type {
					case CacheNotFound:
						cacheNotFoundIssue = issue
					}
				})

				s.Serve(root, w, req)

				Expect(cacheNotFoundIssue).ToNot(BeNil())
			})

			It("should trigger func on cache error", func() {
				urlPath := "/SetOnServerIssue/cache/error"
				url, _ := url.Parse("http://domain.com" + urlPath)
				cachePath := cacher.GenerateHTTPCachePath(rootPath, url)
				cacheDir, _ := path.Split(cachePath)
				fs.MkdirAll(cacheDir, 0777)
				f, _ := t.FsCreate(fs, cachePath)
				f.Close()

				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", urlPath, nil)

				var cacheErrorIssue *ServerIssue
				s.SetOnServerIssue(func(issue *ServerIssue) {
					switch issue.Type {
					case CacheError:
						cacheErrorIssue = issue
					}
				})

				s.Serve(url, w, req)

				Expect(cacheErrorIssue).ToNot(BeNil())
			})

			It("should trigger func on cache expired", func() {
				urlPath := "/SetOnServerIssue/cache/expired"
				url, _ := url.Parse("http://domain.com" + urlPath)
				cachePath := cacher.GenerateHTTPCachePath(rootPath, url)
				cacheDir, _ := path.Split(cachePath)
				fs.MkdirAll(cacheDir, 0777)
				f, _ := t.FsCreate(fs, cachePath)
				f.Write([]byte(fmt.Sprintf(
					"HTTP 200\n%s: %d\n\n",
					cacher.CustomHeaderExpires,
					time.Now().Add(-1*time.Hour).UnixNano(),
				)))
				f.Close()

				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("", urlPath, nil)

				var cacheExpiredIssue *ServerIssue
				s.SetOnServerIssue(func(issue *ServerIssue) {
					switch issue.Type {
					case CacheExpired:
						cacheExpiredIssue = issue
					}
				})

				s.Serve(url, w, req)

				Expect(cacheExpiredIssue).ToNot(BeNil())
			})

			It("should trigger func on cross host invalid path", func() {
				s := newServer()
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/", nil)

				var crossHostInvalidPathIssue *ServerIssue
				s.SetOnServerIssue(func(issue *ServerIssue) {
					switch issue.Type {
					case CrossHostInvalidPath:
						crossHostInvalidPathIssue = issue
					}
				})

				s.Serve(nil, w, req)

				Expect(crossHostInvalidPathIssue).ToNot(BeNil())
			})
		})
	})

	Describe("Stop", func() {
		It("should stop all", func() {
			root1, _ := url.Parse("http://stop.all.one.com")
			root2, _ := url.Parse("http://stop.all.two.com")
			s := newServer()

			s.ListenAndServe(root1, 0)
			s.ListenAndServe(root2, 0)

			hosts := s.Stop()
			Expect(len(hosts)).To(Equal(2))
		})

		It("should stop all except one", func() {
			root1, _ := url.Parse("http://stop.all.except.one.com")
			root2, _ := url.Parse("http://stop.all.except.two.com")
			root3, _ := url.Parse("http://stop.all.except.three.com")
			s := newServer()

			s.ListenAndServe(root1, 0)
			l2, _ := s.ListenAndServe(root2, 0)
			s.ListenAndServe(root3, 0)

			l2.Close()

			hosts := s.Stop()
			sort.Strings(hosts)
			Expect(hosts).To(Equal([]string{root1.Host, root3.Host}))
		})

		It("should do no op on stop being called twice", func() {
			s := newServer()
			s.Stop()
			s.Stop()
		})
	})
})
