package ingest

import (
	"net"
	"net/url"
	"strings"
)

const (
	// DefaultAddr is the loopback HTTP listener the browser extension posts to.
	DefaultAddr = "127.0.0.1:9477"
	// DefaultName is the inbox source name for browser events.
	DefaultName = "browser"
	// Disabled turns the listener off (used by tests).
	Disabled = "off"
)

// NormalizeAddr returns a loopback host:port. Non-loopback hosts are forced to 127.0.0.1.
func NormalizeAddr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, Disabled) {
		return s
	}
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		}
	}
	if !strings.Contains(s, ":") {
		s = "127.0.0.1:" + s
	} else if strings.HasPrefix(s, ":") {
		s = "127.0.0.1" + s
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return DefaultAddr
	}
	if port == "" {
		port = "9477"
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
