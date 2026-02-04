package crawler

import (
	"errors"
	"net"
	"net/url"
	"path"
	"strings"
)

var ErrUnsupportedScheme = errors.New("unsupported scheme")

func Canonicalize(raw string) (string, *url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", nil, err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", nil, ErrUnsupportedScheme
	}
	parsed.Scheme = scheme
	parsed.Fragment = ""
	if parsed.Host == "" {
		return "", nil, errors.New("missing host")
	}
	host := strings.ToLower(parsed.Host)
	host = strings.TrimSuffix(host, ".")
	parsed.Host = normalizeHostPort(host, scheme)
	parsed.Path = cleanPath(parsed.Path)
	if parsed.RawQuery != "" {
		q := parsed.Query()
		parsed.RawQuery = q.Encode()
	}
	return parsed.String(), parsed, nil
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	clean := path.Clean(p)
	if !strings.HasPrefix(clean, "/") {
		clean = "/" + clean
	}
	return clean
}

func normalizeHostPort(host, scheme string) string {
	if strings.Contains(host, ":") {
		h, port, err := net.SplitHostPort(host)
		if err == nil {
			if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
				return h
			}
			return net.JoinHostPort(h, port)
		}
	}
	return host
}

func HostKey(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	host = strings.TrimSuffix(host, ".")
	port := u.Port()
	if port == "" {
		return host
	}
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		return host
	}
	return net.JoinHostPort(host, port)
}