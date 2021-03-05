#!/bin/sh

set -e

echo "env: "
env
currdir=$(pwd)
echo "curr dir: $currdir"

echo "Bootstraping TC, args: $1 $2 $3 $4"
bash /setupTc.sh $1 $2 $3 $4

echo "Bootstraping hyparview"
shift 5
echo "starting with args: $@"
./go/bin/hyparview "$@"