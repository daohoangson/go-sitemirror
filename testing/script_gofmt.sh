#!/bin/bash

set -e

diff -u <(echo -n) <(gofmt -d .)
echo 'gofmt ok'
