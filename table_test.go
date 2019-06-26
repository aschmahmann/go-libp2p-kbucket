package kbucket

import (
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/test"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	mh "github.com/multiformats/go-multihash"
)

// Test basic features of the bucket struct
func TestBucket(t *testing.T) {
	b := newBucket()

	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = test.RandPeerIDFatal(t)
		b.PushFront(peers[i])
	}

	local := test.RandPeerIDFatal(t)
	localID := ConvertPeerID(local)

	i := rand.Intn(len(peers))
	if !b.Has(peers[i]) {
		t.Errorf("Failed to find peer: %v", peers[i])
	}

	spl := b.Split(0, ConvertPeerID(local))
	llist := b.list
	for e := llist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl > 0 {
			t.Fatalf("Split failed. found id with cpl > 0 in 0 bucket")
		}
	}

	rlist := spl.list
	for e := rlist.Front(); e != nil; e = e.Next() {
		p := ConvertPeerID(e.Value.(peer.ID))
		cpl := CommonPrefixLen(p, localID)
		if cpl == 0 {
			t.Fatalf("Split failed. found id with cpl == 0 in non 0 bucket")
		}
	}
}

func RandPeerIDFatal(t testing.TB, r *rand.Rand) peer.ID {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(r, buf); err != nil {
		t.Fatal(err)
	}
	h, _ :=  mh.Sum(buf, mh.SHA2_256, -1)
	return peer.ID(h)
}

func TestTableCallbacks(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	local := RandPeerIDFatal(t, rng)
	m := pstore.NewMetrics()

	t.Run("KRoutingTable", func(t *testing.T) {
		TableCallbacks(t, RoutingTable{NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	t.Run("TrieRoutingTable", func(t *testing.T) {
		TableCallbacks(t, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

func TableCallbacks(t *testing.T, rt RoutingTable, rng *rand.Rand) {
	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = RandPeerIDFatal(t, rng)
	}

	pset := make(map[peer.ID]struct{})
	rt.SetPeerAdded(func(p peer.ID) {
		pset[p] = struct{}{}
	})
	rt.SetPeerRemoved(func(p peer.ID) {
		delete(pset, p)
	})

	rt.Update(peers[0])
	if _, ok := pset[peers[0]]; !ok {
		t.Fatal("should have this peer")
	}

	rt.Remove(peers[0])
	if _, ok := pset[peers[0]]; ok {
		t.Fatal("should not have this peer")
	}

	for _, p := range peers {
		rt.Update(p)
	}

	out := rt.ListPeers()
	for _, outp := range out {
		if _, ok := pset[outp]; !ok {
			t.Fatal("should have peer in the peerset")
		}
		delete(pset, outp)
	}

	if len(pset) > 0 {
		t.Fatal("have peers in peerset that were not in the table", len(pset))
	}
}

func TestTableUpdate(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	t.Run("KRoutingTable", func(t *testing.T) {
		TableUpdate(t, RoutingTable{NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	t.Run("TrieRoutingTable", func(t *testing.T) {
		TableUpdate(t, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

// Right now, this just makes sure that it doesnt hang or crash
func TableUpdate(t *testing.T, rt RoutingTable, rng *rand.Rand) {
	peers := make([]peer.ID, 100)
	for i := 0; i < 100; i++ {
		peers[i] = RandPeerIDFatal(t, rng)
	}

	// Testing Update
	for i := 0; i < 10000; i++ {
		rt.Update(peers[rand.Intn(len(peers))])
	}

	for i := 0; i < 100; i++ {
		id := ConvertPeerID(RandPeerIDFatal(t, rng))
		ret := rt.NearestPeers(id, 5)
		if len(ret) == 0 {
			t.Fatal("Failed to find node near ID.")
		}
	}
}

func TestTableFind(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	t.Run("KRoutingTable", func(t *testing.T) {
		TableFind(t, RoutingTable{NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	t.Run("TrieRoutingTable", func(t *testing.T) {
		TableFind(t, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

func TableFind(t *testing.T,  rt RoutingTable, rng *rand.Rand) {
	peers := make([]peer.ID, 100)
	for i := 0; i < 5; i++ {
		peers[i] = RandPeerIDFatal(t, rng)
		rt.Update(peers[i])
	}

	t.Logf("Searching for peer: '%s'", peers[2])
	found := rt.NearestPeer(ConvertPeerID(peers[2]))
	if !(found == peers[2]) {
		t.Fatalf("Failed to lookup known node...")
	}
}

func TestTableEldestPreferred(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rt := NewRoutingTable(10, ConvertPeerID(local), time.Hour, m)
	rng := rand.New(rand.NewSource(0))

	// generate size + 1 peers to saturate a bucket
	peers := make([]peer.ID, 15)
	for i := 0; i < 15; {
		if p := RandPeerIDFatal(t, rng); CommonPrefixLen(ConvertPeerID(local), ConvertPeerID(p)) == 0 {
			peers[i] = p
			i++
		}
	}

	// test 10 first peers are accepted.
	for _, p := range peers[:10] {
		if _, err := rt.Update(p); err != nil {
			t.Errorf("expected all 10 peers to be accepted; instead got: %v", err)
		}
	}

	// test next 5 peers are rejected.
	for _, p := range peers[10:] {
		if _, err := rt.Update(p); err != ErrPeerRejectedNoCapacity {
			t.Errorf("expected extra 5 peers to be rejected; instead got: %v", err)
		}
	}
}

func TestTableFindMultiple(t *testing.T) {
	local := test.RandPeerIDFatal(t)
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	t.Run("KRoutingTable", func(t *testing.T) {
		TableFindMultiple(t, RoutingTable{NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	t.Run("TrieRoutingTable", func(t *testing.T) {
		TableFindMultiple(t, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

func TableFindMultiple(t *testing.T, rt RoutingTable, rng *rand.Rand) {
	peers := make([]peer.ID, 100)
	for i := 0; i < 18; i++ {
		peers[i] = RandPeerIDFatal(t, rng)
		rt.Update(peers[i])
	}

	t.Logf("Searching for peer: '%s'", peers[2])
	found := rt.NearestPeers(ConvertPeerID(peers[2]), 15)
	if len(found) != 15 {
		t.Fatalf("Got back different number of peers than we expected.")
	}
}

func TestTableMultithreaded(t *testing.T) {
	local := peer.ID("localPeer")
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	t.Run("KRoutingTable", func(t *testing.T) {
		TableMultithreaded(t, RoutingTable{NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	t.Run("TrieRoutingTable", func(t *testing.T) {
		TableMultithreaded(t, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

// Looks for race conditions in table operations. For a more 'certain'
// test, increase the loop counter from 1000 to a much higher number
// and set GOMAXPROCS above 1
func TableMultithreaded(t *testing.T, tab RoutingTable, rng *rand.Rand) {
	var peers []peer.ID
	for i := 0; i < 500; i++ {
		peers = append(peers, RandPeerIDFatal(t, rng))
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Update(peers[n])
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			n := rand.Intn(len(peers))
			tab.Find(peers[n])
		}
		done <- struct{}{}
	}()
	<-done
	<-done
	<-done
}

func BenchmarkUpdates(b *testing.B) {
	local := peer.ID("localPeer")
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	b.Run("KRoutingTable", func(b *testing.B) {
		Updates(b, RoutingTable{NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	b.Run("TrieRoutingTable", func(b *testing.B) {
		Updates(b, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

func Updates(b *testing.B, rt RoutingTable, rng *rand.Rand) {
	b.StopTimer()
	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, RandPeerIDFatal(b, rng))
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		rt.Update(peers[i])
	}
}

func BenchmarkFinds(b *testing.B) {
	local := peer.ID("localPeer")
	m := pstore.NewMetrics()
	rng := rand.New(rand.NewSource(0))

	b.Run("KRoutingTable", func(b *testing.B) {
		Finds(b, RoutingTable{NewRoutingTable(20, ConvertPeerID(local), time.Hour, m)}, rng)
	})
	b.Run("TrieRoutingTable", func(b *testing.B) {
		Finds(b, RoutingTable{NewTrieRoutingTable(ConvertPeerID(local), time.Hour, m)}, rng)
	})
}

func Finds(b *testing.B, rt RoutingTable, rng *rand.Rand) {
	b.StopTimer()

	var peers []peer.ID
	for i := 0; i < b.N; i++ {
		peers = append(peers, RandPeerIDFatal(b, rng))
		rt.Update(peers[i])
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		rt.Find(peers[i])
	}
}