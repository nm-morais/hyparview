package main

import (
	"math/rand"

	"github.com/nm-morais/go-babel/pkg/peer"
)

type View struct {
	capacity int
	peers    map[string]*PeerState
}

func (v View) size() int {
	return len(v.peers)
}

func (v View) contains(p peer.Peer) bool {
	_, ok := v.peers[p.String()]
	return ok
}

func (v View) dropRandom() *PeerState {
	toDrop := getRandInt(len(v.peers))
	i := 0
	for k := range v.peers {
		if i == toDrop {
			tmp := v.peers[k]
			delete(v.peers, k)
			return tmp
		}
		i++
	}
	return nil
}

func (v View) add(p *PeerState, dropIfFull bool) (added bool) {

	if v.isFull() {
		if dropIfFull {
			v.dropRandom()
		} else {
			panic("adding peer to view already full")
		}
	}

	_, alreadyExists := v.peers[p.String()]
	if !alreadyExists {
		v.peers[p.String()] = p
	}
	return !alreadyExists
}

func (v View) remove(p peer.Peer) (existed bool) {
	_, existed = v.peers[p.String()]
	if existed {
		delete(v.peers, p.String())
	}
	return existed
}

func (v View) isFull() bool {
	return len(v.peers) == v.capacity
}

func (v View) toArray() []*PeerState {
	peerArr := make([]*PeerState, 0, len(v.peers))
	for _, p := range v.peers {
		peerArr = append(peerArr, p)
	}
	return peerArr
}

func (v View) getRandomElementsFromView(amount int, exclusions ...peer.Peer) []peer.Peer {
	viewAsArr := v.toArray()
	perm := rand.Perm(len(viewAsArr))
	rndElements := []peer.Peer{}
	for i := 0; i < len(viewAsArr) && len(rndElements) < amount; i++ {
		excluded := false
		curr := viewAsArr[perm[i]]
		for _, exclusion := range exclusions {
			if peer.PeersEqual(exclusion, curr) {
				excluded = true
				break
			}
		}
		if !excluded {
			rndElements = append(rndElements, curr)
		}
	}
	return rndElements
}

type PeerState struct {
	peer.Peer
	outConnected bool
	inConnected  bool
}

type HyparviewState struct {
	activeView  View
	passiveView View
}

func NewHyparviewState() *HyparviewState {
	return &HyparviewState{
		activeView:  View{},
		passiveView: View{},
	}
}

func (h *Hyparview) addPeerToActiveView(newPeer peer.Peer) {
	if peer.PeersEqual(h.babel.SelfPeer(), newPeer) {
		h.logger.Panic("Trying to add self to active view")
	}

	if h.activeView.isFull() {
		h.logger.Panic("Cannot add node to active pool because it is full")
	}

	if h.activeView.contains(newPeer) {
		h.logger.Panic("Trying to add node already in view")
	}

	if h.passiveView.contains(newPeer) {
		h.logger.Panic("Trying to add node to active view already in passive view")
	}

	h.logger.Warnf("Added peer %s to active view", newPeer.String())
	h.activeView.add(&PeerState{
		Peer:         newPeer,
		outConnected: false,
		inConnected:  false,
	}, false)
	h.logger.Infof("dialing new node %s", newPeer.String())
	h.babel.Dial(h.ID(), newPeer, newPeer.ToTCPAddr())
	h.logHyparviewState()
}

func (h *Hyparview) addPeerToPassiveView(newPeer peer.Peer) {
	if peer.PeersEqual(newPeer, h.babel.SelfPeer()) {
		h.logger.Panic("trying to add self to passive view ")
	}

	if h.activeView.contains(newPeer) {
		h.logger.Panic("Trying to add node to passive view which is in active view")
	}

	h.passiveView.add(&PeerState{
		Peer:         newPeer,
		outConnected: false,
		inConnected:  false,
	}, true)
	h.logger.Warnf("Added peer %s to passive view", newPeer.String())
	h.logHyparviewState()
}

func (h *Hyparview) dropRandomElemFromActiveView() {
	removed := h.activeView.dropRandom()
	if removed != nil {
		h.addPeerToPassiveView(removed)
		disconnectMsg := DisconnectMessage{}
		h.babel.SendMessageAndDisconnect(disconnectMsg, removed, h.ID(), h.ID())
		h.logHyparviewState()
	}
}
