package netpolicy

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"testing"
)

func TestPublicPolicyRejectsSpecialUseAddresses(t *testing.T) {
	for _, raw := range []string{
		"http://localhost/mod.zip",
		"http://sub.localhost/mod.zip",
		"http://127.0.0.1/mod.zip",
		"http://10.0.0.1/mod.zip",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.1/mod.zip",
		"http://[::1]/mod.zip",
		"http://[fd00::1]/mod.zip",
		"http://[fe80::1]/mod.zip",
	} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := Public().ValidateURLSyntax(u); !errors.Is(err, ErrDisallowedAddress) {
			t.Errorf("ValidateURLSyntax(%q) error = %v", raw, err)
		}
	}
}

func TestPublicPolicyRejectsMixedDNSAnswers(t *testing.T) {
	policy := Public()
	policy.Resolver = staticResolver{addresses: map[string][]netip.Addr{
		"downloads.example": {
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("127.0.0.1"),
		},
	}}
	u, _ := url.Parse("https://downloads.example/mod.zip")
	if err := policy.ValidateURL(context.Background(), u); !errors.Is(err, ErrDisallowedAddress) {
		t.Fatalf("ValidateURL() error = %v", err)
	}
}

type staticResolver struct {
	addresses map[string][]netip.Addr
}

func (r staticResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return r.addresses[host], nil
}
