# go-sitemirror
Website mirror app with priority for response consistency.

[![GoDoc](https://godoc.org/github.com/daohoangson/go-sitemirror/engine?status.svg)](https://godoc.org/github.com/daohoangson/go-sitemirror/engine)
[![Go Report Card](https://goreportcard.com/badge/github.com/daohoangson/go-sitemirror)](https://goreportcard.com/report/github.com/daohoangson/go-sitemirror)
[![Travis CI](https://api.travis-ci.org/daohoangson/go-sitemirror.svg?branch=master)](https://travis-ci.org/daohoangson/go-sitemirror)

## Goal
Easy to setup and run a mirror which copies content from some where else and provides a near exact web browsing experience in case the source server / network goes down.

## Ideas
1. All web assets should be downloaded and have with their metadata intact (content type etc.)
1. Links should be followed with some restriction to save resources.
1. Cached data should be refresh periodically.
1. A web server should be provided to serve visitor.

## Usage

### Mirror everything at `:8080`
Go to http://localhost:8080/https/github.com/ to see GitHub home page

```
  go-sitemirror -p 8080
```

### Mirror GitHub at `:8081`
Go to http://localhost:8081/ to see GitHub home page

```
  go-sitemirror -mirror https://github.com \
    -mirror-port 8081 \
    -no-cross-host \
    -whitelist github.com
```

* `-no-cross-host` to not modify assets urls from other domains
* `-whitelist` because we don't serve anything other than github.com anyway

### All flags

```
  -auto-download-depth value
    	Maximum link depth for auto downloads, default=1 (default 1)
  -auto-refresh duration
    	Interval for url auto refreshes, default=no refresh
  -cache-bump duration
    	Validity of cache bump (default 1m0s)
  -cache-path string
    	HTTP Cache path (default working directory)
  -cache-ttl duration
    	Validity of cached data (default 10m0s)
  -header value
    	Custom request header, must be 'key=value'
  -log value
    	Logging output level (default 4)
  -mirror value
    	URL to mirror, multiple urls are supported
  -mirror-port value
    	Port to mirror a single site, each port number should immediately follow its URL. For url that doesn't have any port, it will still be mirrored but without a web server.
  -no-cross-host
    	Disable cross-host links
  -port int
    	Port to mirror all sites (default -1)
  -rewrite value
    	Link rewrites, must be 'source.domain.com=http://target.domain.com/some/path'
  -whitelist value
    	Restricted list of crawlable hosts
  -workers value
    	Number of download workers (default 4)
```
