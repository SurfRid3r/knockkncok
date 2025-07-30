
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

const (
	protocolVersion = 0x02
	validTimeWindow = 30 // seconds
)

// SPAInfo holds the decoded information from a valid SPA packet.

type SPAInfo struct {
	AgentID uint64
	IP      string
}

// Verify checks if a packet is a valid SPA packet.
func Verify(packetData []byte, keyE, keyH []byte, nonceStore *NonceStore) (*SPAInfo, bool) {
	// Enhanced bounds checking to prevent buffer overflow
	const minPacketSize = 61 // 1(version) + 4(timestamp) + 8(agentID) + 16(nonce) + 16(MAC) + 16(IV)
	if len(packetData) < minPacketSize {
		return nil, false
	}

	// Ensure we don't access out of bounds
	dataLen := len(packetData)
	if dataLen < 32+16 { // Need at least MAC + IV
		return nil, false
	}

	// Extract components with bounds checking
	ivStart := dataLen - 16
	macStart := dataLen - 32
	
	if ivStart < 0 || macStart < 0 || macStart >= ivStart {
		return nil, false
	}
	
	iv := packetData[ivStart:]
	mac := packetData[macStart:ivStart]
	cipherText := packetData[:macStart]
	
	if len(cipherText) < 29 { // Minimum plaintext size (increased from 21 to 29)
		return nil, false
	}

	// Verify MAC
	expectedMAC := hmac.New(sha256.New, keyH)
	expectedMAC.Write(cipherText)
	expectedMACSum := expectedMAC.Sum(nil)
	if len(expectedMACSum) < 16 {
		return nil, false
	}
	
	if !hmac.Equal(mac, expectedMACSum[:16]) {
		return nil, false
	}

	// Decrypt plaintext
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, false
	}
	stream := cipher.NewCTR(block, iv)
	plainText := make([]byte, len(cipherText))
	stream.XORKeyStream(plainText, cipherText)

	// Parse plaintext with additional validation
	if len(plainText) < 29 {
		return nil, false
	}
	
	if plainText[0] != protocolVersion {
		return nil, false
	}

	timestamp := binary.BigEndian.Uint32(plainText[1:5])
	now := time.Now().Unix()
	packetTime := int64(timestamp)
	
	// Prevent integer overflow in timestamp comparison
	if packetTime < 0 || now < 0 {
		return nil, false
	}
	
	if now-packetTime > validTimeWindow || packetTime-now > validTimeWindow {
		return nil, false
	}

	agentID := binary.BigEndian.Uint64(plainText[5:13])
	nonce16 := plainText[13:29] // 16-byte nonce

	if !nonceStore.IsValid(nonce16) {
		return nil, false // Replay attack detected
	}

	// For now, we'll extract the IP from the packet data itself.
	// This will be improved in the sniffer implementation.
	srcIP := "127.0.0.1" // Placeholder

	return &SPAInfo{AgentID: agentID, IP: srcIP}, true
}
