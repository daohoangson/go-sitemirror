FROM golang:1.7-alpine

# overwrite this via -auto-download-depth
ENV SITEMIRROR_AUTO_DOWNLOAD_DEPTH "0"
# overwrite this via -cache-path or just mount from docker host to this directory
ENV SITEMIRROR_CACHE_PATH "/cache"
# overwrite this via -port
ENV SITEMIRROR_PORT "80"

ENV SITEMIRROR_SOURCE_PATH "/go/src/github.com/daohoangson/go-sitemirror"

COPY . "$SITEMIRROR_SOURCE_PATH"

RUN cd "$SITEMIRROR_SOURCE_PATH" \
  && go install \
  && { \
    echo '#!/bin/sh'; \
    \
    echo 'set -e'; \
    \
    echo 'if [ "${1:0:1}" = "-" ]; then'; \
	  echo '  set -- go-sitemirror "$@"'; \
    echo 'fi'; \
    \
    echo 'exec "$@"'; \
  } > /entrypoint.sh \
  && chmod +x /entrypoint.sh

EXPOSE 80
CMD ["go-sitemirror"]
ENTRYPOINT ["/entrypoint.sh"]
VOLUME ["/cache"]

# Mirror everything at :8080
# Go to http://localhost:8080/https/github.com/ to see GitHub home page
# docker run --rm -it -p 8080:80 daohoangson/go-sitemirror

# Mirror https://github.com at :8081
# Go to http://localhost:8081/ to see GitHub home page
# Use `-no-cross-host` not modify assets urls from other domains
# Use `-whitelist` because we don't serve anything other than github.com anyway
# docker run --rm -it -p 8081:81 daohoangson/go-sitemirror \
#   -mirror https://github.com -mirror-port 81 \
#   -no-cross-host \
#   -whitelist github.com
