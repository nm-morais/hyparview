#!/bin/sh

latencyMap=$1
ipsMap=$2
idx=$3
nrIps=$4
orig_bandwidth=$5

if [ -z "$orig_bandwidth" ]; then
  orig_bandwidth=1000
fi

bandwidth=$((orig_bandwidth / 2))

echo "Bandwidth limit is $orig_bandwidth but will apply $bandwidth to compensate for applying on send and receive"

ips=""
while read -r ip name
do
  ips="${ips} ${ip}"
done <"$ipsMap"

# shellcheck disable=SC2112
function setuptc() {

  in_bandwidth=$bandwidth

  cmd="/sbin/modprobe ifb numifbs=1"
  echo "$cmd"
  eval $cmd

  cmd="ip link add ifb0 type ifb"
  echo "$cmd"
  eval $cmd

  cmd="ip link set dev ifb0 up"
  echo "$cmd"
  eval $cmd

  cmd="tc qdisc add dev eth0 handle ffff: ingress"
  echo "$cmd"
  eval $cmd

  cmd="tc filter add dev eth0 parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev ifb0"
  echo "$cmd"
  eval $cmd

  cmd="tc qdisc add dev ifb0 root handle 1: htb default 1"
  echo "$cmd"
  eval $cmd

  cmd="tc class add dev ifb0 parent 1: classid 1:1 htb rate ${in_bandwidth}mbit"
  echo "$cmd"
  eval $cmd

  #---------------------------------

  cmd="tc qdisc add dev eth0 root handle 1: htb"
  echo "$cmd"
  eval $cmd
  j=1

  cmd="tc class add dev eth0 parent 1: classid 1:1 htb rate ${bandwidth}mbit"
  echo "$cmd"
  eval $cmd

  for n in $1; do
    cmd="tc class add dev eth0 parent 1: classid 1:${j}1 htb rate ${bandwidth}mbit"
    echo "$cmd"
    eval $cmd
    targetIp=$(echo ${ips} | cut -d' ' -f${j})
    cmd="tc qdisc add dev eth0 parent 1:${j}1 netem delay ${n}ms"
    echo "$cmd"
    eval $cmd
    cmd="tc filter add dev eth0 protocol ip parent 1:0 prio 1 u32 match ip dst $targetIp flowid 1:${j}1"
    echo "$cmd"
    eval $cmd
    if [[ $j -eq $nrIps ]]; then
      break
    fi
    j=$((j + 1))
  done
}

i=0
echo "Setting up tc emulated network..."
while read -r line; do
  if [ $idx -eq $i ]; then
    setuptc "$line"
    break
  fi
  i=$((i + 1))
  if [[ $i -eq $nrIps ]]; then
    break
  fi
done <"$latencyMap"

echo "Done."