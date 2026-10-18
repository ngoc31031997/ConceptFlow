// Package sourcefetch downloads the web pages that research facts cite. The
// links come from outside the app, so every address is checked before and while
// connecting: only public hosts on the web ports are reached. See ADR-0043.
package sourcefetch

import (
	"net/netip"
	"net/url"
	"strings"

	"authoring/internal/application"
	"authoring/internal/domain"
)

const (
	reasonNotWeb     = "link không hợp lệ (cần http hoặc https, không kèm tài khoản)"
	reasonPort       = "cổng không được phép (chỉ 80 và 443)"
	reasonInternal   = "địa chỉ nội bộ không được phép"
	reasonNoRedirect = "chuyển hướng quá 5 lần"
)

// internalSuffixes are host name endings that only name machines on a private network.
var internalSuffixes = []string{".local", ".internal", ".localdomain", ".lan"}

// blockedPrefixes are address ranges that are not on the public Internet and
// that the standard library's classifiers do not cover.
var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15", "240.0.0.0/4", "255.255.255.255/32",
)

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}

// publicAddress reports whether ip is on the public Internet: not loopback,
// private, link-local (which holds the cloud metadata address), multicast,
// unspecified, or in a reserved range. An IPv4 address written as IPv6
// (::ffff:10.0.0.1) is judged as the IPv4 address.
func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, p := range blockedPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

// internalName reports whether a host name is one that only names a machine on
// a private network: localhost, a name with no dot (the service names of the
// Docker network are all like that), or one ending in an internal suffix.
func internalName(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" || !strings.Contains(host, ".") {
		return true
	}
	for _, s := range internalSuffixes {
		if strings.HasSuffix(host, s) {
			return true
		}
	}
	return false
}

// defaultPort reports whether port is one of the web ports ("" means the scheme's own).
func defaultPort(port string) bool { return port == "" || port == "80" || port == "443" }

// checkURL validates a link before any connection: a web scheme, no
// credentials, an allowed port, and a host that is neither an internal name
// nor (when written as an address) one allowAddr refuses. Names that resolve
// to a refused address are caught later, when connecting.
func checkURL(raw string, allowAddr func(netip.Addr) bool, allowPort func(string) bool) (*url.URL, error) {
	if !domain.ValidSourceURL(raw) {
		return nil, &application.FetchError{Reason: reasonNotWeb}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, &application.FetchError{Reason: reasonNotWeb}
	}
	if !allowPort(u.Port()) {
		return nil, &application.FetchError{Reason: reasonPort}
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		if !allowAddr(ip) {
			return nil, &application.FetchError{Reason: reasonInternal}
		}
	} else if internalName(host) {
		return nil, &application.FetchError{Reason: reasonInternal}
	}
	u.Fragment = ""
	return u, nil
}
