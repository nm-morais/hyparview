FROM golang:latest AS builder

# RUN apk update && apk add --no-cache git
# RUN apk add --no-cache make gcc gawk bison linux-headers libc-dev
#hyparview
WORKDIR $GOPATH/src/github.com/nm-morais/hyparview
COPY . .

# RUN go get -d -v ./...
RUN go mod download
#build
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/hyparview cmd/hyparview/*.go

# EXECUTABLE IMG
FROM debian:stable-slim as hyparview

# RUN apk add iproute2-tc
RUN apt update 2>/dev/null | grep -P "\d\K upgraded" ; apt install iproute2 -y 2>/dev/null

COPY scripts/setupTc.sh /setupTc.sh
COPY build/docker-entrypoint.sh /docker-entrypoint.sh
COPY --from=builder /go/bin/hyparview /go/bin/hyparview

# ARG LATENCY_MAP=config/inet100Latencies_x0.04.txt
# ARG CONFIG_FILE=config/generated_config.txt

ARG LATENCY_MAP
ARG CONFIG_FILE

COPY ${LATENCY_MAP} /latencyMap.txt
COPY ${CONFIG_FILE} /config.txt
RUN chmod +x /setupTc.sh /docker-entrypoint.sh /go/bin/hyparview

ENTRYPOINT ["/docker-entrypoint.sh", "/latencyMap.txt", "/config.txt"]