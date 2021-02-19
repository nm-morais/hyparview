from netaddr import IPNetwork
import multiprocessing as mp
import os
import subprocess
import json

cidr_provided = "10.10.0.0/16"
network = "test_net"


def run_cmd_with_try(cmd, env=dict(os.environ), stdout=subprocess.DEVNULL):
    print(f"Running | {cmd} | LOCAL")
    cp = subprocess.run(cmd, shell=True, stdout=stdout, env=env)
    if cp.stderr is not None:
        raise Exception(cp.stderr)


def exec_cmd_with_output(cmd):
    (status, out) = subprocess.getstatusoutput(cmd)
    if status != 0:
        print(out)
        exit(1)
    return out


ips = [str(ip) for ip in IPNetwork(cidr_provided)]
# Ignore first two IPs since they normally are the NetAddr and the Gateway, and ignore last one since normally it's the
# broadcast IP
ips = ips[2:-1]
entrypoints_ips = set()
rm_anchor_cmd = f"docker rm -f anchor || true"
run_cmd_with_try(rm_anchor_cmd)
print(f"Setting up anchor")
anchor_cmd = f"docker run -d --name=anchor --network={network} alpine sleep 30m"
run_cmd_with_try(anchor_cmd)

"""
Output is like:
"lb-swarm-network": {
    "Name": "swarm-network-endpoint",
    "EndpointID": "ab543cead9c04275a95df7632165198601de77c183945f2a6ab82ed77a68fdd3",
    "MacAddress": "02:42:c0:a8:a0:03",
    "IPv4Address": "192.168.160.3/20",
    "IPv6Address": ""
}
so we split at max once thus giving us only the value and not the key
"""

get_entrypoint_cmd = f"docker network inspect {network} | grep 'lb-{network}' -A 6"
output = exec_cmd_with_output(get_entrypoint_cmd).strip().split(" ", 1)[1]

entrypoint_json = json.loads(output)

entrypoints_ips.add(entrypoint_json["IPv4Address"].split("/")[0])
get_anchor_cmd = f"docker network inspect {network} | grep 'anchor' -A 5 -B 1"
output = exec_cmd_with_output(get_anchor_cmd).strip().split(" ", 1)[1]
if output[-1] == ",":
    output = output[:-1]

anchor_json = json.loads(output)
entrypoints_ips.add(anchor_json["IPv4Address"].split("/")[0])

print(f"entrypoints: {entrypoints_ips}")
f = open("config/ips.txt", "w")
added = 0
for i, ip in enumerate(reversed(ips)):
    if ip not in entrypoints_ips:
        added += 1
        f.write(f"{ip} node{i}\n")

    if i == 1000:
        break
f.close()

print("Output file written to: config/ips.txt")
