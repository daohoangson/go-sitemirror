package cacher_test

import (
	"net/url"
	"os"
	"path"

	. "github.com/daohoangson/go-sitemirror/cacher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fileop", func() {
	Describe("CreateFile", func() {
		tmpDir := os.TempDir()
		rootPath := tmpDir + "/_TestCreateFile_"

		BeforeEach(func() {
			os.Mkdir(rootPath, os.ModePerm)
		})

		AfterEach(func() {
			os.RemoveAll(rootPath)
		})

		It("should create dir", func() {
			path := path.Join(rootPath, "dir", "file")
			f, _ := CreateFile(path)
			defer f.Close()

			Expect(f.Name()).To(Equal(path))
		})

		It("should create all dirs", func() {
			path := path.Join(rootPath, "dir1", "dir2", "dir3", "file")
			f, _ := CreateFile(path)
			defer f.Close()

			Expect(f.Name()).To(Equal(path))
		})

		It("should fail if dir existed as file", func() {
			dirAsFilePath := path.Join(rootPath, "dir-as-file")
			dirAsFile, _ := os.Create(dirAsFilePath)
			dirAsFile.Close()

			path := path.Join(dirAsFilePath, "file")
			_, err := CreateFile(path)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if file existed", func() {
			path := path.Join(rootPath, "file-existed")
			f, _ := os.Create(path)
			f.Close()

			_, err := CreateFile(path)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GenerateCachePath", func() {

		const rootPath = "/GenerateCachePath"

		It("should keep url scheme + host + path", func() {
			scheme := "http"
			hostAndPath := "domain.com/fileop/keep/scheme/host/path"
			url, _ := url.Parse(scheme + "://" + hostAndPath)
			path := GenerateCachePath(rootPath, url)

			Expect(path).To(Equal(rootPath + "/" + scheme + "/" + hostAndPath))
		})

		It("should keep query", func() {
			scheme := "http"
			hostAndDir := "domain.com/fileop/keep/scheme/host/path"
			file := "file"
			query := "foo=bar"
			url, _ := url.Parse(scheme + "://" + hostAndDir + "/" + file + "?" + query)
			path := GenerateCachePath(rootPath, url)

			Expect(path).To(Equal(rootPath + "/" + scheme + "/" + hostAndDir + "/" + query + "/" + file))
		})
	})

	Describe("BuildQueryPath", func() {
		It("should sort query keys", func() {
			query := make(url.Values)
			query["a"] = []string{"1"}
			query["b"] = []string{"2"}
			query["c"] = []string{"3"}

			path := BuildQueryPath(&query)

			Expect(path).To(Equal("a=1/b=2/c=3"))
		})

		It("should sort query values", func() {
			query := make(url.Values)
			query["a"] = []string{"3", "1"}
			query["b"] = []string{"2", "4"}

			path := BuildQueryPath(&query)

			Expect(path).To(Equal("a=1/a=3/b=2/b=4"))
		})
	})
})
