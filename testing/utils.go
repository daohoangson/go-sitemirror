package testing

import (
	"fmt"
	"net/http"

	"gopkg.in/jarcoal/httpmock.v1"
)

const InvalidUrl = `http://[fe80::1%en0]/`

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
