#!/usr/bin/env python3


from multiprocessing import Process, Manager
import json
import networkx as nx
import pandas as pd
from dateutil.parser import parse
import numpy as np
import os
import argparse
import statistics

latency_collection_tag = "<latency_collection>"
inview_tag = "<inView>"


def get_file_list(log_folder):
    paths = []
    node_folders = os.listdir(log_folder)
    for node_folder in node_folders:
        if not node_folder.startswith("10.10"):
            print(f"skipping folder: {node_folder}")
            continue
        print(f"parsing folder: {node_folder}")
        node_path = "{}/{}".format(log_folder, node_folder)
        for node_file in os.listdir(node_path):
            if node_file == "all.log":
                paths.append("{}/{}".format(node_path, node_file))
    return paths, len(paths)


def read_latencies_file(file_path):
    f = open(file_path, "r")
    node_latencies = []
    for aux in f.readlines():
        line = aux.strip()
        split = line.split(" ")
        node_latencies.append([float(lat) for lat in split])
    return node_latencies


def read_coords_file(file_path, node_ips):
    f = open(file_path, "r")
    node_coords = {}
    i = 0

    for aux in f.readlines():
        line = aux.strip()
        split = line.split(" ")
        if i == len(node_ips):
            break
        node_coords[node_ips[i]] = [float(lat) for lat in split]
        i += 1
    return node_coords


def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument("--ips_file", required=True, metavar='ips_file',
                        type=str, help="the log file")

    parser.add_argument("--latencies_file", required=True, metavar='latencies_file',
                        type=str,  help="the latencies file")

    parser.add_argument("--coords_file", required=True, metavar='coords_file',
                        type=str,  help="the coordinates file")

    parser.add_argument("--logs_folder", required=True,  metavar='logs_folder',
                        type=str,   help="the folder where logs are contained")

    parser.add_argument("--output_path", required=True, metavar='output_path',
                        type=str,  help="the output file")

    args = parser.parse_args()
    return args


def extractJsonFromLine(line, tag):
    line_cut = line[line.index(tag) + len(tag) + 1:len(line) - 1]
    line_cut = line_cut.replace("\\", "")
    return json.loads(line_cut)


def extractInView(line):
    # print("line", line)
    line_cut = line[line.index(inview_tag) + len(inview_tag) + 1:len(line) - 1]
    line_cut = line_cut.replace("\\", "")
    return json.loads(line_cut)


def getLatForIpPair(ip1, ip2, latency_map):
    return latency_map[ip1][ip2]


def parse_file(file, node_ip, node_infos, latency_map):
    f = open(file, "r")

    node_measurements = {
        "ip": [],
        "peers": [],
        "latencies": [],
        "latency_avg": [],
        "degree": [],
        "latency_avg_global": [],
        "timestamp_dt": [],
        "ctrl_msgs_sent": [],
        "ctrl_msgs_rcvd": [],
        "app_msgs_sent": [],
        "app_msgs_rcvd": [],
    }

    ctrl_msgs_sent = 0
    ctrl_msgs_rcvd = 0
    app_msgs_rcvd = 0
    app_msgs_sent = 0

    for aux in f.readlines():
        line = aux.strip()
        levelStr = " level="
        timeStr = "time="

        try:
            if "<control-messages-stats>" in line:
                stats = extractJsonFromLine(line, "<control-messages-stats>")
                ctrl_msgs_rcvd = stats["ControlMessagesReceived"]
                ctrl_msgs_sent = stats["ControlMessagesSent"]
                # print(ctrl_msgs_rcvd)
                pass

            if "<app-messages-stats>" in line:
                stats = extractJsonFromLine(line, "<app-messages-stats>")
                app_msgs_rcvd = stats["ApplicationalMessagesReceived"]
                app_msgs_sent = stats["ApplicationalMessagesSent"]
                app_msgs_sent = app_msgs_sent["1000"]
                app_msgs_rcvd = app_msgs_rcvd["1000"]
                # print(stats)
                pass
        except Exception as e:
            print(f"Exception: {e}")

        if inview_tag in line:
            ts = line[line.find(timeStr) + len(timeStr) + 1:]
            ts = ts[:line.find(levelStr) - len(levelStr) - 1]
            ts_parsed = parse(ts)
            inView = extractInView(line)
            if len(inView) > 0:
                node_measurements["ctrl_msgs_rcvd"].append(ctrl_msgs_rcvd)
                node_measurements["ctrl_msgs_sent"].append(ctrl_msgs_sent)
                node_measurements["app_msgs_rcvd"].append(app_msgs_rcvd)
                node_measurements["app_msgs_sent"].append(app_msgs_sent)

                node_measurements["peers"].append([p["ip"] for p in inView])
                node_measurements["latencies"].append(
                    [getLatForIpPair(p["ip"], node_ip, latency_map) for p in inView])
                node_measurements["timestamp_dt"].append(
                    pd.to_datetime(ts_parsed))
                node_measurements["ip"].append(node_ip)
                node_measurements["degree"].append(len(inView))
                node_measurements["latency_avg"].append(
                    np.mean([getLatForIpPair(p["ip"], node_ip, latency_map) for p in inView]))

                # node_measurements["latency_avg"].append(0)
    # print(node_measurements["latencies"])
    node_infos[node_ip] = node_measurements


def parse_file_list(file_list, latency_map):
    manager = Manager()
    d = manager.dict()
    processes = []
    for file in file_list:
        node_name = str(file.split("/")[-2])
        node_ip = node_name.split(":")[0]
        p = Process(target=parse_file, args=(
            file, node_ip, d, latency_map))
        p.start()
        processes.append(p)

    for p in processes:
        p.join()
    return d


def plot_avg_latency_all_nodes_over_time(df, output_path):
    import matplotlib.pyplot as plt
    fig, ax = plt.subplots()
    resampled = df[["latency_avg", "latency_avg_global"]
                   ].resample('15s').mean()
    resampled.drop(resampled.tail(1).index,
                   inplace=True)
    resampled.plot(ax=ax)

    ax.set(xlabel='time (s)', ylabel='latency (ms)',
           title='Average latency over time in active view')
    ax.grid()
    print(f"saving average latency over time in active view to: {output_path}")
    fig.savefig(f"{output_path}latencies_over_time.svg", dpi=1200)


def plot_avg_out_degree_all_nodes_over_time(df, output_path):
    import matplotlib.pyplot as plt
    fig, ax = plt.subplots()
    resampled = df[["degree"]].resample('15s').mean()

    resampled.drop(resampled.tail(1).index,
                   inplace=True)
    resampled.plot(ax=ax)
    ax.set(xlabel='time (s)', ylabel='degree',
           title='Average degree of nodes over time')
    ax.grid()
    print(f"saving average degree of nodes over time to: {output_path}")
    fig.savefig(f"{output_path}degree_over_time.svg", dpi=1200)


def plot_avg_msg_sent_all_nodes_over_time(df, output_path):
    import matplotlib.pyplot as plt
    fig, ax = plt.subplots()
    print(df)
    filter = ["ctrl_msgs_sent", "ctrl_msgs_rcvd",
              "app_msgs_sent", "app_msgs_rcvd"]
    resampled = df[filter].resample('15s').mean()
    print(resampled)
    resampled.drop(resampled.tail(1).index,
                   inplace=True)
    resampled.plot(ax=ax)
    ax.set(xlabel='time (s)', ylabel='number of messages',
           title='Average number of messages sent over time')
    ax.grid()
    print(
        f"saving average number of messages sent over time to: {output_path}")
    fig.savefig(f"{output_path}msgs_over_time.svg", dpi=1200)


def plot_topology(node_infos, coordinates, output_path):
    import matplotlib.pyplot as plt
    import networkx as nx
    fig, ax = plt.subplots()
    G = nx.DiGraph()
    for k in node_infos:
        curr = node_infos[k]
        edges = [(k, ip) for ip in curr["peers"][-1]]
        coord_pair = (coordinates[k][0], coordinates[k][1])
        # print(str(coord_pair))
        G.add_node(k, pos=coord_pair)
        G.add_edges_from(edges)

    # Specify the edges you want here

    pos = nx.get_node_attributes(G, 'pos')
    options = {
        'node_color': 'blue',
        'node_size': 5,
        'width': 0.2,
    }
    nx.draw(G, pos, **options)
    fig.savefig(f"{output_path}topology.svg", dpi=100)
    print(f"saving topology to: {output_path}")


def plot_degree_hist_last_sample(node_infos, output_path):
    import matplotlib.pyplot as plt
    fig, ax = plt.subplots()
    node_degrees = []
    node_in_degrees = []
    max_degree = -10
    for info in node_infos:
        aux = 0
        for info2 in node_infos:
            for peer in node_infos[info2]["peers"][-1]:
                if peer == info:
                    aux += 1
                    # print(node_infos[info].keys())
                    # print(node_infos[info]["degree"][-1])
                    # print(info, node_infos[info]["degree"][-1])
        max_degree = max(max_degree, node_infos[info]["degree"][-1])
        max_degree = max(max_degree, aux)
        node_degrees.append(node_infos[info]["degree"][-1])
        node_in_degrees.append(aux)
        # {"degree":, "ip": node_infos[info]["ip"]})

    # density=False would make counts
    bins = range(0, max_degree + 2)
    counts, bins, patches = plt.hist(
        node_in_degrees, bins=bins, label='in degree')
    counts, bins, patches = plt.hist(
        node_degrees, bins=bins,  label='out degree')
    plt.legend(loc='upper right')
    maxCount = 0
    for curr in counts:
        maxCount = max(maxCount, curr)
    ax.grid()
    yTticks = range(0, int(maxCount) + 3, 5)
    # ax.set_xticks(bins)
    ax.set_yticks(yTticks)
    ax.set(xlabel='Degree of nodes', ylabel='number of nodes',
           title='Histogram of degree of nodes in last sample')
    print(f"saving histogram of degree of nodes in last sample: {output_path}")
    fig.savefig(f"{output_path}hist_degree.svg", dpi=1200)


def read_conf_file(file_path):
    f = open(file_path, "r")
    node_ips = []
    for aux in f.readlines():
        line = aux.strip()
        split = line.split(" ")
        node_ip = split[0]
        node_ips.append(node_ip)
    return node_ips


def build_latencies_map(latencies, node_ips):
    latency_map = {}
    for ip in node_ips:
        latency_map[ip] = {}

    for idx1,  ip in enumerate(node_ips):
        for idx2, ip2 in enumerate(node_ips):
            latency_map[ip][ip2] = latencies[idx1][idx2]

    for i, v in enumerate(latency_map):
        print(v, latency_map[v])
        if i == 0:
            break

    return latency_map


def main():
    args = parse_args()
    print("args: ", args)
    node_ips = read_conf_file(args.ips_file)
    latencies = read_latencies_file(args.latencies_file)
    coords = read_coords_file(args.coords_file, node_ips)
    print(coords)
    latency_map = build_latencies_map(latencies, node_ips)
    file_list, n_nodes = get_file_list(args.logs_folder)
    print(f"Processing {n_nodes} nodes")
    node_infos = parse_file_list(
        file_list=file_list, latency_map=latency_map)
    system_lat_avg = 0

    for node_lats in latencies[:n_nodes]:
        node_lat_avg = 0
        for lat in node_lats[:n_nodes]:
            node_lat_avg += lat / n_nodes
        # print(node_lat_avg)
        system_lat_avg += node_lat_avg / n_nodes

    pd_data = {
        "ip": [],
        "latency_avg": [],
        "timestamp": [],
        "latency_avg_global": [],
        "degree": [],
        "peers": [],
        "coordinates": [],
        "ctrl_msgs_sent": [],
        "ctrl_msgs_rcvd": [],
        "app_msgs_sent": [],
        "app_msgs_rcvd": [],
    }

    for idx, k in enumerate(node_infos):
        pd_data["degree"] += node_infos[k]["degree"]
        pd_data["peers"] += node_infos[k]["peers"]
        pd_data["ip"] += node_infos[k]["ip"]
        pd_data["latency_avg"] += node_infos[k]["latency_avg"]
        pd_data["timestamp"] += node_infos[k]["timestamp_dt"]

        pd_data["ctrl_msgs_sent"] += node_infos[k]["ctrl_msgs_sent"]
        pd_data["ctrl_msgs_rcvd"] += node_infos[k]["ctrl_msgs_rcvd"]
        pd_data["app_msgs_sent"] += node_infos[k]["app_msgs_sent"]
        pd_data["app_msgs_rcvd"] += node_infos[k]["app_msgs_rcvd"]

        pd_data["latency_avg_global"] += [system_lat_avg] * \
            len(node_infos[k]["timestamp_dt"])
        pd_data["coordinates"] += [coords[k]] * \
            len(node_infos[k]["timestamp_dt"])

    df = pd.DataFrame(pd_data)
    df.index = df["timestamp"]
    print(df)
    print(df["latency_avg"])

    # print(df)
    print("system_lat_avg:", system_lat_avg)
    # plot_degree_hist_last_sample(
    #     node_infos=node_infos, output_path=args.output_path)

    plot_avg_latency_all_nodes_over_time(df=df, output_path=args.output_path)
    plot_avg_out_degree_all_nodes_over_time(
        df=df, output_path=args.output_path)
    plot_avg_msg_sent_all_nodes_over_time(df=df, output_path=args.output_path)

    # plot_topology(node_infos=node_infos, coordinates=coords,
    #               output_path=args.output_path)
    # plotConfigMapAndConnections(node_positions, node_ids, parent_edges,
    #                             landmarks, latencies, args.output_path)


if __name__ == "__main__":
    main()
