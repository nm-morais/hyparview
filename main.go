package main

import (
	"flag"
	"net"
	"os"
	"time"

	babel "github.com/nm-morais/go-babel/pkg"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/ungerik/go-dry"
	"gopkg.in/yaml.v2"
)

type HyparviewConfig struct {
	SelfPeer struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"self"`
	BootstrapPeers []struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"bootstrap"`
	dialTimeoutMiliseconds int    `yaml:"dialTimeoutMiliseconds"`
	LogFolder              string `yaml:"logFolder"`
	joinTimeSeconds        int    `yaml:"joinTimeSeconds"`

	activeViewSize  int `yaml:"activeView"`
	passiveViewSize int `yaml:"passiveView"`
	ARWL            int `yaml:"arwl"`
	PRWL            int `yaml:"pwrl"`

	Ka int `yaml:"ka"`
	Kp int `yaml:"kp"`

	MinShuffleTimerDurationSeconds int `yaml:"minShuffleTimerDurationSeconds"`
}

func main() {
	flag.Parse()
	conf := readConfFile()

	protoManagerConf := babel.Config{
		Silent:    false,
		LogFolder: conf.LogFolder,
		SmConf:    babel.StreamManagerConf{DialTimeout: time.Millisecond * time.Duration(conf.dialTimeoutMiliseconds)},
		Peer:      peer.NewPeer(net.ParseIP(conf.SelfPeer.Host), uint16(conf.SelfPeer.Port), 0),
	}

	p := babel.NewProtoManager(protoManagerConf)
	p.RegisterListenAddr(&net.TCPAddr{IP: protoManagerConf.Peer.IP(), Port: int(protoManagerConf.Peer.ProtosPort())})
	p.RegisterListenAddr(&net.UDPAddr{IP: protoManagerConf.Peer.IP(), Port: int(protoManagerConf.Peer.ProtosPort())})
	p.RegisterProtocol(NewHyparviewProtocol(p, &conf))
	p.StartSync()
}

func readConfFile() HyparviewConfig {
	configFileName := "exampleConfig.yml"
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

	var cfg HyparviewConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}
