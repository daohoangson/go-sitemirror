package cacher

import (
	"crypto/md5"
	"fmt"
	neturl "net/url"
	"os"
	"path"
	"regexp"
	"sort"
)

const (
	// MaxPathNameLength some file system limits the maximum length of a file name
	// therefore each path part should not be too long to avoid os level error
	MaxPathNameLength = 32
	// ShortHashLength length of the short hash
	ShortHashLength = 6
)

var (
	regExpSafePathName = regexp.MustCompile(`[^a-zA-Z0-9\.\-\_\=]`)
)

// MakeDir creates directory tree for the specified path
func MakeDir(cachePath string) error {
	return os.MkdirAll(path.Dir(cachePath), os.ModePerm)
}

// CreateFile returns a file handle for writing with the specified path.
// Existing file will be truncated.
func CreateFile(cachePath string) (*os.File, error) {
	err := MakeDir(cachePath)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(cachePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
}

// OpenFile return a file handle for reading/writing with the specified path.
// File will be created if not already exists.
func OpenFile(cachePath string) (*os.File, error) {
	err := MakeDir(cachePath)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
}

// GenerateHTTPCachePath returns http cache path for the specified url
func GenerateHTTPCachePath(rootPath string, url *neturl.URL) string {
	var (
		urlScheme string
		urlHost   string
		urlPath   string
		queryPath string
	)

	if url != nil {
		urlScheme = url.Scheme
		urlHost = url.Host
		urlPath = url.Path

		urlQuery := url.Query()
		queryPath = BuildQueryPath(&urlQuery)
	}
	dir, file := path.Split(urlPath)

	var fileSafe string
	if len(file) > 0 {
		fileSafe = GetSafePathName(file)
	}

	dirFileAndQuery := path.Join(dir, fileSafe, queryPath)
	if len(dirFileAndQuery) > 0 && dirFileAndQuery != "/" {
		dirFileAndQuery = fmt.Sprintf("%s-%s", dirFileAndQuery, GetShortHash(urlPath))
	} else {
		dirFileAndQuery = GetShortHash(urlPath)
	}

	path := path.Join(
		rootPath,
		GetSafePathName(urlScheme),
		GetSafePathName(urlHost),
		dirFileAndQuery,
	)

	return path
}

// BuildQueryPath returns path elements from the specified query
func BuildQueryPath(query *neturl.Values) string {
	queryKeys := getQuerySortedKeys(query)
	queryPath := ""
	for _, queryKey := range queryKeys {
		queryValues := (*query)[queryKey]
		sort.Strings(queryValues)
		for _, queryValue := range queryValues {
			queryElement := queryKey
			if len(queryValue) > 0 {
				queryElement = fmt.Sprintf("%s=%s", queryKey, queryValue)
			}

			queryPath = path.Join(queryPath, GetSafePathName(queryElement))
		}
	}

	return queryPath
}

// GetSafePathName returns safe version of element name to be used in paths
func GetSafePathName(name string) string {
	name = regExpSafePathName.ReplaceAllString(name, "")

	if len(name) == 0 {
		name = GetShortHash(name)
	} else if len(name) > MaxPathNameLength {
		h := GetShortHash(name)
		name = fmt.Sprintf("%s_%s", name[:MaxPathNameLength-len(h)-1], h)
	}

	return name
}

// GetShortHash returns a short hash of the specified string
func GetShortHash(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum[:ShortHashLength/2])
}

func getQuerySortedKeys(query *neturl.Values) []string {
	keys := make([]string, len(*query))

	i := 0
	for key := range *query {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	return keys
}
