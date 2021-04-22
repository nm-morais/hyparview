package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	babel "github.com/nm-morais/go-babel/pkg"
	"github.com/nm-morais/go-babel/pkg/peer"
	"gopkg.in/yaml.v2"
)

var (
	randomPort   *bool
	bootstraps   *string
	listenIP     *string
	confFilePath *string
)

func main() {
	randomPort = flag.Bool("rport", false, "choose random port")
	bootstraps = flag.String("bootstraps", "", "choose custom bootstrap nodes (space-separated ip:port list)")
	listenIP = flag.String("listenIP", "", "choose custom ip to listen to")
	confFilePath = flag.String("conf", "config/exampleConfig.yml", "specify conf file path")
	fmt.Println("ARGS:", os.Args)
	flag.Parse()
	fmt.Println(*confFilePath)
	conf := readConfFile(*confFilePath)

	if *randomPort {
		fmt.Println("Setting custom port")
		freePort, err := GetFreePort()
		if err != nil {
			panic(err)
		}
		conf.SelfPeer.Port = freePort
	}

	ParseBootstrapArg(bootstraps, conf)
	if listenIP != nil && *listenIP != "" {
		conf.SelfPeer.Host = *listenIP
	}

	conf.LogFolder += fmt.Sprintf("%s:%d/", conf.SelfPeer.Host, conf.SelfPeer.Port)
	if listenIP != nil && *listenIP != "" {
		conf.SelfPeer.Host = *listenIP
	}
	protoManagerConf := babel.Config{
		Silent:    false,
		LogFolder: conf.LogFolder,
		SmConf: babel.StreamManagerConf{
			BatchMaxSizeBytes: 2000,
			BatchTimeout:      time.Second,
			DialTimeout:       time.Millisecond * time.Duration(conf.DialTimeoutMiliseconds),
		},
		Peer: peer.NewPeer(net.ParseIP(conf.SelfPeer.Host), uint16(conf.SelfPeer.Port), 0),
	}

	p := babel.NewProtoManager(protoManagerConf)
	p.RegisterListenAddr(&net.TCPAddr{IP: protoManagerConf.Peer.IP(), Port: int(protoManagerConf.Peer.ProtosPort())})
	p.RegisterListenAddr(&net.UDPAddr{IP: protoManagerConf.Peer.IP(), Port: int(protoManagerConf.Peer.ProtosPort())})
	p.RegisterProtocol(NewHyparviewProtocol(p, conf))
	p.StartSync()
}

func readConfFile(path string) *HyparviewConfig {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	cfg := &HyparviewConfig{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}

func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

func ParseBootstrapArg(arg *string, conf *HyparviewConfig) {
	if arg != nil && *arg != "" {
		bootstrapPeers := []struct {
			Port int    "yaml:\"port\""
			Host string "yaml:\"host\""
		}{}
		fmt.Println("Setting custom bootstrap nodes")
		for _, ipStr := range strings.Split(*arg, " ") {
			bootstrapPeers = append(bootstrapPeers, struct {
				Port int    "yaml:\"port\""
				Host string "yaml:\"host\""
			}{
				Port: int(conf.SelfPeer.Port),
				Host: ipStr,
			})
		}
		conf.BootstrapPeers = bootstrapPeers
	}
}
