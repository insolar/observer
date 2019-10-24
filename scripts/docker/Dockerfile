FROM tsovak/golang as builder

ADD . /observer

WORKDIR /observer
RUN make build

FROM alpine:3.4
COPY --from=builder /go/src/github.com/insolar/observer/bin/observer /bin/observer
ENTRYPOINT ["/bin/observer"]
