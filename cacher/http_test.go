package cacher_test

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	neturl "net/url"
	"strconv"
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
			input2xx = &Input{
				StatusCode: 200,
				URL:        url,
				Header:     make(http.Header),
			}
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
			writtenMirroredUrl := getHeaderValue(written, CustomHeaderURL)
			Expect(writtenMirroredUrl).To(Equal(url.String()))
		})

		It("should write Last-Modified header", func() {
			input := input2xx
			WriteHTTP(&buffer, input)

			written := buffer.String()
			writtenLastModified := getHeaderValue(written, HeaderLastModified)
			Expect(len(writtenLastModified)).ToNot(Equal(0))

			_, err := time.Parse(http.TimeFormat, writtenLastModified)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("Caching", func() {
			It("should write our Expires header", func() {
				input := input2xx
				input.TTL = time.Minute
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenExpires := getHeaderValue(written, CustomHeaderExpires)
				Expect(len(writtenExpires)).ToNot(Equal(0))

				timestamp, err := strconv.ParseUint(writtenExpires, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				Expect(timestamp).To(BeNumerically(">", time.Now().UnixNano()))
			})

			It("should write cache control headers", func() {
				input := input2xx
				input.TTL = time.Minute
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenCacheControl := getHeaderValue(written, HeaderCacheControl)
				Expect(writtenCacheControl).To(Equal("public, max-age=60"))

				writtenExpires := getHeaderValue(written, HeaderExpires)
				Expect(len(writtenExpires)).ToNot(Equal(0))

				t, err := time.Parse(http.TimeFormat, writtenExpires)
				Expect(err).ToNot(HaveOccurred())
				Expect(t.UnixNano()).To(BeNumerically(">", time.Now().UnixNano()))
			})

			It("should not write our Expires header", func() {
				input := input2xx
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenExpires := getHeaderValue(written, CustomHeaderExpires)
				Expect(len(writtenExpires)).To(Equal(0))
			})

			Describe("WriteHTTPCachingHeaders", func() {
				Context(HeaderExpires, func() {
					It("should pick up header value", func() {
						expires := time.Now().Add(time.Hour).Format(http.TimeFormat)
						input := input2xx
						input.Header.Add(HeaderExpires, expires)
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, HeaderExpires)
						Expect(string(writtenExpires)).To(Equal(expires))
					})

					It("should not pick up invalid date", func() {
						input := input2xx
						input.Header.Add(HeaderExpires, "Oops")
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, HeaderExpires)
						Expect(string(writtenExpires)).To(Equal(""))
					})

					It("should not pick up date in the past", func() {
						input := input2xx
						input.Header.Add(HeaderExpires, time.Now().Add(-24*time.Hour).Format(http.TimeFormat))
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, HeaderExpires)
						Expect(string(writtenExpires)).To(Equal(""))
					})
				})

				Context(HeaderCacheControl, func() {
					It("should pick up header max-age value", func() {
						maxAge := 3600
						input := input2xx
						input.Header.Add(HeaderCacheControl, fmt.Sprintf("max-age=%d", maxAge))
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, CustomHeaderExpires)
						timestamp, _ := strconv.ParseUint(writtenExpires, 10, 64)
						Expect(timestamp / uint64(time.Second)).
							To(BeNumerically("==", time.Now().Add(time.Duration(maxAge)*time.Second).Unix()))
					})

					It("should pick up header max-age value after public", func() {
						maxAge := 3601
						input := input2xx
						input.Header.Add(HeaderCacheControl, fmt.Sprintf("public, max-age=%d", maxAge))
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, CustomHeaderExpires)
						timestamp, _ := strconv.ParseUint(writtenExpires, 10, 64)
						Expect(timestamp / uint64(time.Second)).
							To(BeNumerically("==", time.Now().Add(time.Duration(maxAge)*time.Second).Unix()))
					})

					It("should not pick up invalid max-age", func() {
						input := input2xx
						input.Header.Add(HeaderCacheControl, "max-age=foo")
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, CustomHeaderExpires)
						Expect(string(writtenExpires)).To(Equal(""))
					})

					It("should not pick up 0 max-age", func() {
						input := input2xx
						input.Header.Add(HeaderCacheControl, "max-age=0")
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, CustomHeaderExpires)
						Expect(string(writtenExpires)).To(Equal(""))
					})

					It("should not pick up negative max-age", func() {
						input := input2xx
						input.Header.Add(HeaderCacheControl, "max-age=-1")
						bw := bufio.NewWriter(&buffer)
						WriteHTTPCachingHeaders(bw, input)
						bw.Flush()

						written := buffer.String()
						writtenExpires := getHeaderValue(written, CustomHeaderExpires)
						Expect(string(writtenExpires)).To(Equal(""))
					})
				})
			})
		})

		Context("2xx", func() {
			It("should write Content-Type header", func() {
				headerKey := HeaderContentType
				headerValue := "plain/text"
				input := input2xx
				input.Header.Add(headerKey, headerValue)
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenContentType := getHeaderValue(written, headerKey)
				Expect(writtenContentType).To(Equal(headerValue))
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
				headerKey := HeaderLocation
				headerValue := "http://domain.com/target/url"
				url, _ := neturl.Parse("http://domain.com/http/input/3xx/location")
				input := &Input{StatusCode: 301, URL: url, Header: make(http.Header)}
				input.Header.Add(headerKey, headerValue)
				WriteHTTP(&buffer, input)

				written := buffer.String()
				writtenLocation := getHeaderValue(written, headerKey)
				Expect(writtenLocation).To(Equal(headerValue))
				Expect(written).To(HaveSuffix("\n\n"))
			})
		})
	})
})
