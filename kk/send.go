package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func deriveKeys(masterKey []byte) ([]byte, []byte) {
	hmacE := hmac.New(sha256.New, masterKey)
	hmacE.Write([]byte("knockknock-encrypt"))
	keyE := hmacE.Sum(nil)

	hmacH := hmac.New(sha256.New, masterKey)
	hmacH.Write([]byte("knockknock-hmac"))
	keyH := hmacH.Sum(nil)

	return keyE, keyH
}

func sendCmd(serverIPStr, key string) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		fmt.Println("Invalid base64 for key:", err)
		os.Exit(1)
	}

	keyE, keyH := deriveKeys(keyBytes)

	spaPacket, err := createPacket(keyE, keyH, serverIPStr)
	if err != nil {
		fmt.Println("Error creating SPA packet:", err)
		os.Exit(1)
	}

	// --- Raw Packet Sending Logic ---

	serverIP := net.ParseIP(serverIPStr)
	if serverIP == nil {
		fmt.Println("Invalid server IP address")
		os.Exit(1)
	}

	// We need a source IP. We can get it by pretending to dial the server.
	srcIP, err := findSourceAddress(serverIPStr)
	if err != nil {
		fmt.Printf("Could not find source IP: %v\n", err)
		os.Exit(1)
	}

	// Construct the packet layers
	ipLayer := &layers.IPv4{
		SrcIP:    srcIP,
		DstIP:    serverIP,
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
	}
	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(12345), // Source port is random
		DstPort: layers.TCPPort(80),      // Destination port can be anything
		SYN:     true,
		Seq:     1105024978, // Needs to be random
		Window:  14600,
	}
	tcpLayer.SetNetworkLayerForChecksum(ipLayer)

	// Embed the SPA data in TCP options
	tcpOptions := layers.TCPOption{
		OptionType:   layers.TCPOptionKindTimestamps,
		OptionLength: 10, // 2 for kind/length + 8 for timestamps
		OptionData:   append(make([]byte, 8), spaPacket...),
	}
	tcpLayer.Options = append(tcpLayer.Options, tcpOptions)

	// Serialize the packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, ipLayer, tcpLayer); err != nil {
		fmt.Printf("Failed to serialize packet: %v\n", err)
		os.Exit(1)
	}

	// Send the packet
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		fmt.Printf("Failed to create raw socket: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Close(fd)

	addr := syscall.SockaddrInet4{
		Port: 0,
	}
	copy(addr.Addr[:], serverIP.To4())

	if err := syscall.Sendto(fd, buf.Bytes(), 0, &addr); err != nil {
		fmt.Printf("Sendto failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Knock sent successfully to", serverIPStr)
}

// findSourceAddress finds the local IP address that would be used to connect to the given destination.
func findSourceAddress(destination string) (net.IP, error) {
	conn, err := net.Dial("udp", destination+":80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("could not determine local address")
	}

	return localAddr.IP, nil
}
