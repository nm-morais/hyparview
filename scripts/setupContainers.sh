#!/bin/bash

nContainers=$1
shift 1

if [ -z $SWARM_NET ]; then
  echo "Pls specify env var SWARM_NET"
  exit
fi

if [ -z $DOCKER_IMAGE ]; then
  echo "Pls specify env var DOCKER_IMAGE"
  exit
fi

if [ -z $IPS_FILE ]; then
  echo "Pls specify env var IPS_FILE"
  exit
fi

if [ -z $LATENCY_MAP ]; then
  echo "Pls specify env var LATENCY_MAP"
  exit
fi

if [ -z $SWARM_VOL_DIR ]; then
  echo "Pls specify env var SWARM_VOL_DIR"
  exit
fi

echo "SWARM_NET: $SWARM_NET"
echo "DOCKER_IMAGE: $DOCKER_IMAGE"
echo "IPS_FILE: $IPS_FILE"

n_nodes=0
for var in $@
do
  n_nodes=$((n_nodes+1))
done

if [[ $n_nodes -eq 0 ]]; then
  echo "usage <node_array>"
  exit
fi

echo "number of nodes: $n_nodes"
i=0
echo "Lauching containers..."
while read -r ip name bw
do
  if [[ $i -eq $nContainers ]]; then
    break
  fi
  echo "ip: $ip"
  echo "name: $name"
  idx=$(($i % n_nodes))
  idx=$((idx+1))
  node=${!idx}

  cmd="docker run -e config='/config/exampleConfig.yml' -v $SWARM_VOL_DIR:/tmp/logs -v /lib/modules:/lib/modules -d -t --privileged --cap-add=ALL \
   --net $SWARM_NET \
   --ip $ip \
   --name $name \
   -h $name \
    $DOCKER_IMAGE $i $nContainers $bw -bootstraps='$BOOTSTRAPS' -listenIP=$ip"

  echo "running command: $cmd"

  echo "Starting ${i}. Container $name with ip $ip and name $name on: $node"
  ssh -n $node "$cmd"
  i=$((i+1))
done < "$IPS_FILE"