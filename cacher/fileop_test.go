package cacher_test

import (
	"io/ioutil"
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
		rootPath := path.Join(tmpDir, "_TestCreateFile_")

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

		It("should write new file", func() {
			bytes := []byte{1}
			path := path.Join(rootPath, "file")
			w, _ := CreateFile(path)
			w.Write(bytes)
			w.Close()

			read, _ := ioutil.ReadFile(path)
			Expect(read).To(Equal(bytes))
		})

		It("should overwrite if file existed", func() {
			bytes1 := []byte{0, 0, 0, 0, 0, 0, 0, 1}
			bytes2 := []byte{2}
			path := path.Join(rootPath, "file-existed")
			w1, _ := os.Create(path)
			w1.Write(bytes1)
			w1.Close()

			w2, _ := CreateFile(path)
			w2.Write(bytes2)
			w2.Close()

			read, _ := ioutil.ReadFile(path)
			Expect(read).To(Equal(bytes2))
		})
	})

	Describe("OpenFile", func() {
		tmpDir := os.TempDir()
		rootPath := path.Join(tmpDir, "_TestOpenFile_")

		BeforeEach(func() {
			os.Mkdir(rootPath, os.ModePerm)
		})

		AfterEach(func() {
			os.RemoveAll(rootPath)
		})

		It("should create dir", func() {
			path := path.Join(rootPath, "dir", "file")
			f, _ := OpenFile(path)
			defer f.Close()

			Expect(f.Name()).To(Equal(path))
		})

		It("should write new file", func() {
			bytes := []byte{1}
			path := path.Join(rootPath, "file")
			w, _ := OpenFile(path)
			w.Write(bytes)
			w.Close()

			read, _ := ioutil.ReadFile(path)
			Expect(read).To(Equal(bytes))
		})

		It("should append if file existed", func() {
			bytes1 := []byte{1}
			bytes2 := []byte{2}
			path := path.Join(rootPath, "file-existed")
			w1, _ := os.Create(path)
			w1.Write(bytes1)
			w1.Close()

			w2, _ := OpenFile(path)
			w2.Seek(0, 2)
			w2.Write(bytes2)
			w2.Close()

			read, _ := ioutil.ReadFile(path)
			Expect(read[:len(bytes1)]).To(Equal(bytes1))
			Expect(read[len(bytes1):]).To(Equal(bytes2))
		})
	})

	Describe("GenerateCachePath", func() {
		const pathSeparator = "/"

		tmpDir := os.TempDir()
		rootPath := path.Join(tmpDir, "_TestGenerateCachePath_")

		lotsOfA := strings.Repeat("a", MaxPathNameLength+1)

		generateCachePath := func(u *url.URL) (string, string, string) {
			generated := GenerateCachePath(rootPath, u)
			dir, file := path.Split(generated)
			hash := path.Dir(dir)
			dirBeforeHash := path.Dir(hash)

			dirBeforeHashWithoutRoot := dirBeforeHash[len(rootPath):]
			dirBeforeHashWithoutRoot = path.Join(".", dirBeforeHashWithoutRoot)

			return generated, dirBeforeHashWithoutRoot, file
		}

		expectIsHashOf := func(actual string, value string) {
			if len(value) <= MaxPathNameLength {
				ExpectWithOffset(1, actual).To(Equal(value))
			} else {
				ExpectWithOffset(1, len(actual)).To(Equal(MaxPathNameLength))

				lengthRemaining := MaxPathNameLength - ShortHashLength - 1
				ExpectWithOffset(1, actual[:lengthRemaining]).To(Equal(value[:lengthRemaining]))
			}
		}

		It("should keep url scheme + host + path", func() {
			scheme := "http"
			hostAndDir := "domain.com/fileop/keep/host/path"
			file := "file"
			url, _ := url.Parse(scheme + "://" + hostAndDir + "/" + file)
			_, gDir, gFile := generateCachePath(url)

			Expect(gDir).To(Equal(path.Join(scheme, hostAndDir)))
			Expect(gFile).To(Equal(file))
		})

		It("should use hash for no path", func() {
			scheme := "http"
			host := "domain.com"
			url, _ := url.Parse(scheme + "://" + host)
			g, _, file := generateCachePath(url)

			gWithoutRoot := g[len(rootPath)+1:]
			gWithoutFile := gWithoutRoot[:len(gWithoutRoot)-len(file)-1]

			Expect(gWithoutFile).To(Equal(path.Join(scheme, host)))
		})

		It("should use hash for long host", func() {
			host := lotsOfA + ".com"
			dir := "fileop/hash/for/long/host"
			file := "file"
			url, _ := url.Parse("http://" + host + "/" + dir + "/" + file)
			_, gDir, _ := generateCachePath(url)

			gDirParts := strings.Split(gDir, pathSeparator)
			expectIsHashOf(gDirParts[1], host)
		})

		It("should use hash for long file", func() {
			hostAndDir := "domain.com/fileop/hash/long/file"
			file := "file" + lotsOfA
			url, _ := url.Parse("http://" + hostAndDir + "/" + file)
			_, _, gFile := generateCachePath(url)

			expectIsHashOf(gFile, file)
		})

		It("should use hash for no file", func() {
			scheme := "http"
			hostAndDir := "domain.com/fileop/hash/no/file/"
			url, _ := url.Parse(scheme + "://" + hostAndDir)
			g, _, _ := generateCachePath(url)

			gDir, _ := path.Split(g)
			gDir = path.Dir(gDir)
			Expect(gDir).To(Equal(path.Join(rootPath, scheme, hostAndDir)))
		})

		It("should keep query", func() {
			scheme := "http"
			hostAndDir := "domain.com/fileop/keep/query"
			file := "file"
			query := "foo=bar"
			url, _ := url.Parse(scheme + "://" + hostAndDir + "/" + file + "?" + query)
			_, gDir, _ := generateCachePath(url)

			Expect(gDir).To(Equal(path.Join(scheme, hostAndDir, query)))
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
			scheme := "http"
			hostAndDir := "domain.com/fileop/remove/slashes/query"
			file := "file"
			query := "foo=b/a%2Fr"
			url, _ := url.Parse(scheme + "://" + hostAndDir + "/" + file + "?" + query)
			_, gDir, _ := generateCachePath(url)

			Expect(gDir).To(Equal(path.Join(scheme, hostAndDir, "foo=bar")))
		})

		It("should generate different path for slashes", func() {
			url0, _ := url.Parse("http://domain.com/fileop/diff/path/slashes")
			url1, _ := url.Parse("http://domain.com/fileop/diff/path/slashes/")
			path0 := GenerateCachePath(rootPath, url0)
			path1 := GenerateCachePath(rootPath, url1)

			Expect(path0).ToNot(Equal(path1))
		})

		It("should handle nil url", func() {
			GenerateCachePath(rootPath, nil)
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
