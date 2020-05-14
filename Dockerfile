ARG GOLANG_VERSION=1.12
FROM golang:${GOLANG_VERSION} AS build
WORKDIR /observer

COPY ./ /observer
RUN make build

FROM debian:buster-slim as app
RUN apt update && apt install -y ca-certificates && apt-get clean all
COPY $PWD/scripts/migrations /migrations
COPY --from=build /observer/bin/observer /observer/bin/api /observer/bin/migrate /usr/local/bin/
