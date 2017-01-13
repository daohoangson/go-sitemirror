package crawler

import (
	neturl "net/url"
)

func (d *Downloaded) GetResolvedURLs() []*neturl.URL {
	resolvedURLs := make([]*neturl.URL, len(d.Links))

	i := 0
	for _, link := range d.Links {
		resolvedURLs[i] = d.BaseURL.ResolveReference(link.URL)
		i++
	}

	return resolvedURLs
}

func newDownloaded(url *neturl.URL) *Downloaded {
	d := Downloaded{
		URL:     url,
		BaseURL: url,
		Links:   make(map[string]Link),
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
	d.Links[fullURL.String()] = link

	return ReduceURL(d.BaseURL, fullURL)
}
