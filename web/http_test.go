package web_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/crawler"
	. "github.com/daohoangson/go-sitemirror/web"
	"github.com/daohoangson/go-sitemirror/web/internal"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP", func() {

	const int64Zero = int64(0)
	const int64One = int64(1)
	const int64Two = int64(2)

	newReader := func(s string) io.Reader {
		return bytes.NewReader([]byte(s))
	}

	newBufioReader := func(s string) *bufio.Reader {
		return bufio.NewReader(newReader(s))
	}

	Describe("ServeDownloaded", func() {
		It("should write status code, header and content", func() {
			contentType := "text/plain"
			downloaded := &crawler.Downloaded{
				StatusCode: http.StatusOK,
				Body:       "foo/bar",
			}
			downloaded.AddHeader("Content-Type", contentType)
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeDownloaded(downloaded, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(w.Code).To(Equal(downloaded.StatusCode))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("Content-Length")).To(Equal(fmt.Sprintf("%d", len(downloaded.Body))))

			wBody, _ := ioutil.ReadAll(w.Body)
			Expect(string(wBody)).To(Equal(downloaded.Body))
		})

		It("should write Location header", func() {
			location := "http://domain.com/http/ServeDownloaded/write/location/header"
			downloaded := &crawler.Downloaded{
				StatusCode: http.StatusMovedPermanently,
			}
			downloaded.AddHeader("Location", location)
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeDownloaded(downloaded, si)
			si.Flush()

			Expect(w.Header().Get("Location")).To(Equal(location))
		})
	})

	Describe("ServeHTTPCache", func() {
		It("should write status code, header and content", func() {
			statusCode := 200
			contentType := "text/plain"
			content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
			input := newReader("HTTP " + fmt.Sprintf("%d", statusCode) + "\n" +
				"Content-Type: " + contentType + "\n" +
				"Content-Length: " + fmt.Sprintf("%d", len(content)) + "\n" +
				"\n" +
				string(content))
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPCache(input, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(w.Code).To(Equal(statusCode))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("Content-Length")).To(Equal(fmt.Sprintf("%d", len(content))))

			wBody, _ := ioutil.ReadAll(w.Body)
			Expect(len(wBody)).To(Equal(len(content)))
			Expect(string(wBody)).To(Equal(string(content)))
		})

		It("should not write (no status code)", func() {
			input := newReader("")
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPCache(input, si)
			si.Flush()

			Expect(si.HasError()).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusNotImplemented))
		})

		It("should not write (no header)", func() {
			input := newReader("HTTP 200\n")
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPCache(input, si)
			si.Flush()

			Expect(si.HasError()).To(BeTrue())
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Describe("ServeHTTPGetStatusCode", func() {
		It("should parse 200", func() {
			r := newBufioReader("HTTP 200\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(si.GetStatusCode()).To(Equal(200))
		})

		It("should parse 301", func() {
			r := newBufioReader("HTTP 301\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(si.GetStatusCode()).To(Equal(301))
		})

		It("should parse 404", func() {
			r := newBufioReader("HTTP 404\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(si.GetStatusCode()).To(Equal(404))
		})

		It("should parse 503", func() {
			r := newBufioReader("HTTP 503\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			Expect(si.HasError()).To(BeFalse())
			Expect(si.GetStatusCode()).To(Equal(503))
		})

		It("should handle empty input", func() {
			r := newBufioReader("")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorReadLine)))
		})

		It("should handle broken input", func() {
			r := newBufioReader("Oops\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorParseLine)))
		})

		It("should handle broken status code", func() {
			r := newBufioReader("HTTP 4294967296\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPGetStatusCode(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorParseInt)))
		})
	})

	Describe("ServeHTTPAddHeaders", func() {
		It("should add Content-Type header", func() {
			contentType := "application/octet-stream"
			r := newBufioReader("Content-Type: " + contentType + "\n\n")
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPAddHeaders(r, si)
			si.Flush()

			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
		})

		It("should pick up Content-Length header", func() {
			r := newBufioReader("Content-Length: 1\n\n")
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPAddHeaders(r, si)
			si.Flush()

			Expect(w.Header().Get("Content-Length")).To(Equal("1"))
		})

		It("should pick up our expires header", func() {
			expires := time.Now().Add(time.Minute)
			r := newBufioReader(fmt.Sprintf("%s: %d\n\n", cacher.HTTPHeaderExpires, expires.UnixNano()))
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPAddHeaders(r, si)

			siExpires := si.GetExpires()
			Expect(siExpires).ToNot(BeNil())
			Expect(siExpires.UnixNano()).To(Equal(expires.UnixNano()))
		})

		It("should not add internal headers", func() {
			r := newBufioReader(fmt.Sprintf("%s-One: 1\nTwo: 2\n%s-Three: 3\n\n",
				cacher.HTTPHeaderPrefix, cacher.HTTPHeaderPrefix))
			w := httptest.NewRecorder()
			si := internal.NewServeInfo(w)
			ServeHTTPAddHeaders(r, si)
			si.Flush()

			buffer := &bytes.Buffer{}
			w.Header().Write(buffer)
			Expect(buffer.String()).To(Equal("Two: 2\r\n"))
		})

		It("should handle empty input", func() {
			r := newBufioReader("")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPAddHeaders(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorReadLine)))
		})

		It("should handle broken input", func() {
			r := newBufioReader("Oops\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPAddHeaders(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorParseLine)))
		})

		It("should handle broken content length", func() {
			r := newBufioReader("Content-Length: 9223372036854775808\n")
			si := internal.NewServeInfo(httptest.NewRecorder())
			ServeHTTPAddHeaders(r, si)

			errorType, err := si.GetError()
			Expect(err).To(HaveOccurred())
			Expect(errorType).To(Equal(int(internal.ErrorParseInt)))
		})
	})
})
