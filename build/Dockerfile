FROM golang:latest AS builder
WORKDIR $GOPATH/src/github.com/nm-morais/hyparview
COPY --from=nmmorais/go-babel:latest /src/github.com/nm-morais/go-babel ../go-babel/
COPY . .
RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/hyparview *.go

# EXECUTABLE IMG
FROM debian:stable-slim as hyparview

# RUN apk add iproute2-tc
RUN apt update 2>/dev/null | grep -P "\d\K upgraded" ; apt install iproute2 -y 2>/dev/null

COPY scripts/setupTc.sh /setupTc.sh
COPY build/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder /go/bin/hyparview /go/bin/hyparview
COPY config/exampleConfig.yml /config/exampleConfig.yml

ARG LATENCY_MAP
ARG IPS_FILE

COPY ${LATENCY_MAP} /latencyMap.txt
COPY ${IPS_FILE} /config.txt

RUN chmod +x /setupTc.sh /docker-entrypoint.sh /go/bin/hyparview

ENTRYPOINT ["/docker-entrypoint.sh", "/latencyMap.txt", "/config.txt"]