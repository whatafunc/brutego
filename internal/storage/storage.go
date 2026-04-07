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
// In-memory implementation
// ---------------------------------------------------------------------------

// MemoryStorage is a thread-safe in-memory implementation of Storage.
// Suitable for single-instance deployments; replace with a DB-backed
// implementation for multi-instance setups.
type MemoryStorage struct {
	mu        sync.RWMutex
	blacklist map[string]*net.IPNet // key: canonical CIDR string
	whitelist map[string]*net.IPNet
}

// NewMemoryStorage creates an empty MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		blacklist: make(map[string]*net.IPNet),
		whitelist: make(map[string]*net.IPNet),
	}
}

// AddToBlacklist adds a subnet to the blacklist.
func (m *MemoryStorage) AddToBlacklist(subnet string) error {
	return m.addTo(m.blacklist, subnet)
}

// RemoveFromBlacklist removes a subnet from the blacklist.
// Returns ErrSubnetNotFound if the subnet is not present.
func (m *MemoryStorage) RemoveFromBlacklist(subnet string) error {
	return m.removeFrom(m.blacklist, subnet)
}

// IsBlacklisted checks if the given IP is blacklisted.
func (m *MemoryStorage) IsBlacklisted(ip string) (bool, error) {
	return m.contains(m.blacklist, ip)
}

// AddToWhitelist adds a subnet to the whitelist.
func (m *MemoryStorage) AddToWhitelist(subnet string) error {
	return m.addTo(m.whitelist, subnet)
}

// RemoveFromWhitelist removes a subnet from the whitelist.
// Returns ErrSubnetNotFound if the subnet is not present.
func (m *MemoryStorage) RemoveFromWhitelist(subnet string) error {
	return m.removeFrom(m.whitelist, subnet)
}

// IsWhitelisted checks if the given IP is whitelisted.
// An IP is considered whitelisted if it belongs to any subnet in the whitelist,
// regardless of blacklist status.
func (m *MemoryStorage) IsWhitelisted(ip string) (bool, error) {
	return m.contains(m.whitelist, ip)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (m *MemoryStorage) addTo(list map[string]*net.IPNet, cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSubnet, cidr)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	list[network.String()] = network
	return nil
}

func (m *MemoryStorage) removeFrom(list map[string]*net.IPNet, cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSubnet, cidr)
	}

	key := network.String()

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := list[key]; !ok {
		return fmt.Errorf("%w: %s", ErrSubnetNotFound, cidr)
	}

	delete(list, key)
	return nil
}

func (m *MemoryStorage) contains(list map[string]*net.IPNet, ipStr string) (bool, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, network := range list {
		if network.Contains(ip) {
			return true, nil
		}
	}

	return false, nil
}
