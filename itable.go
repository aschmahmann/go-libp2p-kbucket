package kbucket

import (
"github.com/libp2p/go-libp2p-core/peer"

)

// RoutingTable defines the routing table.
type IRoutingTable interface {
	Update(p peer.ID) (evicted peer.ID, err error)
	Remove(p peer.ID)
	Find(id peer.ID) peer.ID
	NearestPeer(id ID) peer.ID
	NearestPeers(id ID, count int) []peer.ID
	Size() int
	ListPeers() []peer.ID
	Print()
	SetPeerAddedCB(func(peer.ID))
	SetPeerRemovedCB(func(peer.ID))
}

type RoutingTable struct {
	Rt IRoutingTable
}

func (rt *RoutingTable) Update(p peer.ID) (evicted peer.ID, err error) {return rt.Rt.Update(p)}
func (rt *RoutingTable) Remove(p peer.ID)                              {rt.Rt.Remove(p)}
func (rt *RoutingTable) Find(id peer.ID) peer.ID                       {return rt.Rt.Find(id)}
func (rt *RoutingTable) NearestPeer(id ID) peer.ID                     {return rt.Rt.NearestPeer(id)}
func (rt *RoutingTable) NearestPeers(id ID, count int) []peer.ID       {return rt.Rt.NearestPeers(id, count)}
func (rt *RoutingTable) Size() int                                     {return rt.Rt.Size()}
func (rt *RoutingTable) ListPeers() []peer.ID                          {return rt.Rt.ListPeers()}
func (rt *RoutingTable) Print()                                        {rt.Rt.Print()}
func (rt *RoutingTable) SetPeerAddedCB(fn func(peer.ID))               {rt.Rt.SetPeerAddedCB(fn)}
func (rt *RoutingTable) SetPeerRemovedCB(fn func(peer.ID))             {rt.Rt.SetPeerRemovedCB(fn)}

