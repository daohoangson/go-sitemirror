package cacher

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
)

const (
	// MaxPathNameLength some file system limits the maximum length of a file name
	// therefore each path part should not be too long to avoid os level error
	MaxPathNameLength = 32
	ShortHashLength   = 6
)

var (
	regExpSafePathName = regexp.MustCompile(`[^a-zA-Z0-9\.\-\_\=]`)
)

func CreateFile(cachePath string) (*os.File, error) {
	dir, _ := path.Split(cachePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(cachePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
}

func GenerateCachePath(rootPath string, url *url.URL) string {
	dir, file := path.Split(url.Path)

	query := url.Query()
	queryPath := BuildQueryPath(&query)

	fileSafe := file
	if len(file) > 0 {
		fileSafe = GetSafePathName(file)
	}

	path := path.Join(
		rootPath,
		GetSafePathName(url.Host),
		dir,
		queryPath,
		GetShortHash(url.Path),
		fileSafe,
	)

	return path
}

func BuildQueryPath(query *url.Values) string {
	queryKeys := getQuerySortedKeys(query)
	queryPath := ""
	for _, queryKey := range queryKeys {
		queryValues := (*query)[queryKey]
		sort.Strings(queryValues)
		for _, queryValue := range queryValues {
			queryPath = path.Join(queryPath, GetSafePathName(fmt.Sprintf("%s=%s", queryKey, queryValue)))
		}
	}

	return queryPath
}

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

func GetShortHash(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum[:ShortHashLength/2])
}

func getQuerySortedKeys(query *url.Values) []string {
	keys := make([]string, len(*query))

	i := 0
	for key := range *query {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	return keys
}
