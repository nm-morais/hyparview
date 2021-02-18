FROM golang:1.16 AS builder

WORKDIR $GOPATH/src/github.com/nm-morais/hyparview
COPY . .
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/hyparview *.go

FROM debian:stable-slim as hyparview
COPY --from=builder /go/bin/hyparview /go/bin/hyparview
COPY config /config

RUN chmod +x /go/bin/hyparview

CMD ["/go/bin/hyparview"]