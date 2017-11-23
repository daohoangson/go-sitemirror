FROM golang:1.9.2-stretch as builder

ARG SITEMIRROR_COMMIT=undefined

ENV SITEMIRROR_SOURCE_PATH "/go/src/github.com/daohoangson/go-sitemirror"

COPY . "$SITEMIRROR_SOURCE_PATH"

RUN cd "$SITEMIRROR_SOURCE_PATH" \
  && go install -ldflags "-X github.com/daohoangson/go-sitemirror/crawler.version=$SITEMIRROR_COMMIT"

FROM debian:stretch-slim

RUN apt-get update \
    && apt-get install -y \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/bin/go-sitemirror /usr/local/bin/.

RUN { \
    echo '#!/bin/bash'; \
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
WORKDIR "/cache"
