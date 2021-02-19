#!/bin/bash

set -e

if [ -z $SWARM_NET ]; then
  echo "Pls specify env var SWARM_NET"
  exit
fi

if [ -z $DOCKER_IMAGE ]; then
  echo "Pls specify env var DOCKER_IMAGE"
  exit
fi

if [ -z $CONFIG_FILE ]; then
  echo "Pls specify env var CONFIG_FILE"
  exit
fi

if [ -z $LATENCY_MAP ]; then
  echo "Pls specify env var LATENCY_MAP"
  exit
fi

echo "SWARM_NET: $SWARM_NET"
echo "DOCKER_IMAGE: $DOCKER_IMAGE"
echo "CONFIG_FILE: $CONFIG_FILE"

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

echo "Building images..."

currdir=$(pwd)
delete_containers_cmd='docker rm -f $(docker ps -a -q)  > /dev/null || true'
build_cmd="export export LATENCY_MAP=$LATENCY_MAP; export CONFIG_FILE=$CONFIG_FILE export DOCKER_IMAGE=$DOCKER_IMAGE; cd ${currdir}; ./scripts/buildImage.sh"
delete_logs_cmd="docker run --rm -v $SWARM_VOL:/data bash sh -c 'rm -rf /data/*'"
host=$(hostname)
for node in $@; do
  {
    echo "deleting running containers on node: $node" 
    oarsh $node "$delete_containers_cmd"
    echo "done deleting containers on node: $node!"  
  } &
done
wait
sleep 1s

# FOR USE IN NAS
docker run --rm -v $SWARM_VOL:/data bash sh -c 'rm -rf /data/*' || true
# UNCOMMENT FOR LOCAL FILES ON NODES
# for node in $@; do
#   {
#     echo "deleting logs on node: $node" 
#     oarsh $node "$delete_logs_cmd" 
#     echo "done deleting logs on node: $node!"  
#     wait
#   } &
# done
# wait
# sleep 1s

for node in $@; do
  {
    echo "starting build on node: $node" 
    oarsh $node $build_cmd 
    echo "done building image on node: $node!" 
 } &
done
wait
sleep 1s
echo "Deploying with config file:"
cat $CONFIG_FILE

maxcpu=$(nproc)
nContainers=$(wc -l $CONFIG_FILE)
i=0

echo "Lauching containers..."
while read -r ip name
do
  echo "ip: $ip"
  echo "name: $name"
  idx=$(($i % n_nodes))
  idx=$((idx+1))
  node=${!idx}

  cmd="docker run -v $SWARM_VOL:/tmp/logs -d -t --cap-add=NET_ADMIN \
   --net $SWARM_NET \
   --ip $ip \
   --name $name \
   -h $name \
    $DOCKER_IMAGE $i $nContainers"

  # echo "running command: '$cmd'"

  echo "Starting ${i}. Container $name with ip $ip and name $name on: $node"
  oarsh -n $node "$cmd"
  i=$((i+1))
done < "$CONFIG_FILE"