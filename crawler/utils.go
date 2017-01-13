package crawler

import (
	neturl "net/url"
	"path"
	"strings"
)

func ReduceURL(base *neturl.URL, url *neturl.URL) string {
	if !base.IsAbs() ||
		!url.IsAbs() ||
		base.Host != url.Host {
		// no hope to reduce the url
		return url.String()
	}

	if base.Scheme != url.Scheme {
		if strings.Index(base.Scheme, "http") == 0 &&
			strings.Index(url.Scheme, "http") == 0 {
			// consider http/https to be the same
		} else {
			return url.String()
		}
	}

	reduced, _ := neturl.Parse(url.String())
	reduced.Scheme = ""
	reduced.Host = ""

	lcp := LongestCommonPrefix(base.Path, reduced.Path)

	basePathLeft := base.Path[len(lcp):]
	basePathParts := strings.Split(basePathLeft, "/")

	urlPath := reduced.Path[len(lcp):]
	for i := 1; i < len(basePathParts); i++ {
		urlPath = path.Join("..", urlPath)
	}
	reduced.Path = urlPath

	return reduced.String()
}

func LongestCommonPrefix(path1 string, path2 string) string {
	x, y := path1, path2
	if path1 > path2 {
		x, y = path2, path1
	}

	lenSorter := len(x)
	if len(y) < lenSorter {
		lenSorter = len(y)
	}

	i := 0
	for {
		if i >= lenSorter {
			break
		}

		tmp := x[i:]
		index := strings.Index(tmp, "/")
		if index < 0 {
			return x[:i]
		}
		partLen := index + 1

		if i+partLen >= lenSorter || x[i:i+partLen] != y[i:i+partLen] {
			return x[:i]
		}

		i += partLen
	}

	return x
}
