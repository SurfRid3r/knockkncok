
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	protocolVersion = 0x02
)

func createPacket(keyE, keyH []byte, serverIP string) ([]byte, error) {
	// 1. Plaintext - enhanced with 16-byte nonce
	plainText := make([]byte, 29) // Increased from 21 to 29 bytes
	plainText[0] = protocolVersion
	binary.BigEndian.PutUint32(plainText[1:5], uint32(time.Now().Unix()))
	
	agentID, err := getAgentID()
	if err != nil {
		return nil, fmt.Errorf("failed to get agent id: %w", err)
	}
	binary.BigEndian.PutUint64(plainText[5:13], agentID)

	// Enhanced 16-byte nonce for better security
	if _, err := rand.Read(plainText[13:29]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 2. Encrypt
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	stream := cipher.NewCTR(block, iv)
	cipherText := make([]byte, len(plainText))
	stream.XORKeyStream(cipherText, plainText)

	// 3. MAC
	mac := hmac.New(sha256.New, keyH)
	mac.Write(cipherText)
	hmacResult := mac.Sum(nil)[:16]

	// 4. Assemble packet
	packet := append(cipherText, hmacResult...)
	packet = append(packet, iv...)

	return packet, nil
}
