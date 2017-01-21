package testing

import (
	"fmt"
	"net/http"
	"time"

	"github.com/daohoangson/go-sitemirror/cacher"

	"gopkg.in/jarcoal/httpmock.v1"
)

// InvalidURL an invalid url as in RFC 6874
// Source: https://golang.org/src/net/url/url_test.go
const InvalidURL = `http://[fe80::1%en0]/`

// TransparentDataURI a valid data uri for the (probaby) smallest transparent gif
// Source: http://stackoverflow.com/questions/6018611/smallest-data-uri-image-possible-for-a-transparent-image
const TransparentDataURI = `data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7`

// NewCSSResponder returns a new responder with css content type
func NewCSSResponder(css string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(http.StatusOK, css)
		resp.Header.Add(cacher.HeaderContentType, "text/css")
		return resp, nil
	}
}

// NewHTMLMarkup returns html wraps around given body mark up
func NewHTMLMarkup(body string) string {
	return fmt.Sprintf("<html><head><title>Title</title>%s</html>", body)
}

// NewHTMLResponder returns a new responder with html content type
func NewHTMLResponder(html string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(http.StatusOK, html)
		resp.Header.Add(cacher.HeaderContentType, "text/html")
		return resp, nil
	}
}

// NewRedirectResponder returns a new responder with Location header
func NewRedirectResponder(status int, location string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(status, "")
		resp.Header.Add(cacher.HeaderLocation, location)
		return resp, nil
	}
}

// NewSlowResponder returns a new responder that takes its time to respond
func NewSlowResponder(duration time.Duration) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		time.Sleep(duration)
		resp := httpmock.NewStringResponse(http.StatusOK, "")
		return resp, nil
	}
}
