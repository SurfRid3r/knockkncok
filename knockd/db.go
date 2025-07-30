
package main

import (
	"go.etcd.io/bbolt"
)

// DB is a wrapper around a bbolt database.
type DB struct {
	db *bbolt.DB
}

// NewDB creates a new DB.
func NewDB(path string) (*DB, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

// GetScore gets the score for a given agent and IP.
func (db *DB) GetScore(agentID uint64, ip string) (int, error) {
	// Placeholder implementation
	return 0, nil
}

// IncrementScore increments the score for a given agent and IP.
func (db *DB) IncrementScore(agentID uint64, ip string) error {
	// Placeholder implementation
	return nil
}

// Close closes the database.
func (db *DB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}
