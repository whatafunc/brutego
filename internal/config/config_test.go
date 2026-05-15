package config

import (
	"os"
	"testing"
)

const (
	defaultGRPCAddr = ":50051"
	defaultHTTPAddr = ":8080"
)

//nolint:funlen
func TestNew(t *testing.T) {
	// Our tested configs.
	tests := []struct {
		name       string
		setupEnv   func()
		want       *Config
		wantErr    bool
		errMessage string
	}{
		{
			name:     "default values when no env vars set",
			setupEnv: func() {},
			want: &Config{
				GRPCAddr:    ":defaultGRPCAddr",
				HTTPAddr:    ":defaultHTTPAddr",
				LoginRPM:    1,
				PasswordRPM: 100,
				IPRPM:       1000,
			},
			wantErr: false,
		},
		{
			name: "custom GRPC_ADDR",
			setupEnv: func() {
				_ = os.Setenv("GRPC_ADDR", ":9090")
			},
			want: &Config{
				GRPCAddr:    ":9090",
				HTTPAddr:    ":defaultHTTPAddr",
				LoginRPM:    1,
				PasswordRPM: 100,
				IPRPM:       1000,
			},
			wantErr: false,
		},
		{
			name: "custom HTTP_ADDR",
			setupEnv: func() {
				_ = os.Setenv("HTTP_ADDR", ":8888")
			},
			want: &Config{
				GRPCAddr:    ":defaultGRPCAddr",
				HTTPAddr:    ":8888",
				LoginRPM:    1,
				PasswordRPM: 100,
				IPRPM:       1000,
			},
			wantErr: false,
		},
		{
			name: "custom LIMIT_LOGIN",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_LOGIN", "20")
			},
			want: &Config{
				GRPCAddr:    ":defaultGRPCAddr",
				HTTPAddr:    ":defaultHTTPAddr",
				LoginRPM:    20,
				PasswordRPM: 100,
				IPRPM:       1000,
			},
			wantErr: false,
		},
		{
			name: "custom LIMIT_PASSWORD",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_PASSWORD", "200")
			},
			want: &Config{
				GRPCAddr:    ":defaultGRPCAddr",
				HTTPAddr:    ":defaultHTTPAddr",
				LoginRPM:    1,
				PasswordRPM: 200,
				IPRPM:       1000,
			},
			wantErr: false,
		},
		{
			name: "custom LIMIT_IP",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_IP", "5000")
			},
			want: &Config{
				GRPCAddr:    ":defaultGRPCAddr",
				HTTPAddr:    ":defaultHTTPAddr",
				LoginRPM:    1,
				PasswordRPM: 100,
				IPRPM:       5000,
			},
			wantErr: false,
		},
		{
			name: "all custom values",
			setupEnv: func() {
				_ = os.Setenv("GRPC_ADDR", ":50055")
				_ = os.Setenv("HTTP_ADDR", ":8085")
				_ = os.Setenv("LIMIT_LOGIN", "30")
				_ = os.Setenv("LIMIT_PASSWORD", "300")
				_ = os.Setenv("LIMIT_IP", "3000")
			},
			want: &Config{
				GRPCAddr:    ":50055",
				HTTPAddr:    ":8085",
				LoginRPM:    30,
				PasswordRPM: 300,
				IPRPM:       3000,
			},
			wantErr: false,
		},
		{
			name: "invalid LIMIT_LOGIN (non-numeric)",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_LOGIN", "invalid")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid LIMIT_PASSWORD (non-numeric)",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_PASSWORD", "not-a-number")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid LIMIT_IP (non-numeric)",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_IP", "abc")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "zero LoginRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_LOGIN", "0")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "zero PasswordRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_PASSWORD", "0")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "zero IPRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_IP", "0")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "negative LoginRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_LOGIN", "-5")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "negative PasswordRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_PASSWORD", "-10")
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "negative IPRPM",
			setupEnv: func() {
				_ = os.Setenv("LIMIT_IP", "-100")
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars before each test.
			_ = os.Unsetenv("GRPC_ADDR")
			_ = os.Unsetenv("HTTP_ADDR")
			_ = os.Unsetenv("LIMIT_LOGIN")
			_ = os.Unsetenv("LIMIT_PASSWORD")
			_ = os.Unsetenv("LIMIT_IP")

			// Setup test-specific env vars.
			if tt.setupEnv != nil {
				tt.setupEnv()
			}

			// Execute.
			got, err := New()

			// Assert error expectation.
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got nil")
				}
				return
			}

			// Assert no error.
			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			// Assert config values.
			if got.GRPCAddr != tt.want.GRPCAddr {
				t.Errorf("GRPCAddr = %v, want %v", got.GRPCAddr, tt.want.GRPCAddr)
			}
			if got.HTTPAddr != tt.want.HTTPAddr {
				t.Errorf("HTTPAddr = %v, want %v", got.HTTPAddr, tt.want.HTTPAddr)
			}
			if got.LoginRPM != tt.want.LoginRPM {
				t.Errorf("LoginRPM = %v, want %v", got.LoginRPM, tt.want.LoginRPM)
			}
			if got.PasswordRPM != tt.want.PasswordRPM {
				t.Errorf("PasswordRPM = %v, want %v", got.PasswordRPM, tt.want.PasswordRPM)
			}
			if got.IPRPM != tt.want.IPRPM {
				t.Errorf("IPRPM = %v, want %v", got.IPRPM, tt.want.IPRPM)
			}
		})
	}
}

// TestEnvPriority ensures that environment variables take precedence over defaults.
func TestEnvPriority(t *testing.T) {
	_ = os.Unsetenv("GRPC_ADDR")
	_ = os.Unsetenv("HTTP_ADDR")
	_ = os.Unsetenv("LIMIT_LOGIN")
	_ = os.Unsetenv("LIMIT_PASSWORD")
	_ = os.Unsetenv("LIMIT_IP")

	_ = os.Setenv("GRPC_ADDR", ":9999")
	_ = os.Setenv("HTTP_ADDR", ":9998")
	_ = os.Setenv("LIMIT_LOGIN", "99")
	_ = os.Setenv("LIMIT_PASSWORD", "999")
	_ = os.Setenv("LIMIT_IP", "9999")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	if cfg.GRPCAddr != ":9999" {
		t.Errorf("GRPCAddr = %v, want :9999", cfg.GRPCAddr)
	}
	if cfg.HTTPAddr != ":9998" {
		t.Errorf("HTTPAddr = %v, want :9998", cfg.HTTPAddr)
	}
	if cfg.LoginRPM != 99 {
		t.Errorf("LoginRPM = %v, want 99", cfg.LoginRPM)
	}
	if cfg.PasswordRPM != 999 {
		t.Errorf("PasswordRPM = %v, want 999", cfg.PasswordRPM)
	}
	if cfg.IPRPM != 9999 {
		t.Errorf("IPRPM = %v, want 9999", cfg.IPRPM)
	}
}