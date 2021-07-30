FROM golang:1.16.6-alpine3.14 as builder
ADD . /go/src/github.com/SpencerMalone/alcides
WORKDIR /go/src/github.com/SpencerMalone/alcides
ENV GOPATH /go/
RUN go build -o alcides ./cmd/alcides

FROM alpine
WORKDIR /
COPY --from=builder /go/src/github.com/SpencerMalone/alcides/alcides .
CMD ["./alcides"]