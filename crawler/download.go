package crawler

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"

	cssScanner "github.com/gorilla/css/scanner"
	"golang.org/x/net/html"
	htmlAtom "golang.org/x/net/html/atom"
)

var cssURIRegexp = regexp.MustCompile(`^url\((.+)\)$`)

const htmlAttrHref = "href"
const htmlAttrRel = "rel"
const htmlAttrRelStylesheet = "stylesheet"
const htmlAttrSrc = "src"

// Download returns parsed data after downloading the specified url.
func Download(client *http.Client, url *neturl.URL) *Downloaded {
	result := Downloaded{BaseURL: url, Links: make([]Link, 0)}

	if client == nil {
		result.Error = errors.New("http.Client cannot be nil")
		return &result
	}

	if url == nil {
		result.Error = errors.New("url.URL cannot be nil")
		return &result
	}

	if !url.IsAbs() {
		result.Error = errors.New("URL must be absolute")
		return &result
	}

	// http://stackoverflow.com/questions/23297520/how-can-i-make-the-go-http-client-not-follow-redirects-automatically
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// do not follow redirects
		return http.ErrUseLastResponse
	}

	resp, err := client.Get(url.String())
	if err != nil {
		result.Error = err
		return &result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if result.StatusCode >= 200 && result.StatusCode <= 299 {
		result.Error = parseBody(resp, &result)
	} else if result.StatusCode >= 300 && result.StatusCode <= 399 {
		result.Error = parseRedirect(resp, &result)
	}

	return &result
}

func parseBody(resp *http.Response, result *Downloaded) error {
	contentType := resp.Header.Get("Content-Type")
	parts := strings.Split(contentType, ";")
	result.ContentType = parts[0]

	switch result.ContentType {
	case "text/css":
		return parseBodyCSS(resp, result)
	case "text/html":
		return parseBodyHTML(resp, result)
	}

	return parseBodyRaw(resp, result)
}

func parseBodyCSS(resp *http.Response, result *Downloaded) error {
	body, _ := ioutil.ReadAll(resp.Body)

	var buffer bytes.Buffer
	defer buffer.Reset()
	result.buffer = &buffer

	err := parseBodyCSSString(string(body), result)

	result.BodyString = buffer.String()
	result.buffer = nil

	return err
}

func parseBodyCSSString(css string, result *Downloaded) error {
	scanner := cssScanner.New(css)
	for {
		token := scanner.Next()
		if token.Type == cssScanner.TokenEOF || token.Type == cssScanner.TokenError {
			break
		}

		if token.Type == cssScanner.TokenURI {
			if m := cssURIRegexp.FindStringSubmatch(token.Value); m != nil {
				url := strings.Trim(m[1], `'"`)
				result.appendURL(CSSUri, result.buffer.Len(), url)
			}
		}

		result.buffer.WriteString(token.Value)
	}

	return nil
}

func parseBodyHTML(resp *http.Response, result *Downloaded) error {
	var buffer bytes.Buffer
	defer buffer.Reset()
	result.buffer = &buffer

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		if parseBodyHTMLToken(tokenizer, result) {
			break
		}
	}

	result.BodyString = buffer.String()
	result.buffer = nil

	return nil
}

func parseBodyHTMLToken(tokenizer *html.Tokenizer, result *Downloaded) bool {
	tokenType := tokenizer.Next()
	if tokenType == html.ErrorToken {
		return true
	}

	token := tokenizer.Token()
	raw := tokenizer.Raw()

	switch tokenType {
	case html.StartTagToken:
		switch token.DataAtom {
		case htmlAtom.A:
			parseBodyHTMLTagA(&token, raw, result)
		case htmlAtom.Script:
			parseBodyHTMLTagScript(&token, raw, result)
		case htmlAtom.Style:
			parseBodyHTMLTagStyleAndWrite(tokenizer, result)
			return false
		}
	case html.SelfClosingTagToken:
		switch token.DataAtom {
		case htmlAtom.Base:
			parseBodyHTMLTagBase(&token, result)
		case htmlAtom.Img:
			parseBodyHTMLTagImg(&token, raw, result)
		case htmlAtom.Link:
			parseBodyHTMLTagLink(&token, raw, result)
		}
	}

	result.buffer.Write(raw)
	return false
}

func parseBodyHTMLTagA(token *html.Token, raw []byte, result *Downloaded) {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrHref {
			offset := result.buffer.Len() + strings.Index(string(raw), attr.Val)
			result.appendURL(HTMLTagA, offset, attr.Val)
		}
	}
}

func parseBodyHTMLTagBase(token *html.Token, result *Downloaded) {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrHref {
			if url, err := neturl.Parse(attr.Val); err == nil {
				result.BaseURL = result.BaseURL.ResolveReference(url)
			}
		}
	}
}

func parseBodyHTMLTagImg(token *html.Token, raw []byte, result *Downloaded) {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrSrc {
			offset := result.buffer.Len() + strings.Index(string(raw), attr.Val)
			result.appendURL(HTMLTagImg, offset, attr.Val)
		}
	}
}

func parseBodyHTMLTagLink(token *html.Token, raw []byte, result *Downloaded) {
	var linkHref string
	var linkRel string

	for _, attr := range token.Attr {
		switch attr.Key {
		case htmlAttrHref:
			linkHref = attr.Val
		case htmlAttrRel:
			linkRel = attr.Val
		}
	}

	if len(linkHref) > 0 {
		offset := result.buffer.Len() + strings.Index(string(raw), linkHref)
		switch linkRel {
		case htmlAttrRelStylesheet:
			result.appendURL(HTMLTagLinkStylesheet, offset, linkHref)
		}
	}
}

func parseBodyHTMLTagScript(token *html.Token, raw []byte, result *Downloaded) {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrSrc {
			offset := result.buffer.Len() + strings.Index(string(raw), attr.Val)
			result.appendURL(HTMLTagScript, offset, attr.Val)
		}
	}
}

func parseBodyHTMLTagStyleAndWrite(tokenizer *html.Tokenizer, result *Downloaded) {
	result.buffer.Write(tokenizer.Raw())

	for {
		tokenType := tokenizer.Next()
		raw := tokenizer.Raw()

		switch tokenType {
		case html.EndTagToken:
			result.buffer.Write(raw)
			return
		case html.TextToken:
			parseBodyCSSString(string(raw), result)
		}
	}
}

func parseBodyRaw(resp *http.Response, result *Downloaded) error {
	body, err := ioutil.ReadAll(resp.Body)
	result.BodyBytes = body
	return err
}

func parseRedirect(resp *http.Response, result *Downloaded) error {
	location := resp.Header.Get("Location")
	url, err := neturl.Parse(location)
	result.HeaderLocation = url

	return err
}

// GetResolvedURL returns resolved url for the specified link
func (result *Downloaded) GetResolvedURL(i int) *neturl.URL {
	if i < 0 || i >= len(result.Links) {
		return nil
	}

	return result.BaseURL.ResolveReference(result.Links[i].URL)
}

func (result *Downloaded) appendURL(context urlContext, offset int, input string) {
	url, err := neturl.Parse(input)
	if err != nil {
		return
	}

	inputLen := len(input)

	fragmentLen := len(url.Fragment)
	if fragmentLen > 0 {
		// discard fragment
		inputLen -= fragmentLen + 1
		url.Fragment = ""
	}

	if len(url.String()) == 0 {
		// empty url, this may happen after fragment removal
		return
	}

	link := Link{
		Context: context,
		Offset:  offset,
		Length:  inputLen,
		URL:     url,
	}
	result.Links = append(result.Links, link)
}
