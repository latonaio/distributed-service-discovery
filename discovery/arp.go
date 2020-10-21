package discovery

import (
	"bytes"
	"log"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func (s *ServiceDiscovery) arp(iface *net.Interface, addr *net.IPNet, addrCh chan *Addr) {
	// Open up a pcap handle for packet reads/writes.
	handle, err := pcap.OpenLive(s.IfaceName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}
	defer handle.Close()

	stop := make(chan struct{})
	go s.readARP(handle, iface, addrCh, stop)
	defer close(stop)

	// Write our scan packets out to the handle.
	if err := s.writeARP(handle, iface, addr); err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	time.Sleep(10 * time.Second)
}

func (s *ServiceDiscovery) readARP(handle *pcap.Handle, iface *net.Interface, addrCh chan *Addr, stop chan struct{}) {
	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	in := src.Packets()

	for {
		var packet gopacket.Packet
		select {
		case <-stop:
			return
		case packet = <-in:
			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)
			if arp.Operation != layers.ARPReply || bytes.Equal([]byte(iface.HardwareAddr), arp.SourceHwAddress) {
				// This is a packet I sent.
				continue
			}
			addrCh <- &Addr{
				IP:           net.IP(arp.SourceProtAddress),
				HardwareAddr: net.HardwareAddr(arp.SourceHwAddress),
			}
		}
	}
}

func (s *ServiceDiscovery) writeARP(handle *pcap.Handle, iface *net.Interface, addr *net.IPNet) error {
	// Set up all the layers' fields we can.
	eth := layers.Ethernet{
		SrcMAC:       iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(iface.HardwareAddr),
		SourceProtAddress: []byte(addr.IP),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
	}
	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	for _, ip := range ips(addr) {
		arp.DstProtAddress = []byte(ip)
		gopacket.SerializeLayers(buf, opts, &eth, &arp)
		if err := handle.WritePacketData(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}
