package main

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"reflect"
	"time"

	"github.com/nm-morais/go-babel/pkg/errors"
	"github.com/nm-morais/go-babel/pkg/logs"
	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	"github.com/nm-morais/go-babel/pkg/protocolManager"
	"github.com/nm-morais/go-babel/pkg/timer"
	"github.com/sirupsen/logrus"
)

const (
	protoID = 1000
	name    = "Hyparview"
)

type HyparviewConfig struct {
	SelfPeer struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"self"`
	BootstrapPeers []struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"bootstrapPeers"`

	DialTimeoutMiliseconds         int    `yaml:"dialTimeoutMiliseconds"`
	LogFolder                      string `yaml:"logFolder"`
	JoinTimeSeconds                int    `yaml:"joinTimeSeconds"`
	ActiveViewSize                 int    `yaml:"activeViewSize"`
	PassiveViewSize                int    `yaml:"passiveViewSize"`
	ARWL                           int    `yaml:"arwl"`
	PRWL                           int    `yaml:"pwrl"`
	Ka                             int    `yaml:"ka"`
	Kp                             int    `yaml:"kp"`
	MinShuffleTimerDurationSeconds int    `yaml:"minShuffleTimerDurationSeconds"`
}
type Hyparview struct {
	babel           protocolManager.ProtocolManager
	lastShuffleMsg  *ShuffleMessage
	timeStart       time.Time
	logger          *logrus.Logger
	conf            *HyparviewConfig
	selfIsBootstrap bool
	bootstrapNodes  []peer.Peer
	*HyparviewState
}

func NewHyparviewProtocol(babel protocolManager.ProtocolManager, conf *HyparviewConfig) protocol.Protocol {
	logger := logs.NewLogger(name)
	selfIsBootstrap := false
	bootstrapNodes := []peer.Peer{}
	for _, p := range conf.BootstrapPeers {
		boostrapNode := peer.NewPeer(net.ParseIP(p.Host), uint16(p.Port), 0)
		bootstrapNodes = append(bootstrapNodes, boostrapNode)
		if peer.PeersEqual(babel.SelfPeer(), boostrapNode) {
			selfIsBootstrap = true
			break
		}
	}
	logger.Infof("Starting with selfPeer:= %+v", babel.SelfPeer())

	return &Hyparview{
		babel:          babel,
		lastShuffleMsg: nil,
		timeStart:      time.Time{},
		logger:         logger,
		conf:           conf,

		bootstrapNodes:  bootstrapNodes,
		selfIsBootstrap: selfIsBootstrap,

		HyparviewState: &HyparviewState{
			activeView:  View{capacity: conf.ActiveViewSize, peers: make(map[string]*PeerState)},
			passiveView: View{capacity: conf.PassiveViewSize, peers: make(map[string]*PeerState)},
		},
	}
}

func (h *Hyparview) ID() protocol.ID {
	return protoID
}

func (h *Hyparview) Name() string {
	return name
}

func (h *Hyparview) Logger() *logrus.Logger {
	return h.logger
}

func (h *Hyparview) Init() {
	h.babel.RegisterTimerHandler(protoID, ShuffleTimerID, h.HandleShuffleTimer)
	h.babel.RegisterTimerHandler(protoID, PromoteTimerID, h.HandlePromoteTimer)

	h.babel.RegisterMessageHandler(protoID, JoinMessage{}, h.HandleJoinMessage)
	h.babel.RegisterMessageHandler(protoID, ForwardJoinMessage{}, h.HandleForwardJoinMessage)
	h.babel.RegisterMessageHandler(protoID, ForwardJoinMessageReply{}, h.HandleForwardJoinMessageReply)
	h.babel.RegisterMessageHandler(protoID, ShuffleMessage{}, h.HandleShuffleMessage)
	h.babel.RegisterMessageHandler(protoID, ShuffleReplyMessage{}, h.HandleShuffleReplyMessage)
	h.babel.RegisterMessageHandler(protoID, NeighbourMessage{}, h.HandleNeighbourMessage)
	h.babel.RegisterMessageHandler(protoID, NeighbourMessageReply{}, h.HandleNeighbourReplyMessage)
	h.babel.RegisterMessageHandler(protoID, DisconnectMessage{}, h.HandleDisconnectMessage)
}

func (h *Hyparview) Start() {
	h.logger.Infof("Starting with confs: %+v", h.conf)
	h.babel.RegisterTimer(h.ID(), ShuffleTimer{duration: 3 * time.Second})
	h.babel.RegisterPeriodicTimer(h.ID(), PromoteTimer{duration: 7 * time.Second})
	h.joinOverlay()
}

func (h *Hyparview) joinOverlay() {
	if h.selfIsBootstrap {
		return
	}
	toSend := JoinMessage{}
	h.logger.Info("Joining overlay...")
	if len(h.bootstrapNodes) == 0 {
		h.logger.Panic("No nodes to join overlay...")
	}
	bootstrapNode := h.bootstrapNodes[getRandInt(len(h.bootstrapNodes))]
	h.babel.SendMessageSideStream(toSend, bootstrapNode, bootstrapNode.ToTCPAddr(), protoID, protoID)
}

func (h *Hyparview) InConnRequested(dialerProto protocol.ID, p peer.Peer) bool {
	if dialerProto != h.ID() {
		h.logger.Warnf("Denying connection  from peer %+v", p)
		return false
	}

	return true
}

func (h *Hyparview) OutConnDown(p peer.Peer) {
	h.handleNodeDown(p)
	h.logger.Errorf("Peer %s out connection went down", p.String())
}

func (h *Hyparview) DialFailed(p peer.Peer) {
	h.logger.Errorf("Failed to dial peer %s", p.String())
	h.handleNodeDown(p)
}

func (h *Hyparview) handleNodeDown(p peer.Peer) {
	defer h.logHyparviewState()
	if h.activeView.remove(p) {
		if !h.activeView.isFull() {
			if h.passiveView.size() == 0 {
				if h.activeView.size() == 0 {
					h.joinOverlay()
				}
				return
			}
			newNeighbor := h.passiveView.getRandomElementsFromView(1)
			h.logger.Warnf("replacing downed with node %s from passive view", newNeighbor[0].String())
			h.sendMessageTmpTransport(NeighbourMessage{
				HighPrio: h.activeView.size() <= 1, // TODO review this
			}, newNeighbor[0])
		}
	}
}

func (h *Hyparview) DialSuccess(sourceProto protocol.ID, p peer.Peer) bool {
	if sourceProto != h.ID() {
		return false
	}

	if h.activeView.contains(p) {
		h.activeView.peers[p.String()].outConnected = true
		h.logger.Info("Dialed node in active view")
		return true
	}
	h.logger.Warnf("Disconnecting connection from peer %+v because it is not in active view", p)
	h.babel.SendMessageSideStream(DisconnectMessage{}, p, p.ToTCPAddr(), h.ID(), h.ID())
	return false
}

func (h *Hyparview) MessageDelivered(msg message.Message, p peer.Peer) {
	h.logger.Infof("Message of type [%s] body: %+v was sent to %s", reflect.TypeOf(msg), msg, p.String())
	if msg.Type() == DisconnectMessageType {
		h.babel.Disconnect(h.ID(), p)
		h.logger.Infof("Disconnecting from %s", p.String())
	}
}

func (h *Hyparview) MessageDeliveryErr(msg message.Message, p peer.Peer, err errors.Error) {
	h.logger.Warnf("Message %s was not sent to %s because: %s", reflect.TypeOf(msg), p.String(), err.Reason())
	_, isNeighMsg := msg.(NeighbourMessage)
	if isNeighMsg {
		h.passiveView.remove(p)
	}
}

// ---------------- Protocol handlers (messages) ----------------

func (h *Hyparview) HandleJoinMessage(sender peer.Peer, msg message.Message) {
	h.logger.Infof("Received join message from %s", sender)
	if h.activeView.isFull() {
		h.dropRandomElemFromActiveView()
	}
	toSend := ForwardJoinMessage{
		TTL:            uint32(h.conf.ARWL),
		OriginalSender: sender,
	}
	h.addPeerToActiveView(sender)
	h.sendMessageTmpTransport(ForwardJoinMessageReply{}, sender)
	for _, neigh := range h.activeView.peers {
		if peer.PeersEqual(neigh, sender) {
			continue
		}

		if neigh.outConnected {
			h.logger.Infof("Sending ForwardJoin (original=%s) message to: %s", sender.String(), neigh.String())
			h.sendMessage(toSend, neigh)
		}
	}
}

func (h *Hyparview) HandleForwardJoinMessage(sender peer.Peer, msg message.Message) {
	fwdJoinMsg := msg.(ForwardJoinMessage)
	h.logger.Infof("Received forward join message with ttl = %d, originalSender=%s from %s",
		fwdJoinMsg.TTL,
		fwdJoinMsg.OriginalSender.String(),
		sender.String())

	if fwdJoinMsg.OriginalSender == h.babel.SelfPeer() {
		h.logger.Panic("Received forward join message sent by myself")
	}

	if fwdJoinMsg.TTL == 0 || h.activeView.size() == 1 {
		if fwdJoinMsg.TTL == 0 {
			h.logger.Infof("Accepting forwardJoin message from %s, fwdJoinMsg.TTL == 0", fwdJoinMsg.OriginalSender.String())
		}
		if h.activeView.size() == 1 {
			h.logger.Infof("Accepting forwardJoin message from %s, h.activeView.size() == 1", fwdJoinMsg.OriginalSender.String())
		}
		if h.addPeerToActiveView(fwdJoinMsg.OriginalSender) {
			h.sendMessageTmpTransport(ForwardJoinMessageReply{}, fwdJoinMsg.OriginalSender)
		}
		return
	}

	if fwdJoinMsg.TTL == uint32(h.conf.PRWL) {
		h.addPeerToPassiveView(fwdJoinMsg.OriginalSender)
	}

	rndSample := h.activeView.getRandomElementsFromView(1, fwdJoinMsg.OriginalSender, sender)
	if len(rndSample) == 0 { // only know original sender, act as if join message
		h.logger.Errorf("Cannot forward forwardJoin message, dialing %s", fwdJoinMsg.OriginalSender.String())
		if h.addPeerToActiveView(fwdJoinMsg.OriginalSender) {
			h.sendMessageTmpTransport(ForwardJoinMessageReply{}, fwdJoinMsg.OriginalSender)
		}
		return
	}

	toSend := ForwardJoinMessage{
		TTL:            fwdJoinMsg.TTL - 1,
		OriginalSender: fwdJoinMsg.OriginalSender,
	}
	nodeToSendTo := rndSample[0]
	h.logger.Infof(
		"Forwarding forwardJoin (original=%s) with TTL=%d message to : %s",
		fwdJoinMsg.OriginalSender.String(),
		toSend.TTL,
		nodeToSendTo.String(),
	)
	h.sendMessage(toSend, nodeToSendTo)
}

func (h *Hyparview) HandleForwardJoinMessageReply(sender peer.Peer, msg message.Message) {
	h.logger.Infof("Received forward join from message %s", sender.String())
	h.addPeerToActiveView(sender)
}

func (h *Hyparview) HandleNeighbourMessage(sender peer.Peer, msg message.Message) {
	neighborMsg := msg.(NeighbourMessage)
	h.logger.Infof("Received neighbor message %+v", neighborMsg)

	if neighborMsg.HighPrio {
		if h.addPeerToActiveView(sender) {
			reply := NeighbourMessageReply{
				Accepted: true,
			}
			h.sendMessageTmpTransport(reply, sender)
		}
		return
	}

	if h.activeView.isFull() {
		reply := NeighbourMessageReply{
			Accepted: false,
		}
		h.sendMessageTmpTransport(reply, sender)
		return
	}
	if h.addPeerToActiveView(sender) {
		reply := NeighbourMessageReply{
			Accepted: true,
		}
		h.sendMessageTmpTransport(reply, sender)
	}
}

func (h *Hyparview) HandleNeighbourReplyMessage(sender peer.Peer, msg message.Message) {
	h.logger.Info("Received neighbor reply message")
	neighborReplyMsg := msg.(NeighbourMessageReply)
	if neighborReplyMsg.Accepted {
		h.addPeerToActiveView(sender)
	}
}

func (h *Hyparview) HandleShuffleMessage(sender peer.Peer, msg message.Message) {
	shuffleMsg := msg.(ShuffleMessage)
	if shuffleMsg.TTL > 0 {
		rndSample := h.activeView.getRandomElementsFromView(1, sender)
		if len(rndSample) != 0 {
			toSend := ShuffleMessage{
				ID:    shuffleMsg.ID,
				TTL:   shuffleMsg.TTL - 1,
				Peers: shuffleMsg.Peers,
			}
			h.logger.Debug("Forwarding shuffle message to :", rndSample[0].String())
			h.sendMessage(toSend, rndSample[0])
			return
		}
	}

	//  TTL is 0 or have no nodes to forward to
	//  select random nr of hosts from passive view
	exclusions := append(shuffleMsg.Peers, h.babel.SelfPeer(), sender)
	toSend := h.passiveView.getRandomElementsFromView(len(shuffleMsg.Peers), exclusions...)
	h.mergeShuffleMsgPeersWithPassiveView(shuffleMsg.Peers, toSend)
	reply := ShuffleReplyMessage{
		ID:    shuffleMsg.ID,
		Peers: toSend,
	}
	h.sendMessageTmpTransport(reply, sender)
}

func (h *Hyparview) mergeShuffleMsgPeersWithPassiveView(shuffleMsgPeers, peersToKickFirst []peer.Peer) {
	for _, receivedHost := range shuffleMsgPeers {
		if h.babel.SelfPeer().String() == receivedHost.String() {
			continue
		}

		if h.activeView.contains(receivedHost) || h.passiveView.contains(receivedHost) {
			continue
		}

		if h.passiveView.isFull() { // if passive view is not full, skip check and add directly
			removed := false
			for _, firstToKick := range peersToKickFirst {
				if h.passiveView.remove(firstToKick) {
					removed = true
					break
				}
			}
			if !removed {
				h.passiveView.dropRandom() // drop random element to make space
			}
		}
		h.addPeerToPassiveView(receivedHost)
	}
}

func (h *Hyparview) HandleShuffleReplyMessage(sender peer.Peer, m message.Message) {
	shuffleReplyMsg := m.(ShuffleReplyMessage)
	h.logger.Infof("Received shuffle reply message %+v", shuffleReplyMsg)
	peersToDiscardFirst := []peer.Peer{}
	if h.lastShuffleMsg != nil {
		peersToDiscardFirst = append(peersToDiscardFirst, h.lastShuffleMsg.Peers...)
	}
	h.lastShuffleMsg = nil
	h.mergeShuffleMsgPeersWithPassiveView(shuffleReplyMsg.Peers, peersToDiscardFirst)
}

// ---------------- Protocol handlers (timers) ----------------

func (h *Hyparview) HandlePromoteTimer(t timer.Timer) {
	h.logger.Info("Promote timer trigger")
	if time.Since(h.timeStart) > time.Duration(h.conf.JoinTimeSeconds)*time.Second {
		if h.activeView.size() == 0 && h.passiveView.size() == 0 {
			h.joinOverlay()
			return
		}
		if !h.activeView.isFull() && h.passiveView.size() > 0 {
			h.logger.Warn("Promoting node from passive view to active view")
			newNeighbor := h.passiveView.getRandomElementsFromView(1)
			h.sendMessageTmpTransport(NeighbourMessage{
				HighPrio: h.activeView.size() <= 1, // TODO review this
			}, newNeighbor[0])
		}
	}
}

func (h *Hyparview) HandleShuffleTimer(t timer.Timer) {
	h.logger.Info("Shuffle timer trigger")
	minShuffleDuration := time.Duration(h.conf.MinShuffleTimerDurationSeconds) * time.Second

	// add jitter to emission of shuffle messages
	toWait := minShuffleDuration + time.Duration(float32(minShuffleDuration)*rand.Float32())
	h.babel.RegisterTimer(h.ID(), ShuffleTimer{duration: toWait})

	if h.activeView.size() == 0 {
		h.logger.Info("No nodes to send shuffle message message to")
		return
	}

	passiveViewRandomPeers := h.passiveView.getRandomElementsFromView(h.conf.Kp - 1)
	activeViewRandomPeers := h.activeView.getRandomElementsFromView(h.conf.Ka)
	peers := append(passiveViewRandomPeers, activeViewRandomPeers...)
	peers = append(peers, h.babel.SelfPeer())
	toSend := ShuffleMessage{
		ID:    uint32(getRandInt(math.MaxUint32)),
		TTL:   uint32(h.conf.PRWL),
		Peers: peers,
	}
	h.lastShuffleMsg = &toSend
	rndNode := h.activeView.getRandomElementsFromView(1)
	h.logger.Info("Sending shuffle message to: ", rndNode[0].String())
	h.sendMessage(toSend, rndNode[0])
}

func (h *Hyparview) HandleDisconnectMessage(sender peer.Peer, m message.Message) {
	h.logger.Warn("Got Disconnect message")
	iPeer := sender
	h.activeView.remove(iPeer)
	h.addPeerToPassiveView(iPeer)
}

// ---------------- Auxiliary functions ----------------

func (h *Hyparview) logHyparviewState() {
	h.logger.Info("------------- Hyparview state -------------")
	var toLog string
	toLog = "Active view : "
	for _, p := range h.activeView.peers {
		toLog += fmt.Sprintf("%s, ", p.String())
	}
	h.logger.Info(toLog)
	toLog = "Passive view : "
	for _, p := range h.passiveView.peers {
		toLog += fmt.Sprintf("%s, ", p.String())
	}
	h.logger.Info(toLog)
	h.logger.Info("-------------------------------------------")
}

func (h *Hyparview) sendMessage(msg message.Message, target peer.Peer) {
	h.babel.SendMessage(msg, target, h.ID(), h.ID(), false)
}

func (h *Hyparview) sendMessageTmpTransport(msg message.Message, target peer.Peer) {
	h.babel.SendMessageSideStream(msg, target, target.ToTCPAddr(), h.ID(), h.ID())
}
