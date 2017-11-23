#!/bin/bash

set -e

go get -u github.com/golang/dep/cmd/dep
echo 'go get ok'

dep ensure
echo 'dep ok'
