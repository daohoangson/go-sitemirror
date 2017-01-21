package crawler

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"

	"github.com/daohoangson/go-sitemirror/cacher"
	cssScanner "github.com/gorilla/css/scanner"
	"golang.org/x/net/html"
	htmlAtom "golang.org/x/net/html/atom"
)

var (
	cssURIRegexp = regexp.MustCompile(`^(url\(['"]?)([^'"]+)(['"]?\))$`)
)

const (
	htmlAttrAction        = "action"
	htmlAttrHref          = "href"
	htmlAttrRel           = "rel"
	htmlAttrRelStylesheet = "stylesheet"
	htmlAttrSrc           = "src"
)

const (
	httpHeaderContentType = "Content-Type"
	httpHeaderLocation    = "Location"
)

// Download returns parsed data after downloading the specified url.
func Download(input *Input) *Downloaded {
	result := &Downloaded{
		Input: input,

		BaseURL:         input.URL,
		LinksAssets:     make(map[string]Link),
		LinksDiscovered: make(map[string]Link),
	}

	if input.Client == nil {
		result.Error = errors.New(".Client cannot be nil")
		return result
	}

	if input.URL == nil {
		result.Error = errors.New(".URL cannot be nil")
		return result
	}

	if !input.URL.IsAbs() {
		result.Error = errors.New(".URL must be absolute")
		return result
	}

	if !strings.HasPrefix(input.URL.Scheme, cacher.SchemeDefault) {
		result.Error = errors.New(".URL.Scheme must be http/https")
		return result
	}

	httpClient := *input.Client
	// http://stackoverflow.com/questions/23297520/how-can-i-make-the-go-http-client-not-follow-redirects-automatically
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// do not follow redirects
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest("GET", input.URL.String(), nil)
	if err != nil {
		result.Error = err
		return result
	}

	if input.Header != nil {
		for headerKey, headerValues := range input.Header {
			for _, headerValue := range headerValues {
				req.Header.Add(headerKey, headerValue)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if result.StatusCode >= 200 && result.StatusCode <= 299 {
		result.Error = parseBody(resp, result)
	} else if result.StatusCode >= 300 && result.StatusCode <= 399 {
		result.Error = parseRedirect(resp, result)
	}

	return result
}

func parseBody(resp *http.Response, result *Downloaded) error {
	respHeaderContentType := resp.Header.Get(httpHeaderContentType)
	if len(respHeaderContentType) > 0 {
		result.AddHeader(httpHeaderContentType, respHeaderContentType)

		parts := strings.Split(respHeaderContentType, ";")
		contentType := parts[0]

		switch contentType {
		case "text/css":
			return parseBodyCSS(resp, result)
		case "text/html":
			return parseBodyHTML(resp, result)
		}
	}

	return parseBodyRaw(resp, result)
}

func parseBodyCSS(resp *http.Response, result *Downloaded) error {
	body, _ := ioutil.ReadAll(resp.Body)

	var buffer bytes.Buffer
	defer buffer.Reset()
	result.buffer = &buffer

	err := parseBodyCSSString(string(body), result)

	result.Body = buffer.String()
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
				before, url, after := m[1], m[2], m[3]
				processedURL, err := result.ProcessURL(CSSUri, url)
				if err == nil && processedURL != url {
					result.buffer.WriteString(before)
					result.buffer.WriteString(processedURL)
					result.buffer.WriteString(after)
					continue
				}
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

	result.Body = buffer.String()
	result.buffer = nil

	return nil
}

func parseBodyHTMLToken(tokenizer *html.Tokenizer, result *Downloaded) bool {
	tokenType := tokenizer.Next()
	if tokenType == html.ErrorToken {
		return true
	}

	switch tokenType {
	case html.StartTagToken:
		token := tokenizer.Token()

		switch token.DataAtom {
		case htmlAtom.A:
			if parseBodyHTMLTagA(&token, result) {
				return false
			}
		case htmlAtom.Form:
			if parseBodyHTMLTagForm(&token, result) {
				return false
			}
		case htmlAtom.Img:
			if parseBodyHTMLTagImg(&token, result) {
				return false
			}
		case htmlAtom.Link:
			if parseBodyHTMLTagLink(&token, result) {
				return false
			}
		case htmlAtom.Script:
			if parseBodyHTMLTagScript(tokenizer, &token, result) {
				return false
			}
		case htmlAtom.Style:
			if parseBodyHTMLTagStyle(tokenizer, result) {
				return false
			}
		}

		rewriteTokenAttr(&token, result)
		return false
	case html.SelfClosingTagToken:
		token := tokenizer.Token()

		switch token.DataAtom {
		case htmlAtom.Base:
			if parseBodyHTMLTagBase(&token, result) {
				return false
			}
		case htmlAtom.Img:
			if parseBodyHTMLTagImg(&token, result) {
				return false
			}
		case htmlAtom.Link:
			if parseBodyHTMLTagLink(&token, result) {
				return false
			}
		}

		rewriteTokenAttr(&token, result)
		return false
	}

	result.buffer.Write(tokenizer.Raw())
	return false
}

func parseBodyHTMLTagA(token *html.Token, result *Downloaded) bool {
	for i, attr := range token.Attr {
		if attr.Key == htmlAttrHref {
			processedURL, err := result.ProcessURL(HTMLTagA, attr.Val)
			if err == nil && processedURL != attr.Val {
				token.Attr[i].Val = processedURL
				return rewriteTokenAttr(token, result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagForm(token *html.Token, result *Downloaded) bool {
	for i, attr := range token.Attr {
		if attr.Key == htmlAttrAction {
			processedURL, err := result.ProcessURL(HTMLTagForm, attr.Val)
			if err == nil && processedURL != attr.Val {
				token.Attr[i].Val = processedURL
				return rewriteTokenAttr(token, result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagBase(token *html.Token, result *Downloaded) bool {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrHref {
			if url, err := neturl.Parse(attr.Val); err == nil {
				result.BaseURL = result.BaseURL.ResolveReference(url)
				return true
			}
		}
	}

	return false
}

func parseBodyHTMLTagImg(token *html.Token, result *Downloaded) bool {
	needRewrite := false

	for i, attr := range token.Attr {
		if attr.Key != htmlAttrSrc && !strings.HasPrefix(attr.Key, "data-") {
			// process src attribute and any data-* attribute that contains url
			// some website uses those for lazy loading / high resolution quality / etc.
			continue
		}

		url := attr.Val
		parsedURL, err := neturl.Parse(url)
		if err != nil {
			continue
		}

		if attr.Key != htmlAttrSrc && !parsedURL.IsAbs() {
			// data-* attribute url must be absolute
			continue
		}

		processedURL, err := result.ProcessURL(HTMLTagImg, url)
		if err == nil && processedURL != attr.Val {
			token.Attr[i].Val = processedURL
			needRewrite = true
		}
	}

	if needRewrite {
		return rewriteTokenAttr(token, result)
	}

	return false
}

func parseBodyHTMLTagLink(token *html.Token, result *Downloaded) bool {
	var linkHref string
	var linkHrefAttrIndex int
	var linkRel string

	for i, attr := range token.Attr {
		switch attr.Key {
		case htmlAttrHref:
			linkHref = attr.Val
			linkHrefAttrIndex = i
		case htmlAttrRel:
			linkRel = attr.Val
		}
	}

	if len(linkHref) > 0 {
		switch linkRel {
		case htmlAttrRelStylesheet:
			processedURL, err := result.ProcessURL(HTMLTagLinkStylesheet, linkHref)
			if err == nil && processedURL != linkHref {
				token.Attr[linkHrefAttrIndex].Val = processedURL
				return rewriteTokenAttr(token, result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagScript(tokenizer *html.Tokenizer, token *html.Token, result *Downloaded) bool {
	for i, attr := range token.Attr {
		if attr.Key == htmlAttrSrc {
			processedURL, err := result.ProcessURL(HTMLTagScript, attr.Val)
			if err == nil && processedURL != attr.Val {
				token.Attr[i].Val = processedURL
				return rewriteTokenAttr(token, result)
			}
		}
	}

	// handle inline script
	result.buffer.Write(tokenizer.Raw())
	for {
		tokenType := tokenizer.Next()
		raw := tokenizer.Raw()

		switch tokenType {
		case html.EndTagToken:
			result.buffer.Write(raw)
			return true
		case html.TextToken:
			parseBodyJsString(string(raw), result)
		}
	}
}

func parseBodyHTMLTagStyle(tokenizer *html.Tokenizer, result *Downloaded) bool {
	result.buffer.Write(tokenizer.Raw())

	for {
		tokenType := tokenizer.Next()
		raw := tokenizer.Raw()

		switch tokenType {
		case html.EndTagToken:
			result.buffer.Write(raw)
			return true
		case html.TextToken:
			parseBodyCSSString(string(raw), result)
		}
	}
}

func parseBodyJsString(js string, result *Downloaded) {
	if strings.Index(js, "getElementsByTagName('base')") > -1 {
		// skip inline js that deals with <base />
		return
	}

	result.buffer.WriteString(js)
}

func rewriteTokenAttr(token *html.Token, result *Downloaded) bool {
	result.buffer.WriteString("<")
	result.buffer.WriteString(token.Data)

	for _, attr := range token.Attr {
		result.buffer.WriteString(" ")
		result.buffer.WriteString(attr.Key)
		result.buffer.WriteString("=\"")

		if attr.Key == "style" {
			parseBodyCSSString(attr.Val, result)
		} else {
			result.buffer.WriteString(html.EscapeString(attr.Val))
		}

		result.buffer.WriteString("\"")
	}

	if token.Type == html.SelfClosingTagToken {
		result.buffer.WriteString(" /")
	}

	result.buffer.WriteString(">")

	return true
}

func parseBodyRaw(resp *http.Response, result *Downloaded) error {
	body, err := ioutil.ReadAll(resp.Body)
	result.Body = string(body)
	return err
}

func parseRedirect(resp *http.Response, result *Downloaded) error {
	location := resp.Header.Get(httpHeaderLocation)
	processedURL, err := result.ProcessURL(HTTP3xxLocation, location)
	if err == nil {
		result.AddHeader(httpHeaderLocation, processedURL)
	}

	return err
}
