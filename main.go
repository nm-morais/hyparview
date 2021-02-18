package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	babel "github.com/nm-morais/go-babel/pkg"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/ungerik/go-dry"
	"gopkg.in/yaml.v2"
)

var (
	randomPort *bool
	bootstraps *string
	listenIP   *string
)

func main() {
	randomPort = flag.Bool("rport", false, "choose random port")
	bootstraps = flag.String("bootstraps", "", "choose custom bootstrap nodes (space-separated ip:port list)")
	listenIP = flag.String("listenIP", "", "choose custom ip to listen to")

	flag.Parse()

	conf := readConfFile()

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

	content, err := ioutil.ReadFile("config/exampleConfig.yml")
	if err != nil {
		log.Fatal(err)
	}

	// Convert []byte to string and print to screen
	text := string(content)
	fmt.Println(text)
	if err != nil {
		panic(err)
	}
	conf.LogFolder += fmt.Sprintf("%s_%d/", conf.SelfPeer.Host, conf.SelfPeer.Port)

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

func readConfFile() *HyparviewConfig {
	configFileName := "config/exampleConfig.yml"
	envVars := dry.EnvironMap()
	customConfig, ok := envVars["config"]
	if ok {
		configFileName = customConfig
	}
	f, err := os.Open(configFileName)
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
		for _, ipPortStr := range strings.Split(*arg, " ") {
			split := strings.Split(ipPortStr, ":")
			ip := split[0]
			portInt, err := strconv.ParseInt(split[1], 10, 32)
			if err != nil {
				panic(err)
			}
			bootstrapPeers = append(bootstrapPeers, struct {
				Port int    "yaml:\"port\""
				Host string "yaml:\"host\""
			}{
				Port: int(portInt),
				Host: ip,
			})
		}
		conf.BootstrapPeers = bootstrapPeers
	}
}
