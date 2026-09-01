package mesh

import (
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"modelmesh/pkg/log"
)

type GateProvider interface {
	Has(s string) bool
}

type GateKeeper struct {
	provider GateProvider
}

// --- relay.ACLFilter ---
func (l *GateKeeper) AllowReserve(p peer.ID, _ ma.Multiaddr) bool {
	allow := l.provider.Has(p.String())
	if !allow {
		log.WithName("gate").Warnf("reserve attempt from %s denied", p)
	}
	return allow
}
func (l *GateKeeper) AllowConnect(src peer.ID, _ ma.Multiaddr, dest peer.ID) bool {
	allow := l.provider.Has(src.String()) && l.provider.Has(dest.String())
	if !allow {
		log.WithName("gate").Warnf("connection attempt from %s to %s denied", src, dest)
	}
	return allow
}

// --- connmgr.ConnectionGater ---

func (l *GateKeeper) InterceptPeerDial(p peer.ID) bool             { return l.provider.Has(p.String()) }
func (l *GateKeeper) InterceptAddrDial(peer.ID, ma.Multiaddr) bool { return true }
func (l *GateKeeper) InterceptAccept(network.ConnMultiaddrs) bool  { return true }
func (l *GateKeeper) InterceptSecured(_ network.Direction, p peer.ID, _ network.ConnMultiaddrs) bool {
	return l.provider.Has(p.String())
}
func (l *GateKeeper) InterceptUpgraded(network.Conn) (bool, control.DisconnectReason) {
	return true, 0
}

func NewGateKeeper(p GateProvider) *GateKeeper {
	return &GateKeeper{provider: p}
}
