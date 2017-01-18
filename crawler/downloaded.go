package crawler

import (
	neturl "net/url"
)

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

func newDownloaded(url *neturl.URL) *Downloaded {
	d := Downloaded{
		URL:             url,
		BaseURL:         url,
		LinksAssets:     make(map[string]Link),
		LinksDiscovered: make(map[string]Link),
	}

	return &d
}

func (d *Downloaded) appendURL(context urlContext, input string) string {
	if len(input) == 0 {
		return input
	}

	url, err := neturl.Parse(input)
	if err != nil {
		return input
	}

	return d.appendParsedURL(context, input, url)
}

func (d *Downloaded) appendParsedURL(context urlContext, input string, url *neturl.URL) string {
	fullURL := d.BaseURL.ResolveReference(url)
	if fullURL.Scheme != "http" && fullURL.Scheme != "https" {
		return input
	}

	filteredURL, _ := neturl.Parse(fullURL.String())
	filteredURL.Fragment = ""
	if filteredURL.String() == d.BaseURL.String() {
		return input
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

	return ReduceURL(d.URL, fullURL)
}
