FROM golang:1.20.4-bullseye as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build

FROM debian:bullseye-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/go-sitemirror /usr/local/bin/.

RUN { \
    echo '#!/bin/bash'; \
    echo 'set -e'; \
    echo 'if [ "${1:0:1}" = "-" ]; then'; \
	  echo '  set -- go-sitemirror "$@"'; \
    echo 'fi'; \
    echo 'exec "$@"'; \
  } > /entrypoint.sh \
  && chmod +x /entrypoint.sh

EXPOSE 80
CMD ["go-sitemirror"]
ENTRYPOINT ["/entrypoint.sh"]
VOLUME ["/cache"]
WORKDIR /cache
