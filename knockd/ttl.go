
package main

import (
	"math/bits"
)

// TTLEngine calculates the TTL for a given agent and IP.
type TTLEngine struct {
	baseTTL int
	maxTTL  int
	db      *DB
}

// NewTTLEngine creates a new TTL engine.
func NewTTLEngine(baseTTL, maxTTL int, db *DB) *TTLEngine {
	return &TTLEngine{baseTTL: baseTTL, maxTTL: maxTTL, db: db}
}

// Next calculates the next TTL for the given agent and IP.
func (e *TTLEngine) Next(agentID uint64, ip string) int {
	score, _ := e.db.GetScore(agentID, ip)
	ttl := e.baseTTL * (1 << bits.Len(uint(score+1)))
	if ttl > e.maxTTL {
		ttl = e.maxTTL
	}
	return ttl
}
