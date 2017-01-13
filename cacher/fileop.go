package cacher

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
)

func CreateFile(cachePath string) (*os.File, error) {
	dir, _ := path.Split(cachePath)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, os.ModePerm)
}

func GenerateCachePath(rootPath string, url *url.URL) string {
	dir, file := path.Split(url.Path)

	query := url.Query()
	queryPath := BuildQueryPath(&query)

	path := path.Join(
		rootPath,
		url.Host,
		dir,
		queryPath,
		GetShortHash(url),
		file,
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
			queryPath = path.Join(queryPath, fmt.Sprintf("%s=%s", queryKey, queryValue))
		}
	}

	return queryPath
}

func GetShortHash(url *url.URL) string {
	sum := md5.Sum([]byte(url.Path))
	return fmt.Sprintf("%x", sum[:3])
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
