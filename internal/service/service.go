// Package service implements the AntiBruteforceService gRPC interface.
package service

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/whatafunc/brutego/internal/bucket"
	"github.com/whatafunc/brutego/internal/storage"
	pb "github.com/whatafunc/brutego/pkg/api/antibruteforce/v1"
)

// Limits holds the configured requests-per-minute thresholds.
type Limits struct {
	LoginRPM    int
	PasswordRPM int
	IPRPM       int
}

// Deps groups the external dependencies injected into Service.
type Deps struct {
	Logger  *zap.Logger
	Storage storage.Storage
	Limits  Limits
}

// Service implements pb.AntiBruteforceServiceServer.
type Service struct {
	pb.UnimplementedAntiBruteforceServiceServer

	log     *zap.Logger
	storage storage.Storage

	loginBuckets    *bucket.Store
	passwordBuckets *bucket.Store
	ipBuckets       *bucket.Store
}

// New creates a fully initialised Service.
func New(deps Deps) *Service {
	return &Service{
		log:             deps.Logger,
		storage:         deps.Storage,
		loginBuckets:    bucket.NewStore(deps.Limits.LoginRPM),
		passwordBuckets: bucket.NewStore(deps.Limits.PasswordRPM),
		ipBuckets:       bucket.NewStore(deps.Limits.IPRPM),
	}
}

// ---------------------------------------------------------------------------
// Authorization
// ---------------------------------------------------------------------------

// CheckAuth decides whether an authorization attempt should be allowed.
func (s *Service) CheckAuth(_ context.Context, req *pb.CheckAuthRequest) (*pb.CheckAuthResponse, error) {
	if err := validateCheckAuth(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 1. Whitelist check — short-circuit allow.
	whitelisted, err := s.storage.IsWhitelisted(req.Ip)
	if err != nil {
		s.log.Error("whitelist check failed", zap.String("ip", req.Ip), zap.Error(err))
		return nil, status.Error(codes.Internal, "whitelist check failed")
	}
	if whitelisted {
		return &pb.CheckAuthResponse{Ok: true}, nil
	}

	// 2. Blacklist check — short-circuit deny.
	blacklisted, err := s.storage.IsBlacklisted(req.Ip)
	if err != nil {
		s.log.Error("blacklist check failed", zap.String("ip", req.Ip), zap.Error(err))
		return nil, status.Error(codes.Internal, "blacklist check failed")
	}
	if blacklisted {
		return &pb.CheckAuthResponse{Ok: false}, nil
	}

	// 3. Rate-limit checks — all three buckets must allow.
	ok := s.loginBuckets.Allow(req.Login) &&
		s.passwordBuckets.Allow(req.Password) &&
		s.ipBuckets.Allow(req.Ip)

	return &pb.CheckAuthResponse{Ok: ok}, nil
}

// ResetBucket clears the login and IP buckets for the given identifiers.
func (s *Service) ResetBucket(_ context.Context, req *pb.ResetBucketRequest) (*emptypb.Empty, error) {
	if req.Login == "" && req.Ip == "" {
		return nil, status.Error(codes.InvalidArgument, "login or ip must be provided")
	}

	if req.Login != "" {
		s.loginBuckets.Reset(req.Login)
	}
	if req.Ip != "" {
		s.ipBuckets.Reset(req.Ip)
	}

	return &emptypb.Empty{}, nil
}

// ---------------------------------------------------------------------------
// Blacklist
// ---------------------------------------------------------------------------

// AddToBlacklist adds a CIDR subnet to the blacklist.
func (s *Service) AddToBlacklist(_ context.Context, req *pb.SubnetRequest) (*emptypb.Empty, error) {
	if err := s.storage.AddToBlacklist(req.Subnet); err != nil {
		return nil, subnetError(err, req.Subnet)
	}
	return &emptypb.Empty{}, nil
}

// RemoveFromBlacklist removes a CIDR subnet from the blacklist.
func (s *Service) RemoveFromBlacklist(_ context.Context, req *pb.SubnetRequest) (*emptypb.Empty, error) {
	if err := s.storage.RemoveFromBlacklist(req.Subnet); err != nil {
		return nil, subnetError(err, req.Subnet)
	}
	return &emptypb.Empty{}, nil
}

// ---------------------------------------------------------------------------
// Whitelist
// ---------------------------------------------------------------------------

// AddToWhitelist adds a CIDR subnet to the whitelist.
func (s *Service) AddToWhitelist(_ context.Context, req *pb.SubnetRequest) (*emptypb.Empty, error) {
	if err := s.storage.AddToWhitelist(req.Subnet); err != nil {
		return nil, subnetError(err, req.Subnet)
	}
	return &emptypb.Empty{}, nil
}

// RemoveFromWhitelist removes a CIDR subnet from the whitelist.
func (s *Service) RemoveFromWhitelist(_ context.Context, req *pb.SubnetRequest) (*emptypb.Empty, error) {
	if err := s.storage.RemoveFromWhitelist(req.Subnet); err != nil {
		return nil, subnetError(err, req.Subnet)
	}
	return &emptypb.Empty{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func validateCheckAuth(req *pb.CheckAuthRequest) error {
	if req.Login == "" {
		return errors.New("login is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if req.Ip == "" {
		return errors.New("ip is required")
	}
	return nil
}

func subnetError(err error, subnet string) error {
	switch {
	case errors.Is(err, storage.ErrSubnetNotFound):
		return status.Errorf(codes.NotFound, "subnet not found: %s", subnet)
	case errors.Is(err, storage.ErrInvalidSubnet):
		return status.Errorf(codes.InvalidArgument, "invalid subnet: %s", subnet)
	default:
		return status.Errorf(codes.Internal, "storage error: %v", err)
	}
}
