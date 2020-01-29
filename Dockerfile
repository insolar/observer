ARG GOLANG_VERSION=1.12
FROM golang:${GOLANG_VERSION} AS build
WORKDIR /observer

COPY ./ /observer
RUN make build

FROM debian:buster-slim as app
COPY $PWD/scripts/migrations /migrations
COPY --from=build /observer/bin/observer /usr/local/bin/observer
COPY --from=build /observer/bin/api /usr/local/bin/observer-api
COPY --from=build /observer/bin/stats-collector /usr/local/bin/observer-stats-collector
COPY --from=build /observer/bin/migrate /usr/local/bin/observer-migrate
