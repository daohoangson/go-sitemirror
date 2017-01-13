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

var cssURIRegexp = regexp.MustCompile(`^(url\(['"]?)([^'"]+)(['"]?\))$`)

const htmlAttrHref = "href"
const htmlAttrRel = "rel"
const htmlAttrRelStylesheet = "stylesheet"
const htmlAttrSrc = "src"

// Download returns parsed data after downloading the specified url.
func Download(client *http.Client, url *neturl.URL) *Downloaded {
	result := newDownloaded(url)

	if client == nil {
		result.Error = errors.New("http.Client cannot be nil")
		return result
	}

	if url == nil {
		result.Error = errors.New("url.URL cannot be nil")
		return result
	}

	if !url.IsAbs() {
		result.Error = errors.New("URL must be absolute")
		return result
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		result.Error = errors.New("URL scheme must be http/https")
		return result
	}

	// http://stackoverflow.com/questions/23297520/how-can-i-make-the-go-http-client-not-follow-redirects-automatically
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// do not follow redirects
		return http.ErrUseLastResponse
	}

	resp, err := client.Get(url.String())
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
				before, url, after := m[1], m[2], m[3]
				relative := result.appendURL(CSSUri, url)
				if relative != url {
					result.buffer.WriteString(before)
					result.buffer.WriteString(relative)
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
			if parseBodyHTMLTagA(&token, result) {
				return false
			}
		case htmlAtom.Script:
			if parseBodyHTMLTagScript(&token, result) {
				return false
			}
		case htmlAtom.Style:
			if parseBodyHTMLTagStyle(tokenizer, result) {
				return false
			}
		}
	case html.SelfClosingTagToken:
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
	}

	result.buffer.Write(raw)
	return false
}

func parseBodyHTMLTagA(token *html.Token, result *Downloaded) bool {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrHref {
			relative := result.appendURL(HTMLTagA, attr.Val)
			if relative != attr.Val {
				return rewriteTokenAttr(token, attr.Key, relative, result)
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
				return rewriteTokenAttr(token, attr.Key, ".", result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagImg(token *html.Token, result *Downloaded) bool {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrSrc {
			relative := result.appendURL(HTMLTagImg, attr.Val)
			if relative != attr.Val {
				return rewriteTokenAttr(token, attr.Key, relative, result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagLink(token *html.Token, result *Downloaded) bool {
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
		switch linkRel {
		case htmlAttrRelStylesheet:
			relative := result.appendURL(HTMLTagLinkStylesheet, linkHref)
			if relative != linkHref {
				return rewriteTokenAttr(token, htmlAttrHref, relative, result)
			}
		}
	}

	return false
}

func parseBodyHTMLTagScript(token *html.Token, result *Downloaded) bool {
	for _, attr := range token.Attr {
		if attr.Key == htmlAttrSrc {
			relative := result.appendURL(HTMLTagScript, attr.Val)
			if relative != attr.Val {
				return rewriteTokenAttr(token, attr.Key, relative, result)
			}
		}
	}

	return false
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

func rewriteTokenAttr(token *html.Token, attrKey string, attrVal string, result *Downloaded) bool {
	result.buffer.WriteString("<")
	result.buffer.WriteString(token.Data)

	for _, attr := range token.Attr {
		val := attr.Val
		if attr.Key == attrKey {
			val = attrVal
		}

		result.buffer.WriteString(" ")
		result.buffer.WriteString(attr.Key)
		result.buffer.WriteString("=\"")
		result.buffer.WriteString(html.EscapeString(val))
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
	result.BodyBytes = body
	return err
}

func parseRedirect(resp *http.Response, result *Downloaded) error {
	location := resp.Header.Get("Location")
	url, err := neturl.Parse(location)
	if err != nil {
		return err
	}

	result.HeaderLocation = url
	result.appendURL(HTTP3xxLocation, location)

	return nil
}