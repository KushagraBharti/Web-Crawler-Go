package crawler

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"
)

type TargetPolicyError struct {
	Code    string
	Message string
}

func (e *TargetPolicyError) Error() string {
	return e.Message
}

type TargetPolicy struct {
	lookup   func(ctx context.Context, host string) ([]net.IP, error)
	cacheTTL time.Duration

	mu    sync.Mutex
	cache map[string]targetDecision
}

type targetDecision struct {
	blocked bool
	reason  string
	expires time.Time
}

func NewTargetPolicy(cacheTTL time.Duration) *TargetPolicy {
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute
	}
	return &TargetPolicy{
		cacheTTL: cacheTTL,
		cache:    make(map[string]targetDecision),
		lookup: func(ctx context.Context, host string) ([]net.IP, error) {
			addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			out := make([]net.IP, 0, len(addrs))
			for _, addr := range addrs {
				out = append(out, addr.IP)
			}
			return out, nil
		},
	}
}

func (p *TargetPolicy) ValidateURL(ctx context.Context, target *url.URL) error {
	if target == nil {
		return &TargetPolicyError{Code: "invalid_target", Message: "target URL is required"}
	}
	scheme := strings.ToLower(target.Scheme)
	if scheme != "http" && scheme != "https" {
		return &TargetPolicyError{Code: "invalid_scheme", Message: "only http and https targets are allowed"}
	}

	host := strings.ToLower(strings.TrimSpace(target.Hostname()))
	if host == "" {
		return &TargetPolicyError{Code: "invalid_host", Message: "target host is required"}
	}
	if isBlockedHostname(host) {
		return &TargetPolicyError{Code: "blocked_hostname", Message: "target host is not allowed"}
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return &TargetPolicyError{Code: "blocked_ip", Message: "target IP address is not allowed"}
		}
		return nil
	}

	blocked, reason, err := p.lookupHost(ctx, host)
	if err != nil {
		return &TargetPolicyError{Code: "host_resolution_failed", Message: "unable to resolve target host"}
	}
	if blocked {
		return &TargetPolicyError{Code: "blocked_ip", Message: reason}
	}
	return nil
}

func (p *TargetPolicy) lookupHost(ctx context.Context, host string) (bool, string, error) {
	now := time.Now()

	p.mu.Lock()
	if cached, ok := p.cache[host]; ok && now.Before(cached.expires) {
		p.mu.Unlock()
		return cached.blocked, cached.reason, nil
	}
	p.mu.Unlock()

	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	addrs, err := p.lookup(lookupCtx, host)
	if err != nil {
		return false, "", err
	}
	if len(addrs) == 0 {
		return false, "", errors.New("no addresses resolved")
	}

	blocked := false
	reason := ""
	for _, ip := range addrs {
		if isBlockedIP(ip) {
			blocked = true
			reason = "target resolves to a private or reserved IP"
			break
		}
	}

	p.mu.Lock()
	p.cache[host] = targetDecision{
		blocked: blocked,
		reason:  reason,
		expires: now.Add(p.cacheTTL),
	}
	p.mu.Unlock()

	return blocked, reason, nil
}

func isBlockedHostname(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".intranet") {
		return true
	}
	// Disallow single-label hosts (e.g. "db", "internal"), which are usually internal.
	return !strings.Contains(host, ".")
}

var blockedPrefixes = []netip.Prefix{
	mustPrefix("100.64.0.0/10"),   // carrier-grade NAT
	mustPrefix("192.0.0.0/24"),    // IETF protocol assignments
	mustPrefix("192.0.2.0/24"),    // TEST-NET-1
	mustPrefix("198.18.0.0/15"),   // benchmark tests
	mustPrefix("198.51.100.0/24"), // TEST-NET-2
	mustPrefix("203.0.113.0/24"),  // TEST-NET-3
	mustPrefix("240.0.0.0/4"),     // reserved
	mustPrefix("2001:db8::/32"),   // documentation
}

func isBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsMulticast() || addr.IsLinkLocalMulticast() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func mustPrefix(cidr string) netip.Prefix {
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		panic(err)
	}
	return p
}
