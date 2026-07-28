package httputil

import "net"

// Reserved IPv4 ranges that net.IP's own predicates do not classify, but which
// should never be reachable from user-supplied URLs. net.IP.IsPrivate covers
// RFC 1918 and IPv6 unique-local (fc00::/7); net.IP.IsGlobalUnicast covers
// loopback, link-local, multicast, unspecified and broadcast. What is left is
// the set below — most importantly carrier-grade NAT, which is routable inside
// many hosting environments and is a real SSRF target.
var reservedIPv4Blocks = []net.IPNet{
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)},   // RFC 6598 carrier-grade NAT
	{IP: net.IPv4(192, 0, 0, 0), Mask: net.CIDRMask(24, 32)},    // RFC 6890 IETF protocol assignments
	{IP: net.IPv4(192, 0, 2, 0), Mask: net.CIDRMask(24, 32)},    // RFC 5737 TEST-NET-1
	{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)},   // RFC 2544 benchmarking
	{IP: net.IPv4(198, 51, 100, 0), Mask: net.CIDRMask(24, 32)}, // RFC 5737 TEST-NET-2
	{IP: net.IPv4(203, 0, 113, 0), Mask: net.CIDRMask(24, 32)},  // RFC 5737 TEST-NET-3
	{IP: net.IPv4(240, 0, 0, 0), Mask: net.CIDRMask(4, 32)},     // RFC 1112 reserved (includes 255.255.255.255)
}

// IsPublicIP reports whether ip is safe to open an outbound connection to for
// a user-supplied URL. It is deliberately an allowlist: anything not clearly a
// routable public address is rejected, so a range we failed to enumerate fails
// closed rather than open.
//
// IPv4-in-IPv6 forms (::ffff:127.0.0.1) are handled because To4 unwraps them
// before the IPv4 range checks run, and net.IP's own predicates are 4-in-6
// aware. 6to4 (2002::/16) and NAT64 (64:ff9b::/96) embed an IPv4 address that
// these checks would not otherwise unwrap, so they are unwrapped explicitly.
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if embedded := embeddedIPv4(ip); embedded != nil {
		ip = embedded
	}

	// Rejects loopback, link-local unicast/multicast (incl. the 169.254.169.254
	// cloud-metadata endpoint), multicast, unspecified and broadcast.
	if !ip.IsGlobalUnicast() {
		return false
	}

	// RFC 1918 / RFC 4193 unique-local.
	if ip.IsPrivate() {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		for i := range reservedIPv4Blocks {
			if reservedIPv4Blocks[i].Contains(ip4) {
				return false
			}
		}
	}

	return true
}

// embeddedIPv4 returns the IPv4 address carried inside a 6to4 or NAT64 address,
// or nil when ip carries neither. Without this, 2002:7f00:1:: (6to4 for
// 127.0.0.1) and 64:ff9b::7f00:1 (NAT64 for 127.0.0.1) would both pass as
// ordinary global-unicast IPv6 addresses.
func embeddedIPv4(ip net.IP) net.IP {
	ip16 := ip.To16()
	if ip16 == nil || ip.To4() != nil {
		return nil
	}

	// 6to4: 2002:AABB:CCDD::/48 carries IPv4 AA.BB.CC.DD in bytes 2-5.
	if ip16[0] == 0x20 && ip16[1] == 0x02 {
		return net.IPv4(ip16[2], ip16[3], ip16[4], ip16[5])
	}

	// NAT64 well-known prefix: 64:ff9b::/96 carries IPv4 in the low 32 bits.
	if ip16[0] == 0x00 && ip16[1] == 0x64 && ip16[2] == 0xff && ip16[3] == 0x9b {
		var allZero = true
		for _, b := range ip16[4:12] {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15])
		}
	}

	return nil
}

// FirstPublicIP returns the first address in ips that IsPublicIP accepts, or
// nil when none qualify. Callers pin the returned address for the actual dial
// so that a second DNS lookup cannot return a different (internal) answer —
// see SafeDialContext.
func FirstPublicIP(ips []net.IP) net.IP {
	for _, ip := range ips {
		if IsPublicIP(ip) {
			return ip
		}
	}
	return nil
}
