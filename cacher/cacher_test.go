package cacher_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	. "github.com/daohoangson/go-sitemirror/cacher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpCacher", func() {
	tmpDir := os.TempDir()
	rootPath := path.Join(tmpDir, "_TestHttpCacher_")

	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	var newHttpCacherWithRootPath = func() Cacher {
		c := NewHttpCacher(logger)
		c.SetPath(rootPath)

		return c
	}

	BeforeEach(func() {
		os.Mkdir(rootPath, os.ModePerm)
	})

	AfterEach(func() {
		os.RemoveAll(rootPath)
	})

	It("should use working directory as default path", func() {
		c := NewHttpCacher(nil)
		wd, _ := os.Getwd()

		Expect(c.GetPath()).To(Equal(wd))
	})

	It("should return cacher mode", func() {
		c := newHttpCacherWithRootPath()

		Expect(c.GetMode()).To(Equal(HttpMode))
	})

	It("should set path", func() {
		c := NewHttpCacher(logger)
		c.SetPath(rootPath)

		Expect(c.GetPath()).To(Equal(rootPath))
	})

	Describe("CheckCacheExists", func() {
		It("should report cache exists", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/exists")
			cachePath := GenerateCachePath(rootPath, url)
			cacheDir, _ := path.Split(cachePath)
			os.MkdirAll(cacheDir, os.ModePerm)
			f, _ := os.Create(cachePath)
			f.Close()

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeTrue())
		})

		It("should report cache not exists", func() {
			url, _ := url.Parse("http://domain.com/cacher/check/cache/not/exists")

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeFalse())
		})

		It("should report cache not exists (dir as file)", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/not/write/dir/as/file")
			cachePath := GenerateCachePath(rootPath, url)
			cacheDir, _ := path.Split(cachePath)
			cacheDirParent := path.Dir(path.Dir(cacheDir))
			os.MkdirAll(cacheDirParent, os.ModePerm)
			cacheDirAsFile, _ := os.Create(path.Join(cacheDirParent, path.Base(cacheDir)))
			cacheDirAsFile.Close()

			c := newHttpCacherWithRootPath()

			Expect(c.CheckCacheExists(url)).To(BeFalse())
		})
	})

	Describe("Write", func() {
		It("should write", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/write")
			input := &Input{URL: url, StatusCode: 200}
			cachePath := GenerateCachePath(rootPath, input.URL)

			c := newHttpCacherWithRootPath()
			c.Write(input)

			written, _ := ioutil.ReadFile(cachePath)
			Expect(string(written)).To(HavePrefix(fmt.Sprintf(
				"HTTP %d\n%s: %s\n",
				input.StatusCode,
				HTTPHeaderURL,
				input.URL.String(),
			)))
		})

		It("should not write (dir as file)", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/not/write/dir/as/file")
			input := &Input{URL: url, StatusCode: 200}
			cachePath := GenerateCachePath(rootPath, input.URL)
			cacheDir, _ := path.Split(cachePath)
			cacheDirParent := path.Dir(path.Dir(cacheDir))
			os.MkdirAll(cacheDirParent, os.ModePerm)
			cacheDirAsFile, _ := os.Create(path.Join(cacheDirParent, path.Base(cacheDir)))
			cacheDirAsFile.Close()

			c := newHttpCacherWithRootPath()

			writerError := c.Write(input)
			Expect(writerError).To(HaveOccurred())

			_, readError := ioutil.ReadFile(cachePath)
			Expect(readError).To(HaveOccurred())
		})

		It("should write placeholder", func() {
			url, _ := url.Parse("http://domain.com/http/cacher/write")
			cachePath := GenerateCachePath(rootPath, url)

			c := newHttpCacherWithRootPath()
			c.WritePlaceholder(url)

			written, _ := ioutil.ReadFile(cachePath)
			writtenString := string(written)
			Expect(writtenString).To(HavePrefix(fmt.Sprintf(
				"HTTP %d\n%s: %s\n",
				http.StatusNoContent,
				HTTPHeaderURL,
				url.String(),
			)))
			Expect(writtenString).To(HaveSuffix("\n\n"))
		})
	})

	Describe("Open", func() {
		It("should open without error", func() {
			url, _ := url.Parse("http://domain.com/cacher/delete/ok")
			cachePath := GenerateCachePath(rootPath, url)
			cacheDir, _ := path.Split(cachePath)
			os.MkdirAll(cacheDir, os.ModePerm)
			f1, _ := os.Create(cachePath)
			f1.Close()

			c := newHttpCacherWithRootPath()
			Expect(c.CheckCacheExists(url)).To(BeTrue())

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
