
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
)

func getAgentID() (uint64, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}

	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && i.HardwareAddr != nil {
			hash := sha256.Sum256(i.HardwareAddr)
			return binary.BigEndian.Uint64(hash[:8]), nil
		}
	}

	return 0, fmt.Errorf("no suitable network interface found for agent ID generation")
}
