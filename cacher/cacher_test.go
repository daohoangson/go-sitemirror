package cacher_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	. "github.com/daohoangson/go-sitemirror/cacher"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpCacher", func() {
	tmpDir := os.TempDir()
	rootPath := path.Join(tmpDir, "_TestHttpCacher_")

	logger := t.Logger()

	var newHttpCacherWithRootPath = func() Cacher {
		c := NewHTTPCacher(logger)
		c.SetPath(rootPath)
		c.SetDefaultTTL(time.Millisecond)

		return c
	}

	BeforeEach(func() {
		os.Mkdir(rootPath, os.ModePerm)
	})

	AfterEach(func() {
		os.RemoveAll(rootPath)
	})

	It("should use working directory as default path", func() {
		c := NewHTTPCacher(nil)
		wd, _ := os.Getwd()

		Expect(c.GetPath()).To(Equal(wd))
	})

	It("should return cacher mode", func() {
		c := newHttpCacherWithRootPath()

		Expect(c.GetMode()).To(Equal(HTTPMode))
	})

	It("should set path", func() {
		c := NewHTTPCacher(logger)
		c.SetPath(rootPath)

		Expect(c.GetPath()).To(Equal(rootPath))
	})

	It("should set default ttl", func() {
		ttl := time.Hour
		c := NewHTTPCacher(logger)
		c.SetDefaultTTL(ttl)

		Expect(c.GetDefaultTTL()).To(Equal(ttl))
	})

	Describe("CheckCacheExists", func() {
		It("should report cache exists", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/exists")
			cachePath := GenerateHTTPCachePath(rootPath, url)
			f, _ := CreateFile(cachePath)
			f.WriteString("HTTP 200\n\n")
			f.Close()

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeTrue())
		})

		It("should report cache not exists (no file)", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/not/exists/no/file")

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeFalse())
		})

		It("should report cache not exists (empty file)", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/not/exists/empty/file")
			cachePath := GenerateHTTPCachePath(rootPath, url)
			f, _ := CreateFile(cachePath)
			f.Close()

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeFalse())
		})

		It("should report cache not exists (placeholder)", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/not/exists/empty/file")
			cachePath := GenerateHTTPCachePath(rootPath, url)
			f, _ := CreateFile(cachePath)
			f.WriteString("HTTP 204\n\n")
			f.Close()

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeFalse())
		})
	})

	Describe("Write", func() {

		expectPlaceholder := func(url *url.URL) {
			cachePath := GenerateHTTPCachePath(rootPath, url)
			written, _ := ioutil.ReadFile(cachePath)
			writtenString := string(written)
			Expect(writtenString).To(HavePrefix(fmt.Sprintf(
				"HTTP %d\n%s: %s\n",
				http.StatusNoContent,
				CustomHeaderURL,
				url.String(),
			)))
			Expect(writtenString).To(HaveSuffix("\n\n"))

			expiresHeaderValue := getHeaderValue(writtenString, CustomHeaderExpires)
			expiresValue, _ := strconv.ParseInt(expiresHeaderValue, 10, 64)
			Expect(expiresValue).To(BeNumerically(">", 0))
		}

		It("should write", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/write")
			input := &Input{URL: url, StatusCode: 200}
			cachePath := GenerateHTTPCachePath(rootPath, input.URL)

			c := newHttpCacherWithRootPath()
			c.Write(input)

			written, _ := ioutil.ReadFile(cachePath)
			Expect(string(written)).To(HavePrefix(fmt.Sprintf(
				"HTTP %d\n%s: %s\n",
				input.StatusCode,
				CustomHeaderURL,
				input.URL.String(),
			)))
		})

		It("should not write (dir as file)", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/not/write/dir/as/file")
			input := &Input{URL: url}
			cachePath := GenerateHTTPCachePath(rootPath, input.URL)
			cacheDir := path.Dir(cachePath)
			f, _ := CreateFile(cacheDir)
			f.Close()

			c := newHttpCacherWithRootPath()

			writeError := c.Write(input)
			Expect(writeError).To(HaveOccurred())

			_, readError := ioutil.ReadFile(cachePath)
			Expect(readError).To(HaveOccurred())
		})

		Describe("Bump", func() {
			It("should bump", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/bump")
				input := &Input{URL: url, StatusCode: 200, Body: "Hello World."}
				cachePath := GenerateHTTPCachePath(rootPath, input.URL)

				c := newHttpCacherWithRootPath()
				c.Write(input)
				written, _ := ioutil.ReadFile(cachePath)
				writtenString := string(written)
				writtenExpiresValue := getHeaderValue(writtenString, CustomHeaderExpires)
				writtenExpires, _ := strconv.ParseInt(writtenExpiresValue, 10, 64)
				ttl := time.Duration((writtenExpires-time.Now().UnixNano())*2) * time.Second

				c.Bump(url, ttl)
				bumped, _ := ioutil.ReadFile(cachePath)
				bumpedString := string(bumped)

				Expect(len(bumpedString)).To(BeNumerically(">", 0))
				Expect(bumpedString).ToNot(Equal(writtenString))

				expiresRegexp := regexp.MustCompile(fmt.Sprintf(`%s:[^\n]+\n`, CustomHeaderExpires))
				writtenWithoutExpires := expiresRegexp.ReplaceAllString(writtenString, "")
				bumpedWithoutExpires := expiresRegexp.ReplaceAllString(bumpedString, "")
				Expect(bumpedWithoutExpires).To(Equal(writtenWithoutExpires))

				bumpedExpiresValue := getHeaderValue(bumpedString, CustomHeaderExpires)
				bumpedExpires, _ := strconv.ParseInt(bumpedExpiresValue, 10, 64)
				Expect(bumpedExpires).To(BeNumerically(">", writtenExpires))
			})

			It("should write placeholder (no file)", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/bump/placeholder/no/file")
				c := newHttpCacherWithRootPath()
				c.Bump(url, time.Minute)

				expectPlaceholder(url)
			})

			It("should write placeholder (no expires header)", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/bump/placeholder/no/expires")
				cachePath := GenerateHTTPCachePath(rootPath, url)
				f, _ := CreateFile(cachePath)
				f.WriteString("\n")
				f.Close()

				c := newHttpCacherWithRootPath()
				c.Bump(url, time.Minute)

				expectPlaceholder(url)
			})

			It("should write placeholder (too short expires header)", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/bump/placeholder/no/expires")
				cachePath := GenerateHTTPCachePath(rootPath, url)
				f, _ := CreateFile(cachePath)
				f.WriteString(fmt.Sprintf("%s: 1\n", CustomHeaderExpires))
				f.Close()

				c := newHttpCacherWithRootPath()
				c.Bump(url, time.Minute)

				expectPlaceholder(url)
			})

			It("should not bump (dir as file)", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/not/bump/dir/as/file")
				input := &Input{URL: url}
				cachePath := GenerateHTTPCachePath(rootPath, input.URL)
				cacheDir := path.Dir(cachePath)
				f, _ := CreateFile(cacheDir)
				f.Close()

				c := newHttpCacherWithRootPath()

				bumpError := c.Bump(url, time.Minute)
				Expect(bumpError).To(HaveOccurred())
			})
		})

		Describe("WritePlaceholder", func() {
			It("should write", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/write/placeholder")
				c := newHttpCacherWithRootPath()
				c.WritePlaceholder(url, time.Minute)

				expectPlaceholder(url)
			})

			It("should not write (dir as file)", func() {
				url, _ := url.Parse("http://domain.com/http/cacher/not/write/placeholder/dir/as/file")
				cachePath := GenerateHTTPCachePath(rootPath, url)
				cacheDir := path.Dir(cachePath)
				f, _ := CreateFile(cacheDir)
				f.Close()

				c := newHttpCacherWithRootPath()

				writeError := c.WritePlaceholder(url, time.Minute)
				Expect(writeError).To(HaveOccurred())

				_, readError := ioutil.ReadFile(cachePath)
				Expect(readError).To(HaveOccurred())
			})
		})
	})

	Describe("Open", func() {
		It("should open without error", func() {
			url, _ := url.Parse("http://domain.com/cacher/delete/ok")
			cachePath := GenerateHTTPCachePath(rootPath, url)
			f1, _ := CreateFile(cachePath)
			f1.Close()

			c := newHttpCacherWithRootPath()
			f2, err := c.Open(url)
			Expect(err).ToNot(HaveOccurred())
			f2.Close()
		})

		It("should open with error", func() {
			url, _ := url.Parse("http://domain.com/cacher/delete/error")

			c := newHttpCacherWithRootPath()
			_, err := c.Open(url)
			Expect(err).To(HaveOccurred())
		})
	})
})
