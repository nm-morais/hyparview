#!/bin/sh

(cd ../go-babel && ./scripts/buildImage.sh)

docker build --build-arg LATENCY_MAP=$LATENCY_MAP --build-arg IPS_FILE=$IPS_FILE  --file build/Dockerfile -t nmmorais/hyparview:latest .