#!/bin/sh

_srcPath='/go/src/github.com/daohoangson/go-sitemirror'

docker run --rm -it \
  -v "$PWD:$_srcPath" \
  -w "$_srcPath" \
  golang:1.9.2-stretch bash
