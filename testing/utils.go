package testing

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/jarcoal/httpmock.v1"
)

// InvalidURL an invalid url as in RFC 6874
// Source: https://golang.org/src/net/url/url_test.go
const InvalidURL = `http://[fe80::1%en0]/`

// TransparentDataURI a valid data uri for the (probaby) smallest transparent gif
// Source: http://stackoverflow.com/questions/6018611/smallest-data-uri-image-possible-for-a-transparent-image
const TransparentDataURI = `data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7`

func NewCssResponder(css string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(200, css)
		resp.Header.Add("Content-Type", "text/css")
		return resp, nil
	}
}

func NewHtmlMarkup(body string) string {
	return fmt.Sprintf("<html><head><title>Title</title>%s</html>", body)
}

func NewHtmlResponder(html string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(200, html)
		resp.Header.Add("Content-Type", "text/html")
		return resp, nil
	}
}

func NewRedirectResponder(status int, location string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(status, "")
		resp.Header.Add("Location", location)
		return resp, nil
	}
}

func NewSlowResponder(duration time.Duration) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		time.Sleep(duration)
		resp := httpmock.NewStringResponse(200, "")
		return resp, nil
	}
}
