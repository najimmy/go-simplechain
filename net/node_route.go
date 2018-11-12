package net

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type NodeRoute struct {
	mu           sync.Mutex
	done         chan bool
	node         *Node
	addrMap      map[peer.ID]ma.Multiaddr // TODO  change []ma.Multiaddr
	routingTable *kb.RoutingTable
}

func NewNodeRoute(node *Node) *NodeRoute {
	nodeRoute := &NodeRoute{node: node, addrMap: make(map[peer.ID]ma.Multiaddr)}
	nodeRoute.routingTable =
		kb.NewRoutingTable(20, kb.ConvertPeerID(node.host.ID()), time.Minute, node.host.Peerstore())
	return nodeRoute
}

func (nodeRoute *NodeRoute) Update(peerid peer.ID, addr ma.Multiaddr) {
	nodeRoute.mu.Lock()
	nodeRoute.routingTable.Update(peerid)
	nodeRoute.addrMap[peerid] = addr
	nodeRoute.mu.Unlock()

	log.WithFields(log.Fields{
		"ID":   peerid,
		"addr": addr,
	}).Info("Update")
}

func (nodeRoute *NodeRoute) Remove(peerid peer.ID) {
	nodeRoute.mu.Lock()
	nodeRoute.routingTable.Remove(peerid)
	delete(nodeRoute.addrMap, peerid)
	nodeRoute.mu.Unlock()
}

func (nodeRoute *NodeRoute) NearestPeers(peerid peer.ID, count int) map[peer.ID]ma.Multiaddr {
	addrMap := make(map[peer.ID]ma.Multiaddr)
	nodeRoute.mu.Lock()
	peers := nodeRoute.routingTable.NearestPeers(kb.ConvertPeerID(peerid), count)
	for i, p := range peers {
		addrMap[p] = nodeRoute.addrMap[peers[i]]
	}
	nodeRoute.mu.Unlock()
	log.WithFields(log.Fields{
		"peers": peers,
	}).Info("NearestPeers")
	return addrMap
}

//AddNodeFromSeedString is
func (nodeRoute *NodeRoute) AddNodeFromSeedString(seed string) {

	ipfsaddr, err := ma.NewMultiaddr(seed)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}
	log.WithFields(log.Fields{
		"ipfsaddr": ipfsaddr,
		"pid":      pid,
		"peerid":   peerid,
	}).Info("peer info")

	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr) //  /ip4/127.0.0.1/tcp/9990
	nodeRoute.Update(peerid, targetAddr)
	nodeRoute.node.host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	nodeRoute.node.seedID = peerid
}

func (nodeRoute *NodeRoute) FindNewNodes() {
	log.Info("FindNewNodes>>>>>")
	node := nodeRoute.node
	peers := nodeRoute.NearestPeers(nodeRoute.node.host.ID(), 20)
	log.Info(peers, "peers")
	for peerid, addr := range peers {
		if peerid == node.host.ID() {
			continue
		}

		v, ok := node.p2pStreamMap.Load(peerid)
		if ok {
			streamState := v.(*P2PStream)
			streamState.mu.Lock()
			if !streamState.isClosed {
				log.Info("reuse stream")
				streamState.SendPeers()
			} else {
				node.p2pStreamMap.Delete(peerid)
				node.host.Peerstore().ClearAddrs(peerid)
				nodeRoute.Remove(streamState.peerID)
				log.Warning("streamState is removed")

				if node.seedID == peerid {
					log.Warning("add seed node")
					nodeRoute.Update(peerid, addr)
				}
			}
			streamState.mu.Unlock()
		} else {
			node.host.Peerstore().AddAddr(peerid, addr, pstore.PermanentAddrTTL)
			streamState, err := NewP2PStream(node, peerid)
			if err != nil {
				node.host.Peerstore().ClearAddrs(peerid)
				nodeRoute.Remove(peerid)
			} else {
				log.Info(streamState.addr, " new")
				node.p2pStreamMap.Store(streamState.peerID, streamState)
				streamState.Start()
				streamState.SendHello()
				streamState.WaitFinshedHandshake()
				streamState.SendPeers()
			}

		}
	}
	log.Info("<<<<<FindNewNodes")
}

func (nodeRoute *NodeRoute) Start() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.WithFields(log.Fields{
				"count": runtime.NumGoroutine(),
			}).Info("NumGoroutine")
			nodeRoute.FindNewNodes()
		}
	}
}
