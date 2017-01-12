package cacher_test

import (
	"bytes"
	"fmt"
	neturl "net/url"
	"strings"

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
				Expect(written).To(Equal(fmt.Sprintf("HTTP %d\n", status)))
			})

			It("should write status code 200", func() {
				status := 200
				input := Input{StatusCode: status}
				WriteHTTP(&buffer, &input)

				written := buffer.String()
				Expect(written).To(Equal(fmt.Sprintf("HTTP %d\n", status)))
			})

			It("should write status code 301", func() {
				status := 301
				input := Input{StatusCode: status}
				WriteHTTP(&buffer, &input)

				written := buffer.String()
				Expect(written).To(Equal(fmt.Sprintf("HTTP %d\n", status)))
			})
		})

		It("should write url", func() {
			status := 200
			url, _ := neturl.Parse("http://domain.com/http/url")
			input := Input{StatusCode: status, URL: url}
			WriteHTTP(&buffer, &input)

			written := buffer.String()
			Expect(written).To(Equal(fmt.Sprintf("HTTP %d\nX-Mirrored-Url: %s\n", status, url.String())))
		})

		Context("2xx", func() {
			It("should write Content-Type header", func() {
				input := input2xx
				input.ContentType = "text/html"
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentType := getHeaderValue(written, "Content-Type")
				Expect(writtenContentType).To(Equal(input.ContentType))
			})

			It("should write body string", func() {
				input := input2xx
				input.BodyString = "foo/bar"
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentLength := getHeaderValue(written, "Content-Length")
				Expect(writtenContentLength).To(Equal(fmt.Sprintf("%d", len(input.BodyString))))

				writtenContent := getContent(written)
				Expect(writtenContent).To(Equal(input.BodyString))
			})

			It("should write body bytes", func() {
				input := input2xx
				input.BodyBytes = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentLength := getHeaderValue(written, "Content-Length")
				Expect(writtenContentLength).To(Equal(fmt.Sprintf("%d", len(input.BodyBytes))))

				writtenContent := getContent(written)
				Expect(writtenContent).To(Equal(string(input.BodyBytes)))
			})
		})

		Context("3xx", func() {
			It("should write Location header", func() {
				url, _ := neturl.Parse("http://domain.com/http/input/3xx/location")
				targetUrl, _ := neturl.Parse("http://domain.com/target/url")
				input := &Input{StatusCode: 301, URL: url, HeaderLocation: targetUrl}
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenLocation := getHeaderValue(written, "Location")
				Expect(writtenLocation).To(Equal(targetUrl.String()))
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
