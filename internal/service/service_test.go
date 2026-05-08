package service_test

import (
	"context"
	"net"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/stretchr/testify/require"
	pb "github.com/whatafunc/brutego/pkg/api/antibruteforce/v1"
	"github.com/whatafunc/brutego/internal/service"
	"github.com/whatafunc/brutego/internal/storage"
)

// based on 03-anti-bruteforce-API-tests.md

// ---------------------------------------------------------------------------
// mock storage 
// ---------------------------------------------------------------------------

type mockStorage struct {
	whitelisted bool
	blacklisted bool
	removeErr   error
}

func (m *mockStorage) IsWhitelisted(_ net.IP) bool           { return m.whitelisted }
func (m *mockStorage) IsBlacklisted(_ net.IP) bool           { return m.blacklisted }
func (m *mockStorage) AddToBlacklist(_ *net.IPNet) error     { return nil }
func (m *mockStorage) AddToWhitelist(_ *net.IPNet) error     { return nil }
func (m *mockStorage) RemoveFromBlacklist(_ *net.IPNet) error { return m.removeErr }
func (m *mockStorage) RemoveFromWhitelist(_ *net.IPNet) error { return m.removeErr }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newService(t *testing.T, store storage.Storage, loginRPM int) *service.Service {
	t.Helper()
	return service.New(service.Deps{
		Logger:  zap.NewNop(),
		Storage: store,
		Limits: service.Limits{
			LoginRPM:    loginRPM,
			PasswordRPM: 100,
			IPRPM:       1000,
		},
	})
}

var validReq = &pb.CheckAuthRequest{
	Login:    "testuser",
	Password: "testpass123",
	Ip:       "192.168.1.100",
}

// ---------------------------------------------------------------------------
// CheckAuth
// ---------------------------------------------------------------------------

func TestCheckAuth_Allowed(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	resp, err := svc.CheckAuth(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Ok {
		t.Fatal("expected ok=true")
	}
}

func TestCheckAuth_WhitelistedIP_ShortCircuits(t *testing.T) {
	t.Parallel()

	// even with limit=0 whitelist should always allow
	svc := newService(t, &mockStorage{whitelisted: true}, 0)
	resp, err := svc.CheckAuth(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Ok {
		t.Fatal("expected ok=true for whitelisted IP")
	}
}

func TestCheckAuth_BlacklistedIP_ShortCircuits(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{blacklisted: true}, 10)
	resp, err := svc.CheckAuth(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ok {
		t.Fatal("expected ok=false for blacklisted IP")
	}
}

func TestCheckAuth_RateLimitExceeded(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 1)

	// first call — allowed
	resp, err := svc.CheckAuth(context.Background(), validReq)
	if err != nil || !resp.Ok {
		t.Fatal("expected first call to be allowed")
	}

	// second call — denied
	resp, err = svc.CheckAuth(context.Background(), validReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ok {
		t.Fatal("expected ok=false after rate limit exceeded")
	}
}

func TestCheckAuth_EmptyLogin_InvalidArgument(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	_, err := svc.CheckAuth(context.Background(), &pb.CheckAuthRequest{
		Login:    "",
		Password: "secret",
		Ip:       "192.168.1.1",
	})

	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

func TestCheckAuth_InvalidIP_InvalidArgument(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	_, err := svc.CheckAuth(context.Background(), &pb.CheckAuthRequest{
		Login:    "alice",
		Password: "secret",
		Ip:       "not-an-ip",
	})

	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

// ---------------------------------------------------------------------------
// ResetBucket
// ---------------------------------------------------------------------------

func TestResetBucket_ClearsLoginBucket(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 1)

	// exhaust the login bucket
	resp, err := svc.CheckAuth(context.Background(), validReq)
	require.NoError(t, err)
	require.True(t, resp.Ok)

	// reset
	_, err = svc.ResetBucket(context.Background(), &pb.ResetBucketRequest{
		Login: validReq.Login,
		Ip:    validReq.Ip,
	})
	if err != nil {
		t.Fatalf("unexpected error on reset: %v", err)
	}

	// should be allowed again
	resp, err := svc.CheckAuth(context.Background(), validReq)
	if err != nil || !resp.Ok {
		t.Fatal("expected ok=true after bucket reset")
	}

	// invalid reset (without login or ip) should return error
	_, err = svc.ResetBucket(context.Background(), &pb.ResetBucketRequest{
	})
	if err == nil {
		t.Fatalf("unexpected nil error for invalid reset request")
	}
}

func TestResetBucket_EmptyRequest_InvalidArgument(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	_, err := svc.ResetBucket(context.Background(), &pb.ResetBucketRequest{})

	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

// ---------------------------------------------------------------------------
// AddToBlacklist / AddToWhitelist — invalid subnet
// ---------------------------------------------------------------------------

func TestAddToBlacklist_InvalidSubnet_InvalidArgument(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	_, err := svc.AddToBlacklist(context.Background(), &pb.SubnetRequest{Subnet: "not-a-cidr"})

	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

func TestAddToWhitelist_InvalidSubnet_InvalidArgument(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{}, 10)
	_, err := svc.AddToWhitelist(context.Background(), &pb.SubnetRequest{Subnet: "not-a-cidr"})

	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", code)
	}
}

// ---------------------------------------------------------------------------
// RemoveFromBlacklist — not found
// ---------------------------------------------------------------------------

func TestRemoveFromBlacklist_NotFound(t *testing.T) {
	t.Parallel()

	svc := newService(t, &mockStorage{removeErr: storage.ErrSubnetNotFound}, 10)
	_, err := svc.RemoveFromBlacklist(context.Background(), &pb.SubnetRequest{Subnet: "10.0.0.0/8"})

	if code := status.Code(err); code != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", code)
	}
}
