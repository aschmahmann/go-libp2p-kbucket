package kbucket

import (
	"github.com/k-sone/critbitgo"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"sync"
	"time"
)

type TrieRoutingTable struct {
	trie critbitgo.Trie

	// ID of the local peer
	local ID

	// Blanket lock, refine later for better performance
	tabLock sync.RWMutex

	// latency metrics
	metrics peerstore.Metrics

	// Maximum acceptable latency for peers in this cluster
	maxLatency time.Duration

	// kBuckets define all the fingers to other nodes.
	Buckets    []*Bucket
	bucketsize int

	// notification functions
	PeerRemoved func(peer.ID)
	PeerAdded   func(peer.ID)
}

func NewTrieRoutingTable(localID ID, latency time.Duration, m peerstore.Metrics) *TrieRoutingTable {
	return &TrieRoutingTable{
		local: localID,
		maxLatency: latency,
		metrics: m,
		PeerAdded: func(id peer.ID) {},
		PeerRemoved: func(id peer.ID) {},
	}
}

func (rt *TrieRoutingTable) Update(p peer.ID) (evicted peer.ID, err error) {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	if rt.metrics.LatencyEWMA(p) > rt.maxLatency {
		// Connection doesnt meet requirements, skip!
		return "", ErrPeerRejectedHighLatency
	}

	rt.trie.Insert(ConvertPeerID(p), p)
	rt.PeerAdded(p)
	return "", nil
}

func (rt *TrieRoutingTable) Remove(p peer.ID) {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	_, ok := rt.trie.Delete(ConvertPeerID(p))
	if ok {
		rt.PeerRemoved(p)
	}
}

func (rt *TrieRoutingTable) Find(id peer.ID) peer.ID {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	pid := ConvertPeerID(id)
	_, ok := rt.trie.Get(pid)
	if ok {
		return id
	} else {
		return ""
	}
}

func (rt *TrieRoutingTable) NearestPeer(id ID) peer.ID {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	foundID := rt.trie.GetClosest(id)
	if foundID == nil || len(foundID) == 0{
		return ""
	}
	val, _ := rt.trie.Get(foundID)
	return val.(peer.ID)
}

func (rt *TrieRoutingTable) NearestPeers(id ID, count int) []peer.ID {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	afterWalk := 0
	beforeWalk := 0

	peers := make([]peer.ID, 0, count*2)

	sz := rt.trie.Size()
	if sz == 0 {
		return []peer.ID{}
	}

	wid := rt.trie.GetClosest(id)
	//wid, _, _ := rt.trie.LongestPrefix(id)
	//if wid == nil {
	//	wid = rt.trie.GetArbitrary()
	//}

	rt.trie.Walk(wid, func(key []byte, value interface{}) bool {
		peers = append(peers, value.(peer.ID))
		afterWalk++
		return afterWalk < count
	})

	rt.trie.RevWalk(wid, func(key []byte, value interface{}) bool {
		if beforeWalk == 0{
			beforeWalk ++
			return true
		}

		peers = append(peers, value.(peer.ID))
		beforeWalk++
		return beforeWalk < count
	})

	max := len(peers)
	if max > count {
		max = count
	}

	return SortClosestPeers(peers, id)[:max]
}

func (rt *TrieRoutingTable) Size() int {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	return rt.trie.Size()
}

func (rt *TrieRoutingTable) ListPeers() []peer.ID {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	peers := make([]peer.ID, 0, 100)
	k ,_, _:= rt.trie.LongestPrefix([]byte{0,0,0,0,0,0,0,0})
	rt.trie.Walk(k, func(key []byte, value interface{}) bool {
		peers = append(peers, value.(peer.ID))
		return true
	})
	return peers
}
func (rt *TrieRoutingTable) Print() {}

func (rt *TrieRoutingTable) SetPeerAddedCB(fn func(id peer.ID)) {
	rt.PeerAdded = fn
}
func (rt *TrieRoutingTable) SetPeerRemovedCB(fn func(id peer.ID)) {
	rt.PeerRemoved = fn
}