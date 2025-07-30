package main

import (
	"fmt"
	"net"
)

// autoSelectInterface tries to find the best non-loopback interface for sniffing.
// It does this by finding the interface associated with the default route.
func autoSelectInterface() (string, error) {
	// The "default" route can be found by dialing a public IP.
	// The local address used for the connection will belong to the interface we want.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("could not determine default route: %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("could not determine local address")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.Equal(localAddr.IP) {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable interface found")
}
