package crawler_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/jarcoal/httpmock.v1"

	"testing"
)

func TestCrawler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crawler Suite")
}

const invalidUrl = `http://[fe80::1%en0]/`

func newCssResponder(css string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(200, css)
		resp.Header.Add("Content-Type", "text/css")
		return resp, nil
	}
}

func newHtmlMarkup(body string) string {
	return fmt.Sprintf("<html><head><title>Title</title>%s</html>", body)
}

func newHtmlResponder(html string) httpmock.Responder {
	return func(req *http.Request) (*http.Response, error) {
		resp := httpmock.NewStringResponse(200, html)
		resp.Header.Add("Content-Type", "text/html")
		return resp, nil
	}
}
