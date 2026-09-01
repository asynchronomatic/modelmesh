package mesh

import (
	"net"

	libp2pnet "github.com/libp2p/go-libp2p/core/network"
)

type streamConn struct {
	libp2pnet.Stream
}

func (c streamConn) LocalAddr() net.Addr {
	return addr{"libp2p", c.Stream.Conn().LocalPeer().String()}
}

func (c streamConn) RemoteAddr() net.Addr {
	return addr{"libp2p", c.Stream.Conn().RemotePeer().String()}
}

type addr struct{ net, s string }

func (a addr) Network() string { return a.net }
func (a addr) String() string  { return a.s }
