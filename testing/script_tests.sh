#!/bin/bash

set -e

export TESTING_LOGGER_LEVEL=fatal

go vet $(go list ./...)
echo 'go vet ok'

go test -v -race $(go list ./...)
echo 'go test ok'
