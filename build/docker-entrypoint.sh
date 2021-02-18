#!/bin/sh

set -e

echo "env: "
env

echo "Bootstraping TC, args: $1 $2 $3 $4"
bash setupTc.sh $1 $2 $3 $4

echo "Bootstraping hyparview"
shift 4
echo "$@"
./go/bin/hyparview