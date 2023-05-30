#!/bin/bash

set -e

export GO111MODULE=on
export TESTING_LOGGER_LEVEL=fatal

go vet ./...
echo 'go vet ok'

go test -v -race ./...
echo 'go test ok'
