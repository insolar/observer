FROM golang:1.12-buster as builder
WORKDIR /observer

COPY ./ /observer
RUN make build

FROM debian:buster as app
COPY $PWD/scripts/migrations /migrations
COPY --from=builder /observer/bin/observer /usr/local/bin/observer
COPY --from=builder /observer/bin/api /usr/local/bin/observer-api
COPY --from=builder /observer/bin/stats-collector /usr/local/bin/observer-stats-collector
COPY --from=builder /observer/bin/migrate /usr/local/bin/observer-migrate
