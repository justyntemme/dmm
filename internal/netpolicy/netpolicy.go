package netpolicy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

var ErrDisallowedAddress = errors.New("network address is not allowed")

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Policy struct {
	AllowPrivateNetworks bool
	Resolver             Resolver
	Dialer               Dialer
}

func Public() Policy {
	return Policy{}
}

func AllowPrivate() Policy {
	return Policy{AllowPrivateNetworks: true}
}

func (p Policy) ValidateURLSyntax(u *url.URL) error {
	if u == nil {
		return errors.New("URL is required")
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	if scheme != "http" && scheme != "https" {
		return errors.New("URL must use http or https")
	}
	if u.User != nil {
		return errors.New("URL credentials are not allowed")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return errors.New("URL must include a host")
	}
	if p.AllowPrivateNetworks {
		return nil
	}
	canonicalHost := strings.ToLower(strings.TrimSuffix(host, "."))
	if canonicalHost == "localhost" || strings.HasSuffix(canonicalHost, ".localhost") {
		return fmt.Errorf("%w: localhost", ErrDisallowedAddress)
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return validateAddress(addr)
	}
	return nil
}

func (p Policy) ValidateURL(ctx context.Context, u *url.URL) error {
	if err := p.ValidateURLSyntax(u); err != nil {
		return err
	}
	_, err := p.resolve(ctx, u.Hostname())
	return err
}

func (p Policy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split network address: %w", err)
	}
	addresses, err := p.resolve(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	var dialErrors []error
	for _, addr := range addresses {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		dialErrors = append(dialErrors, dialErr)
	}
	return nil, errors.Join(dialErrors...)
}

func (p Policy) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, errors.New("network host is required")
	}
	var addresses []netip.Addr
	if addr, err := netip.ParseAddr(host); err == nil {
		addresses = []netip.Addr{addr}
	} else {
		resolver := p.Resolver
		if resolver == nil {
			resolver = net.DefaultResolver
		}
		var err error
		addresses, err = resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, err)
		}
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve %s: no addresses", host)
	}
	if p.AllowPrivateNetworks {
		return addresses, nil
	}
	for _, addr := range addresses {
		if err := validateAddress(addr); err != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, err)
		}
	}
	return addresses, nil
}

var specialUsePrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

func validateAddress(addr netip.Addr) error {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() {
		return fmt.Errorf("%w: %s", ErrDisallowedAddress, addr)
	}
	for _, prefix := range specialUsePrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("%w: %s", ErrDisallowedAddress, addr)
		}
	}
	return nil
}
