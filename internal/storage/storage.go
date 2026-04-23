// Package storage provides persistence for whitelist and blacklist subnets.
package storage

import (
	"fmt"
	"net"
	"sync"
)

// ErrSubnetNotFound is returned when a subnet to be deleted does not exist.
var ErrSubnetNotFound = fmt.Errorf("subnet not found")

// ErrInvalidSubnet is returned when a CIDR string cannot be parsed.
var ErrInvalidSubnet = fmt.Errorf("invalid subnet")

// Storage defines the persistence contract for whitelist/blacklist management.
type Storage interface {
	// AddToBlacklist adds a CIDR subnet to the blacklist.
	AddToBlacklist(subnet string) error

	// RemoveFromBlacklist removes a CIDR subnet from the blacklist.
	// Returns ErrSubnetNotFound if the subnet is not present.
	RemoveFromBlacklist(subnet string) error

	// IsBlacklisted reports whether the given IPv4 address belongs to any
	// blacklisted subnet.
	IsBlacklisted(ip string) (bool, error)

	// AddToWhitelist adds a CIDR subnet to the whitelist.
	AddToWhitelist(subnet string) error

	// RemoveFromWhitelist removes a CIDR subnet from the whitelist.
	// Returns ErrSubnetNotFound if the subnet is not present.
	RemoveFromWhitelist(subnet string) error

	// IsWhitelisted reports whether the given IPv4 address belongs to any
	// whitelisted subnet.
	IsWhitelisted(ip string) (bool, error)
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

// add parses cidr and inserts it into the list.
// Returns ErrInvalidSubnet if cidr cannot be parsed.
func (l *ipSet) add(cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSubnet, cidr)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.subnets[network.String()] = network
	return nil
}

// remove deletes cidr from the list.
// Returns ErrSubnetNotFound if cidr is not present, ErrInvalidSubnet if unparseable.
func (l *ipSet) remove(cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSubnet, cidr)
	}

	key := network.String()

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.subnets[key]; !ok {
		return fmt.Errorf("%w: %s", ErrSubnetNotFound, cidr)
	}

	delete(l.subnets, key)
	return nil
}

// contains reports whether ipStr falls within any subnet in the list.
// Returns an error if ipStr is not a valid IP address.
func (l *ipSet) contains(ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, network := range l.subnets {
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
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
func (m *MemoryStorage) AddToBlacklist(subnet string) error {
	return m.blacklist.add(subnet)
}

// RemoveFromBlacklist removes a subnet from the blacklist.
// Returns ErrSubnetNotFound if the subnet is not
func (m *MemoryStorage) RemoveFromBlacklist(subnet string) error {
	return m.blacklist.remove(subnet)
}

// IsBlacklisted checks if the given IP is blacklisted.
func (m *MemoryStorage) IsBlacklisted(ip string) (bool, error) {
	return m.blacklist.contains(ip)
}

// AddToWhitelist adds a subnet to the whitelist.
func (m *MemoryStorage) AddToWhitelist(subnet string) error {
	return m.whitelist.add(subnet)
}

// RemoveFromWhitelist removes a subnet from the whitelist.
// Returns ErrSubnetNotFound if the subnet is not present.
func (m *MemoryStorage) RemoveFromWhitelist(subnet string) error {
	return m.whitelist.remove(subnet)
}

// IsWhitelisted checks if the given IP is whitelisted.
// An IP is considered whitelisted if it belongs to any subnet in the whitelist,
// regardless of blacklist status.
func (m *MemoryStorage) IsWhitelisted(ip string) (bool, error) {
	return m.whitelist.contains(ip)
}
