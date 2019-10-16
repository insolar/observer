FROM tsovak/golang as builder

RUN go get -u github.com/golang/dep/cmd/dep

ADD ./ /go/src/github.com/insolar/observer

WORKDIR /go/src/github.com/insolar/observer
RUN make ensure
RUN make build

FROM centos:7 as app
COPY --from=builder /go/src/github.com/insolar/observer/bin/observer /observer
CMD [ "/observer" ]
