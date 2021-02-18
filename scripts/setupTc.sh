#!/bin/sh

# Credit : https://github.com/pedroAkos

latencyMap=$1
ipsMap=$2
idx=$3
nrIps=$4

ips=""
while read -r ip name
do
  ips="${ips} ${ip}"
done < "$ipsMap"


function setuptc {
  cmd="tc qdisc add dev eth0 root handle 1: htb"
  echo "$cmd"
  eval $cmd
  j=1

  for n in $1
  do
    cmd="tc class add dev eth0 parent 1: classid 1:${j}1 htb rate 1000mbit"
    echo "$cmd"
    eval $cmd
    targetIp=$(echo ${ips} | cut -d' ' -f${j})
    cmd="tc qdisc add dev eth0 parent 1:${j}1 netem delay ${n}ms"
    echo "$cmd"
    eval $cmd
    cmd="tc filter add dev eth0 protocol ip parent 1:0 prio 1 u32 match ip dst ${targetIp} flowid 1:${j}1"
    echo "$cmd"
    eval $cmd
    if [[ $j -eq $nrIps ]]; then
      break
    fi
    j=$((j+1))
  done

  echo "Set up delays to ${j} nodes"
  return
}

i=0
echo "Setting up tc emulated network for node ${idx}..."
while read -r line
do
  if [ $idx == $i ]; then
    echo "Setting up with line ${line}"
    setuptc "${line}"
    break
  fi
  i=$((i+1))
  if [[ $i -eq $nrIps ]]; then
      break
  fi
done < "$latencyMap"

echo "Done."