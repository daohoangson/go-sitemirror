#!/bin/bash

set -e

./testing/script_gofmt.sh
./testing/script_dep.sh
./testing/script_tests.sh
