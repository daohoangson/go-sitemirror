package cacher_test

import (
	"net/url"
	"os"
	"path"
	"strings"

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
		const pathSeparator = "/"

		lotsOfA := strings.Repeat("a", MaxPathNameLength+1)

		generateCachePath := func(u *url.URL) (string, string, string) {
			generated := GenerateCachePath(rootPath, u)
			dir, file := path.Split(generated)
			hash := path.Dir(dir)
			dirBeforeHash := path.Dir(hash)

			Expect(dirBeforeHash).To(HavePrefix(rootPath + pathSeparator))
			dirBeforeHashWithoutRoot := dirBeforeHash[len(rootPath)+len(pathSeparator):]

			return generated, dirBeforeHashWithoutRoot, file
		}

		expectIsHashOf := func(actual string, value string) {
			if len(value) <= MaxPathNameLength {
				Expect(actual).To(Equal(value))
			} else {
				Expect(len(actual)).To(Equal(MaxPathNameLength))

				lengthRemaining := MaxPathNameLength - ShortHashLength - 1
				Expect(actual[:lengthRemaining]).To(Equal(value[:lengthRemaining]))
			}
		}

		It("should keep url host + path", func() {
			hostAndDir := "domain.com/fileop/keep/host/path"
			file := "file"
			url, _ := url.Parse("http://" + hostAndDir + "/" + file)
			_, gDir, gFile := generateCachePath(url)

			Expect(gDir).To(Equal(hostAndDir))
			Expect(gFile).To(Equal(file))
		})

		It("should use hash for long host", func() {
			host := lotsOfA + ".com"
			dir := "fileop/hash/for/long/host"
			file := "file"
			url, _ := url.Parse("http://" + host + "/" + dir + "/" + file)
			_, gDir, _ := generateCachePath(url)

			gDirParts := strings.Split(gDir, pathSeparator)
			expectIsHashOf(gDirParts[0], host)
		})

		It("should use hash for long file", func() {
			hostAndDir := "domain.com/fileop/hash/long/file"
			file := "file" + lotsOfA
			url, _ := url.Parse("http://" + hostAndDir + "/" + file)
			_, _, gFile := generateCachePath(url)

			expectIsHashOf(gFile, file)
		})

		It("should use hash for no file", func() {
			hostAndDir := "domain.com/fileop/hash/no/file/"
			url, _ := url.Parse("http://" + hostAndDir)
			g, _, _ := generateCachePath(url)

			gDir, _ := path.Split(g)
			Expect(gDir).To(Equal(rootPath + "/" + hostAndDir))
		})

		It("should keep query", func() {
			hostAndDir := "domain.com/fileop/keep/query"
			file := "file"
			query := "foo=bar"
			url, _ := url.Parse("http://" + hostAndDir + "/" + file + "?" + query)
			_, gDir, _ := generateCachePath(url)

			Expect(gDir).To(Equal(hostAndDir + "/" + query))
		})

		It("should use hash for long query", func() {
			hostAndDir := "domain.com/fileop/hash/long/query"
			file := "file"
			query := "foo=" + strings.Repeat("a", 100)
			url, _ := url.Parse("http://" + hostAndDir + "/" + file + "?" + query)
			_, gDir, _ := generateCachePath(url)

			gDirParts := strings.Split(gDir, pathSeparator)
			queryPart := gDirParts[len(gDirParts)-1]
			expectIsHashOf(queryPart, query)
		})

		It("should remove slashes from query", func() {
			hostAndDir := "domain.com/fileop/remove/slashes/query"
			file := "file"
			query := "foo=b/a%2Fr"
			url, _ := url.Parse("http://" + hostAndDir + "/" + file + "?" + query)
			_, gDir, _ := generateCachePath(url)

			Expect(gDir).To(Equal(hostAndDir + "/foo=bar"))
		})

		It("should generate different path for slashes", func() {
			url0, _ := url.Parse("http://domain.com/fileop/diff/path/slashes")
			url1, _ := url.Parse("http://domain.com/fileop/diff/path/slashes/")
			path0 := GenerateCachePath(rootPath, url0)
			path1 := GenerateCachePath(rootPath, url1)

			Expect(path0).ToNot(Equal(path1))
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

	Describe("GetSafePathName", func() {
		It("should remove slash", func() {
			name := "a/b/c"
			safe := GetSafePathName(name)

			Expect(safe).To(Equal("abc"))
		})

		It("should remove unicode", func() {
			name := "a\u03b2c"
			safe := GetSafePathName(name)

			Expect(safe).To(Equal("ac"))
		})

		It("should return something for string with all characters invalid", func() {
			name := "\u03b1\u03b2\u03b3"
			safe := GetSafePathName(name)

			Expect(len(safe)).ToNot(Equal(0))
		})

		It("should return shortened name when name is too long", func() {
			name := strings.Repeat("a", 100)
			safe := GetSafePathName(name)

			Expect(len(safe)).To(BeNumerically("<", len(name)))
		})
	})
})
