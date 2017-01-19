package crawler

import (
	"fmt"
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
	lcpDir, _ := path.Split(lcp)

	basePathRemaining := base.Path[len(lcpDir):]
	basePathParts := strings.Split(basePathRemaining, "/")

	urlPath := reduced.Path[len(lcpDir):]
	urlDir, urlFile := path.Split(urlPath)

	if len(basePathParts) > 1 {
		for i := 1; i < len(basePathParts); i++ {
			urlDir = path.Join("..", urlDir)
		}
	} else {
		urlDir = fmt.Sprintf("./%s", strings.TrimLeft(urlDir, "/"))
	}

	reduced.Path = fmt.Sprintf("%s/%s", strings.TrimRight(urlDir, "/"), urlFile)

	return reduced.String()
}

func LongestCommonPrefix(path1 string, path2 string) string {
	const sep = "/"
	x, y := strings.Split(path1, sep), strings.Split(path2, sep)
	if path1 > path2 {
		x, y = strings.Split(path2, sep), strings.Split(path1, sep)
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

		if i >= lenSorter || x[i] != y[i] {
			return strings.Join(append(x[:i], ""), sep)
		}

		i++
	}

	return strings.Join(x, sep)
}
