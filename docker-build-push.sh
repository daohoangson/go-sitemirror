#!/bin/bash

set -e

_imageName=${1:-'xfrocks/go-sitemirror'}
_gitHead=`git rev-parse HEAD`
_tagged="$_imageName:$_gitHead"

echo "Building image $_tagged"
docker build . -t "$_imageName" -t "$_tagged" --build-arg SITEMIRROR_COMMIT="$_gitHead"

while true
do
  read -p "Push image? [yN]" yn
  case $yn in
    [Yy]* ) break;;
    * ) exit;;
  esac
done

docker push "$_imageName"
docker push "$_tagged"
