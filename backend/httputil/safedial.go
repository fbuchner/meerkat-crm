package httputil

import (
	"context"
	"net"
	"time"
)

// DialTimeout is the connect/keep-alive timeout used by SSRF-guarded dialers.
const DialTimeout = 10 * time.Second

// SafeDialContext builds a DialContext that resolves the target host, refuses
// to connect unless at least one answer is a public address, and then dials
// that resolved IP directly.
//
// Pinning the IP is the point: validating a URL and then handing the hostname
// to the transport lets a second DNS lookup return a different answer (DNS
// rebinding), and lets an HTTP redirect reach an address the pre-flight check
// never saw. Because every connection a transport opens — including ones for
// redirect targets — goes through this dialer, both bypasses are closed here
// rather than at the URL layer.
//
// unreachableErr is returned when the host cannot be resolved and privateErr
// when it resolves only to non-public addresses. Callers pass their own
// sentinels so domain-specific error handling (calendar sync, contact sync)
// keeps working unchanged.
func SafeDialContext(unreachableErr, privateErr error) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, unreachableErr
		}

		safeIP := FirstPublicIP(ips)
		if safeIP == nil {
			return nil, privateErr
		}

		dialer := &net.Dialer{Timeout: DialTimeout, KeepAlive: DialTimeout}
		return dialer.DialContext(ctx, network, net.JoinHostPort(safeIP.String(), port))
	}
}
