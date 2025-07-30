package main

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

// NonceStore is a thread-safe, expiring cache for nonces.
type NonceStore struct {
	mu      sync.Mutex
	nonces  map[uint64]time.Time
	expiry  time.Duration
}

// NewNonceStore creates a new nonce store with a given expiration duration.
func NewNonceStore(expiry time.Duration) *NonceStore {
	ns := &NonceStore{
		nonces: make(map[uint64]time.Time),
		expiry: expiry,
	}
	go ns.cleanupLoop()
	return ns
}

// IsValid checks if a nonce is new. If it is, it's added to the store and returns true.
// If the nonce has been seen before, it returns false.
func (ns *NonceStore) IsValid(nonce16 []byte) bool {
	// Hash 16-byte nonce to 8-byte for storage
	if len(nonce16) != 16 {
		return false
	}
	
	hash := sha256.Sum256(nonce16)
	nonce := binary.BigEndian.Uint64(hash[:8])
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, found := ns.nonces[nonce]; found {
		return false // Nonce already seen
	}

	ns.nonces[nonce] = time.Now()
	return true
}

// cleanupLoop periodically removes expired nonces from the store.
func (ns *NonceStore) cleanupLoop() {
	ticker := time.NewTicker(ns.expiry / 2)
	defer ticker.Stop()

	for range ticker.C {
		ns.mu.Lock()
		for nonce, timestamp := range ns.nonces {
			if time.Since(timestamp) > ns.expiry {
				delete(ns.nonces, nonce)
			}
		}
		ns.mu.Unlock()
	}
}
