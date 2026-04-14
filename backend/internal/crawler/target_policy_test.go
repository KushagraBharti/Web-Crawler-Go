package crawler

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestTargetPolicyValidateURL(t *testing.T) {
	policy := NewTargetPolicy(time.Minute)
	policy.lookup = func(ctx context.Context, host string) ([]net.IP, error) {
		switch host {
		case "example.com":
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		case "internal.example":
			return []net.IP{net.ParseIP("10.0.0.1")}, nil
		default:
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
	}

	tests := []struct {
		name    string
		rawURL  string
		wantErr string
	}{
		{name: "public target allowed", rawURL: "https://example.com", wantErr: ""},
		{name: "invalid scheme blocked", rawURL: "ftp://example.com/file", wantErr: "invalid_scheme"},
		{name: "localhost blocked", rawURL: "http://localhost:8080", wantErr: "blocked_hostname"},
		{name: "single label blocked", rawURL: "http://internal", wantErr: "blocked_hostname"},
		{name: "private ip blocked", rawURL: "http://10.1.2.3", wantErr: "blocked_ip"},
		{name: "resolved private ip blocked", rawURL: "https://internal.example", wantErr: "blocked_ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("parse url: %v", err)
			}
			err = policy.ValidateURL(context.Background(), parsed)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr != "" {
				var targetErr *TargetPolicyError
				if err == nil {
					t.Fatalf("expected error code %s", tt.wantErr)
				}
				if ok := asTargetPolicyError(err, &targetErr); !ok {
					t.Fatalf("expected TargetPolicyError, got %T", err)
				}
				if targetErr.Code != tt.wantErr {
					t.Fatalf("expected %s, got %s", tt.wantErr, targetErr.Code)
				}
			}
		})
	}
}

func TestIsBlockedHostname(t *testing.T) {
	if !isBlockedHostname("localhost") {
		t.Fatal("localhost should be blocked")
	}
	if !isBlockedHostname("intranet") {
		t.Fatal("single-label hostname should be blocked")
	}
	if isBlockedHostname("example.com") {
		t.Fatal("public hostname should be allowed")
	}
}

func asTargetPolicyError(err error, target **TargetPolicyError) bool {
	typed, ok := err.(*TargetPolicyError)
	if !ok {
		return false
	}
	*target = typed
	return true
}
