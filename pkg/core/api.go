package core

type PeerConnectionDetails struct {
	PeerID        string
	PeerName      string
	Kind          string
	RemoteAddress string
	LocalAddress  string
	Direction     string
	Security      string
	Multiplexer   string
	StreamCount   int
	Streams       []string
}

type MeshInfo struct {
	AdvertisedAddresses []string
	Connections         []PeerConnectionDetails
}
