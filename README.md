# go-sitemirror
Website mirror app with priority for response consistency.

![](https://api.travis-ci.org/daohoangson/go-sitemirror.svg?branch=master)

## Goal
Easy to setup and run a mirror which copies content from some where else and provides a near exact web browsing experience in case the source server / network goes down.

## Ideas
1. All web assets should be downloaded and have with their metadata intact (content type etc.)
1. Links should be followed with some restriction to save resources.
1. Cached data should be refresh periodically.
1. A web server should be provided to serve visitor.

## Usage
`go-sitemirror -mirror http://github.com -port 8080`
