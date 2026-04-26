// Package storage provides persistence for whitelist and blacklist subnets.
package storage

import (
	"fmt"
	"net"
	"sync"
)

// ErrSubnetNotFound is returned when a subnet to be deleted does not exist.
var ErrSubnetNotFound = fmt.Errorf("subnet not found")

// Storage defines the persistence contract for whitelist/blacklist management.
// All methods accept already-parsed net types — parsing and validation is the
// responsibility of the caller (service layer).
type Storage interface {
	// AddToBlacklist adds a parsed subnet to the blacklist.
	AddToBlacklist(network *net.IPNet) error

	// RemoveFromBlacklist removes a subnet from the blacklist.
	// Returns ErrSubnetNotFound if the subnet is not present.
	RemoveFromBlacklist(network *net.IPNet) error

	// IsBlacklisted reports whether ip belongs to any blacklisted subnet.
	IsBlacklisted(ip net.IP) bool

	// AddToWhitelist adds a parsed subnet to the whitelist.
	AddToWhitelist(network *net.IPNet) error

	// RemoveFromWhitelist removes a subnet from the whitelist.
	// Returns ErrSubnetNotFound if the subnet is not present.
	RemoveFromWhitelist(network *net.IPNet) error

	// IsWhitelisted reports whether ip belongs to any whitelisted subnet.
	IsWhitelisted(ip net.IP) bool
}

// ---------------------------------------------------------------------------
// ipSet — a thread-safe set of CIDR subnets with its own mutex.
// ---------------------------------------------------------------------------

// ipSet is an internal type that owns both its data and its lock.
// blacklist and whitelist are independent instances so they never contend
// on each other's mutex.
type ipSet struct {
	mu      sync.RWMutex
	subnets map[string]*net.IPNet // key: canonical CIDR string e.g. "192.168.0.0/24"
}

func newIPSet() *ipSet {
	return &ipSet{
		subnets: make(map[string]*net.IPNet),
	}
}

// add inserts an already-parsed network into the set.
func (l *ipSet) add(network *net.IPNet) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.subnets[network.String()] = network
}

// remove deletes a network from the set.
// Returns ErrSubnetNotFound if it is not present.
func (l *ipSet) remove(network *net.IPNet) error {
	key := network.String()

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.subnets[key]; !ok {
		return fmt.Errorf("%w: %s", ErrSubnetNotFound, key)
	}

	delete(l.subnets, key)
	return nil
}

// contains reports whether ip falls within any subnet in the set.
func (l *ipSet) contains(ip net.IP) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, network := range l.subnets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// MemoryStorage — in-memory implementation of Storage.
// ---------------------------------------------------------------------------

// MemoryStorage is a thread-safe (new / after review) in-memory implementation of Storage.
// blacklist and whitelist each own their mutex so they never block each other.
// Suitable for single-instance deployments; replace with a DB-backed
// implementation for multi-instance setups.
type MemoryStorage struct {
	blacklist *ipSet
	whitelist *ipSet
}

// NewMemoryStorage creates an empty MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		blacklist: newIPSet(),
		whitelist: newIPSet(),
	}
}

// AddToBlacklist adds a subnet to the blacklist.
func (m *MemoryStorage) AddToBlacklist(network *net.IPNet) error {
	m.blacklist.add(network)
	return nil
}

// RemoveFromBlacklist removes a subnet from the blacklist.
func (m *MemoryStorage) RemoveFromBlacklist(network *net.IPNet) error {
	return m.blacklist.remove(network)
}

// IsBlacklisted checks if the given IP is blacklisted.
func (m *MemoryStorage) IsBlacklisted(ip net.IP) bool {
	return m.blacklist.contains(ip)
}

// AddToWhitelist adds a subnet to the whitelist.
func (m *MemoryStorage) AddToWhitelist(network *net.IPNet) error {
	m.whitelist.add(network)
	return nil
}

// RemoveFromWhitelist removes a subnet from the whitelist.
func (m *MemoryStorage) RemoveFromWhitelist(network *net.IPNet) error {
	return m.whitelist.remove(network)
}

// IsWhitelisted checks if the given IP is whitelisted.
// An IP is considered whitelisted if it belongs to any subnet in the whitelist,
// regardless of blacklist status.
func (m *MemoryStorage) IsWhitelisted(ip net.IP) bool {
	return m.whitelist.contains(ip)
}
