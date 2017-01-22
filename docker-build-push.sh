#!/bin/sh

set -e

IMAGE_NAME="$1"
if [ "x$IMAGE_NAME" = "x" ]; then
  echo 'Please specify image name to build and push'
  exit 1
fi

GIT_HEAD=`git rev-parse HEAD`

echo "Building image $IMAGE_NAME:$GIT_HEAD"
docker build . -t "$IMAGE_NAME" -t "$IMAGE_NAME:$GIT_HEAD" --build-arg SITEMIRROR_COMMIT="$GIT_HEAD"

read -p "Push image? [yN]" yn
case $yn in
	[Yy]* ) break;;
	* ) exit;;
esac

docker push "$IMAGE_NAME"
docker push "$IMAGE_NAME:$GIT_HEAD"
