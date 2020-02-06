ARG GOLANG_VERSION=1.12
FROM golang:${GOLANG_VERSION} AS build
WORKDIR /observer

COPY ./ /observer
RUN make build

FROM debian:buster-slim as app
COPY $PWD/scripts/migrations /migrations
COPY --from=build /observer/bin/observer /observer/bin/api /observer/bin/stats-collector /observer/bin/migrate /observer/bin/binance-collector /observer/bin/coin-market-cap-collector /usr/local/bin/
