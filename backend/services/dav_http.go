package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// builds a DialContext that refuses connections to private/loopback/link-local addresses (SSRF protection for cloud
// deployments). The caller supplies its feature-specific sentinel errors so
// controllers can map failures without cross-feature coupling.
func newPrivateBlockingDialContext(errUnreachable, errPrivateAddress error) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("%w: could not resolve host", errUnreachable)
		}
		var safeIP net.IP
		for _, ip := range ips {
			if !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
				safeIP = ip
				break
			}
		}
		if safeIP == nil {
			return nil, errPrivateAddress
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 10 * time.Second}
		return dialer.DialContext(ctx, network, net.JoinHostPort(safeIP.String(), port))
	}
}

// limitedBody caps a response body's size, returning errTooLarge once exceeded.
type limitedBody struct {
	inner       io.ReadCloser
	remaining   int64
	errTooLarge error
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		// The limit was consumed exactly; only fail if more data follows.
		var probe [1]byte
		n, err := b.inner.Read(probe[:])
		if n > 0 {
			return 0, b.errTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.inner.Read(p)
	b.remaining -= int64(n)
	return n, err
}

func (b *limitedBody) Close() error { return b.inner.Close() }
