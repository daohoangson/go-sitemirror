package crawler

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
)

var (
	errorEmptyURL      = errors.New("Empty url")
	errorEmptyInput    = errors.New("Empty .Input")
	errorEmptyInputURL = errors.New("Empty .Input.URL")
)

// AddHeader adds a new header
func (d *Downloaded) AddHeader(key string, value string) {
	if d.header == nil {
		d.header = make(http.Header)
	}

	d.header.Add(key, value)
}

// GetHeaderKeys returns all header keys
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

// GetHeaderValues returns values of the specified header key
func (d *Downloaded) GetHeaderValues(key string) []string {
	if d.header == nil {
		return nil
	}

	if values, ok := d.header[http.CanonicalHeaderKey(key)]; ok {
		return values
	}

	return nil
}

// ProcessURL validates url and returns rewritten string representation
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
	if filteredURL.String() != d.Input.URL.String() {
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
	}

	reduced := d.Reduce(fullURL)
	return reduced, nil
}

// Reduce returns relative version of url from .Input.URL
func (d *Downloaded) Reduce(url *neturl.URL) string {
	var (
		crossHostOf = func(target *neturl.URL) *neturl.URL {
			scheme := target.Scheme
			if len(scheme) == 0 {
				scheme = "http"
			}

			crossHost := *target
			crossHost.Scheme = "http"
			crossHost.Path = fmt.Sprintf("/%s/%s%s", scheme, target.Host, target.Path)
			crossHost.Host = "cross.localhost"

			return &crossHost
		}
	)

	if !d.Input.NoCrossHost &&
		(d.Input.URL.Scheme != url.Scheme ||
			d.Input.URL.Host != url.Host) {
		// different host, use cross-host relative path
		return ReduceURL(crossHostOf(d.Input.URL), crossHostOf(url))
	}

	return ReduceURL(d.Input.URL, url)
}

// GetAssetURLs returns resolved asset urls
func (d *Downloaded) GetAssetURLs() []*neturl.URL {
	urls := make([]*neturl.URL, len(d.LinksAssets))

	i := 0
	for _, link := range d.LinksAssets {
		urls[i] = d.BaseURL.ResolveReference(link.URL)
		i++
	}

	return urls
}

// GetDiscoveredURLs returns resolved discovered link urls
func (d *Downloaded) GetDiscoveredURLs() []*neturl.URL {
	urls := make([]*neturl.URL, len(d.LinksDiscovered))

	i := 0
	for _, link := range d.LinksDiscovered {
		urls[i] = d.BaseURL.ResolveReference(link.URL)
		i++
	}

	return urls
}
