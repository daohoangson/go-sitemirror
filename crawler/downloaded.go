package crawler

import (
	"errors"
	"net/http"
	neturl "net/url"
)

var (
	errorEmptyURL      = errors.New("Empty url")
	errorEmptyInput    = errors.New("Empty .Input")
	errorEmptyInputURL = errors.New("Empty .Input.URL")
)

func (d *Downloaded) AddHeader(key string, value string) {
	if d.header == nil {
		d.header = make(http.Header)
	}

	d.header.Add(key, value)
}

func (d *Downloaded) GetHeaderKeys() []string {
	if d.header == nil {
		return nil
	}

	keys := make([]string, len(d.header))
	i := 0
	for key := range d.header {
		keys[i] = key
	}

	return keys
}

func (d *Downloaded) GetHeaderValues(key string) []string {
	if d.header == nil {
		return nil
	}

	if values, ok := d.header[http.CanonicalHeaderKey(key)]; ok {
		return values
	}

	return nil
}

func (d *Downloaded) ProcessURL(context urlContext, url string) (string, error) {
	if len(url) == 0 {
		return url, errorEmptyURL
	}

	if d.Input == nil {
		return url, errorEmptyInput
	}

	if d.Input.URL == nil {
		return url, errorEmptyInputURL
	}

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return url, err
	}

	if d.Input.Rewriter != nil {
		(*d.Input.Rewriter)(parsedURL)
	}

	fullURL := d.BaseURL.ResolveReference(parsedURL)
	if fullURL.Scheme != "http" && fullURL.Scheme != "https" {
		return url, nil
	}

	filteredURL, _ := neturl.Parse(fullURL.String())
	filteredURL.Fragment = ""
	if filteredURL.String() == d.BaseURL.String() {
		return url, nil
	}

	link := Link{
		Context: context,
		URL:     filteredURL,
	}

	mapKey := filteredURL.String()

	switch context {
	case HTMLTagA:
		d.LinksDiscovered[mapKey] = link
	case HTMLTagForm:
		d.LinksDiscovered[mapKey] = link
	case HTTP3xxLocation:
		d.LinksDiscovered[mapKey] = link
	default:
		d.LinksAssets[mapKey] = link
	}

	reduced := ReduceURL(d.Input.URL, fullURL)
	return reduced, nil
}

func (d *Downloaded) GetAssetURLs() []*neturl.URL {
	urls := make([]*neturl.URL, len(d.LinksAssets))

	i := 0
	for _, link := range d.LinksAssets {
		urls[i] = d.BaseURL.ResolveReference(link.URL)
		i++
	}

	return urls
}

func (d *Downloaded) GetDiscoveredURLs() []*neturl.URL {
	urls := make([]*neturl.URL, len(d.LinksDiscovered))

	i := 0
	for _, link := range d.LinksDiscovered {
		urls[i] = d.BaseURL.ResolveReference(link.URL)
		i++
	}

	return urls
}
