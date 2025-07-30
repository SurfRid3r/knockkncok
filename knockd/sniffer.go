
package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Sniffer listens for packets on a given interface.

type Sniffer struct {
	handle *pcap.Handle
}

// NewSniffer creates a new sniffer for the given interface.
func NewSniffer(iface string) (*Sniffer, error) {
	handle, err := pcap.OpenLive(iface, 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	return &Sniffer{handle: handle}, nil
}

// C returns a channel of packets.
func (s *Sniffer) C() <-chan gopacket.Packet {
	packetSource := gopacket.NewPacketSource(s.handle, s.handle.LinkType())
	return packetSource.Packets()
}

// Close closes the sniffer.
func (s *Sniffer) Close() {
	s.handle.Close()
}
