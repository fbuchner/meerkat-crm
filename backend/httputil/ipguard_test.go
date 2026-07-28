package httputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name   string
		ip     string
		public bool
	}{
		// Routable addresses the guard must not block.
		{"public IPv4", "8.8.8.8", true},
		{"public IPv4 other", "1.1.1.1", true},
		{"public IPv6", "2606:4700:4700::1111", true},

		// Classified by net.IP's own predicates.
		{"loopback IPv4", "127.0.0.1", false},
		{"loopback IPv6", "::1", false},
		{"private 10/8", "10.0.0.1", false},
		{"private 172.16/12", "172.16.5.4", false},
		{"private 192.168/16", "192.168.1.1", false},
		{"unique local IPv6", "fd00::1", false},
		{"unspecified", "0.0.0.0", false},
		{"multicast", "224.0.0.1", false},
		{"broadcast", "255.255.255.255", false},

		// Cloud metadata endpoint — the highest-value SSRF target.
		{"link-local metadata", "169.254.169.254", false},
		{"link-local IPv6", "fe80::1", false},

		// Ranges the previous checks missed.
		{"CGNAT low", "100.64.0.1", false},
		{"CGNAT high", "100.127.255.254", false},
		{"IETF protocol assignments", "192.0.0.1", false},
		{"TEST-NET-1", "192.0.2.1", false},
		{"benchmarking", "198.18.0.1", false},
		{"TEST-NET-2", "198.51.100.1", false},
		{"TEST-NET-3", "203.0.113.1", false},
		{"reserved 240/4", "240.0.0.1", false},

		// CGNAT boundaries: 100.63.x and 100.128.x are outside 100.64.0.0/10.
		{"just below CGNAT", "100.63.255.255", true},
		{"just above CGNAT", "100.128.0.0", true},

		// IPv6 forms that wrap an internal IPv4 address.
		{"IPv4-mapped loopback", "::ffff:127.0.0.1", false},
		{"IPv4-mapped private", "::ffff:10.0.0.1", false},
		{"IPv4-mapped metadata", "::ffff:169.254.169.254", false},
		{"6to4 wrapping loopback", "2002:7f00:1::", false},
		{"6to4 wrapping metadata", "2002:a9fe:a9fe::", false},
		{"6to4 wrapping public", "2002:808:808::", true},
		{"NAT64 wrapping loopback", "64:ff9b::7f00:1", false},
		{"NAT64 wrapping private", "64:ff9b::a00:1", false},
		{"NAT64 wrapping public", "64:ff9b::808:808", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.NotNil(t, ip, "test fixture %q must parse as an IP", tt.ip)
			assert.Equal(t, tt.public, IsPublicIP(ip))
		})
	}
}

func TestIsPublicIPNil(t *testing.T) {
	assert.False(t, IsPublicIP(nil), "a nil IP must fail closed")
}

func TestFirstPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want string
	}{
		{"empty", nil, ""},
		{"all private", []string{"127.0.0.1", "10.0.0.1"}, ""},
		{"public first", []string{"8.8.8.8", "10.0.0.1"}, "8.8.8.8"},
		{"public after private", []string{"127.0.0.1", "8.8.8.8"}, "8.8.8.8"},
		{"skips CGNAT", []string{"100.64.0.1", "1.1.1.1"}, "1.1.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ips []net.IP
			for _, raw := range tt.ips {
				ips = append(ips, net.ParseIP(raw))
			}

			got := FirstPublicIP(ips)
			if tt.want == "" {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.True(t, got.Equal(net.ParseIP(tt.want)), "got %v, want %s", got, tt.want)
		})
	}
}
