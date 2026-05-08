package storage_test

import (
	"net"
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/whatafunc/brutego/internal/storage"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustParseCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("mustParseCIDR(%q): %v", cidr, err)
	}
	return network
}

func mustParseIP(t *testing.T, ip string) net.IP {
	t.Helper()
	parsed := net.ParseIP(ip)
	if parsed == nil {
		t.Fatalf("mustParseIP(%q): invalid", ip)
	}
	return parsed
}

// based on 03-anti-bruteforce-API-tests.md

// ---------------------------------------------------------------------------
// blacklist
// ---------------------------------------------------------------------------

func TestMemoryStorage_Blacklist_AddAndContains(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "192.168.5.0/24")
	ip := mustParseIP(t, "192.168.5.42")

	require.NoError(t, s.AddToBlacklist(network))

	if !s.IsBlacklisted(ip) {
		t.Fatal("expected IP to be blacklisted")
	}
}

func TestMemoryStorage_Blacklist_IPOutsideSubnet(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "192.168.5.0/24")
	ip := mustParseIP(t, "10.0.0.1")

	require.NoError(t, s.AddToBlacklist(network))

	if s.IsBlacklisted(ip) {
		t.Fatal("expected IP outside subnet to not be blacklisted")
	}
}

func TestMemoryStorage_Blacklist_Remove(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "192.168.5.0/24")
	ip := mustParseIP(t, "192.168.5.1")

	require.NoError(t, s.AddToBlacklist(network))

	if err := s.RemoveFromBlacklist(network); err != nil {
		t.Fatalf("unexpected error on remove: %v", err)
	}

	if s.IsBlacklisted(ip) {
		t.Fatal("expected IP to not be blacklisted after removal")
	}
}

func TestMemoryStorage_Blacklist_RemoveNotFound(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "10.0.0.0/8")

	err := s.RemoveFromBlacklist(network)
	if err == nil {
		t.Fatal("expected ErrSubnetNotFound, got nil")
	}
}

// ---------------------------------------------------------------------------
// whitelist
// ---------------------------------------------------------------------------

func TestMemoryStorage_Whitelist_AddAndContains(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "10.0.0.0/8")
	ip := mustParseIP(t, "10.1.2.3")

	require.NoError(t, s.AddToWhitelist(network))

	if !s.IsWhitelisted(ip) {
		t.Fatal("expected IP is supposed to be whitelisted")
	}
}

func TestMemoryStorage_Whitelist_RemoveNotFound(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "10.0.0.0/8")

	err := s.RemoveFromWhitelist(network)
	if err == nil {
		t.Fatal("expected ErrSubnetNotFound, got nil")
	}
}

// ---------------------------------------------------------------------------
// independence — blacklist and whitelist do not affect each other
// ---------------------------------------------------------------------------

func TestMemoryStorage_BlacklistAndWhitelist_AreIndependent(t *testing.T) {
	t.Parallel()

	s := storage.NewMemoryStorage()
	network := mustParseCIDR(t, "192.168.0.0/16")
	ip := mustParseIP(t, "192.168.1.1")

	require.NoError(t, s.AddToBlacklist(network))

	if s.IsWhitelisted(ip) {
		t.Fatal("blacklisted subnet should not affect whitelist")
	}
}
