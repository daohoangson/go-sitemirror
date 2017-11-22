#!/bin/sh

_srcPath='/go/src/github.com/daohoangson/go-sitemirror'

docker run --rm -it \
  -e "SITEMIRROR_AUTO_DOWNLOAD_DEPTH=0" \
  -e "SITEMIRROR_CACHE_PATH=/cache" \
  -e "SITEMIRROR_PORT=80" \
  -p "8081:80" \
  -v "$PWD:$_srcPath" \
  -v "$PWD/cache:/cache" \
  -w "$_srcPath" \
  golang:1.9.2-stretch bash
