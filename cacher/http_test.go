package cacher_test

import (
	"bytes"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"

	. "github.com/daohoangson/go-sitemirror/cacher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Http", func() {
	Describe("WriteHTTP", func() {
		var buffer bytes.Buffer
		var input2xx *Input

		BeforeEach(func() {
			buffer.Reset()

			url, _ := neturl.Parse("http://domain.com/http/input/2xx")
			input2xx = &Input{StatusCode: 200, URL: url}
		})

		Describe("StatusCode", func() {
			It("should write status code 100", func() {
				status := 100
				input := Input{StatusCode: status}
				WriteHTTP(&buffer, &input)

				written := buffer.String()
				Expect(written).To(HavePrefix(fmt.Sprintf("HTTP %d\n", status)))
				Expect(written).To(HaveSuffix("\n\n"))
			})

			It("should write status code 200", func() {
				status := 200
				input := Input{StatusCode: status}
				WriteHTTP(&buffer, &input)

				written := buffer.String()
				Expect(written).To(HavePrefix(fmt.Sprintf("HTTP %d\n", status)))
			})

			It("should write status code 301", func() {
				status := 301
				input := Input{StatusCode: status}
				WriteHTTP(&buffer, &input)

				written := buffer.String()
				Expect(written).To(HavePrefix(fmt.Sprintf("HTTP %d\n", status)))
			})
		})

		It("should write url header", func() {
			input := input2xx
			url, _ := neturl.Parse("http://domain.com/http/url")
			input.URL = url
			WriteHTTP(&buffer, input)

			written := buffer.String()
			writtenMirroredUrl := getHeaderValue(written, HTTPHeaderURL)
			Expect(writtenMirroredUrl).To(Equal(url.String()))
		})

		It("should write Last-Modified header", func() {
			input := input2xx
			WriteHTTP(&buffer, input)

			written := buffer.String()
			writtenLastModified := getHeaderValue(written, "Last-Modified")
			Expect(len(writtenLastModified)).ToNot(Equal(0))

			_, err := time.Parse(http.TimeFormat, writtenLastModified)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("TTL", func() {
			It("should write our Expires header", func() {
				input := input2xx
				input.TTL = time.Minute
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenExpires := getHeaderValue(written, HTTPHeaderExpires)
				Expect(len(writtenExpires)).ToNot(Equal(0))

				timestamp, err := strconv.ParseUint(writtenExpires, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				Expect(timestamp).To(BeNumerically(">", time.Now().Unix()))
			})

			It("should write cache control headers", func() {
				input := input2xx
				input.TTL = time.Minute
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenCacheControl := getHeaderValue(written, "Cache-Control")
				Expect(writtenCacheControl).To(Equal("public"))

				writtenExpires := getHeaderValue(written, "Expires")
				Expect(len(writtenExpires)).ToNot(Equal(0))

				t, err := time.Parse(http.TimeFormat, writtenExpires)
				Expect(err).ToNot(HaveOccurred())
				Expect(t.UnixNano()).To(BeNumerically(">", time.Now().UnixNano()))
			})

			It("should not write our Expires header", func() {
				input := input2xx
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenExpires := getHeaderValue(written, HTTPHeaderExpires)
				Expect(len(writtenExpires)).To(Equal(0))
			})
		})

		Context("2xx", func() {
			It("should write Content-Type header", func() {
				input := input2xx
				input.ContentType = "text/html"
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentType := getHeaderValue(written, "Content-Type")
				Expect(writtenContentType).To(Equal(input.ContentType))
				Expect(written).To(HaveSuffix("\n\n"))
			})

			It("should write body string", func() {
				input := input2xx
				input.Body = "foo/bar"
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentLength := getHeaderValue(written, "Content-Length")
				Expect(writtenContentLength).To(Equal(fmt.Sprintf("%d", len(input.Body))))

				writtenContent := getContent(written)
				Expect(writtenContent).To(Equal(input.Body))
			})
		})

		Context("3xx", func() {
			It("should write Location header", func() {
				url, _ := neturl.Parse("http://domain.com/http/input/3xx/location")
				targetUrl, _ := neturl.Parse("http://domain.com/target/url")
				input := &Input{StatusCode: 301, URL: url, Redirection: targetUrl}
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenLocation := getHeaderValue(written, "Location")
				Expect(writtenLocation).To(Equal(targetUrl.String()))
				Expect(written).To(HaveSuffix("\n\n"))
			})
		})
	})
})

func getHeaderValue(written string, headerKey string) string {
	lines := strings.Split(written, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			// reached content, return asap
			return ""
		}

		parts := strings.Split(line, ": ")
		if parts[0] == headerKey {
			return parts[1]
		}
	}

	// header not found
	return ""
}

func getContent(written string) string {
	sep := "\n\n"
	index := strings.Index(written, sep)
	if index == -1 {
		// content not found
		return ""
	}

	return written[index+len(sep):]
}
