package deliverer

import (
	"net/netip"
	"testing"
)

func TestDomainMatch(t *testing.T) {
	cases := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"a.example.com", "*.example.com", true},
		{"a.b.example.com", "*.example.com", true},
		{"a.b.c.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"example.com", "example.com", true},
		{"EXAMPLE.COM", "example.com", true},
		{"example.com", "EXAMPLE.COM", true},
		{"a.b", "*.b", true},
		{"b", "*.b", false},
		{"a.b.c", "*.b.c", true},
		{"x.y.b.c", "*.b.c", true},
		{"", "example.com", false},
	}

	for _, tc := range cases {
		got := Domain(tc.pattern).Match(tc.host)
		if got != tc.want {
			t.Errorf("hostMatch(%s, %s) got %v, want %v", tc.pattern, tc.host, got, tc.want)
		}
	}
}

func TestAllow(t *testing.T) {
	tests := []struct {
		scenario string
		rules    []string
		hostname string
		ip       string
		allow    bool
	}{
		{
			scenario: "deny 127.0.0.1",
			rules:    []string{"@default"},
			ip:       "127.0.0.1",
			allow:    false,
		},
		{
			scenario: "deny ::1",
			rules:    []string{"@default"},
			ip:       "::1",
			allow:    false,
		},
		{
			scenario: "deny ip4",
			rules:    []string{"8.8.8.8"},
			ip:       "8.8.8.8",
			allow:    false,
		},
		{
			scenario: "deny ip6",
			rules:    []string{"2606:2800:220:1:248:1893:25c8:1946"},
			ip:       "2606:2800:220:1:248:1893:25c8:1946",
			allow:    false,
		},
		{
			scenario: "deny IPv4-mapped IPv6 address",
			rules:    []string{"@default"},
			ip:       "::ffff:127.0.0.1",
			allow:    false,
		},
		{
			scenario: "allow example.com",
			rules:    []string{"@default"},
			hostname: "example.com",
			ip:       "1.1.1.1",
			allow:    true,
		},
		{
			scenario: "deny subdomain",
			rules:    []string{"@default", "*.example.com"},
			hostname: "foo.example.com",
			ip:       "1.1.1.1",
			allow:    false,
		},
		{
			scenario: "allow root domain",
			rules:    []string{"@default", "*.example.com"},
			hostname: "example.com",
			ip:       "1.1.1.1",
			allow:    true,
		},
		{
			scenario: "deny punycode domian",
			rules:    []string{"@default", "xn--6qq79v.com"},
			hostname: "xn--6qq79v.com",
			ip:       "1.1.1.1",
			allow:    false,
		},
	}

	for _, test := range tests {
		acl := NewACL(AclOptions{Rules: test.rules})
		actual := acl.Allow(test.hostname, netip.MustParseAddr(test.ip))
		if actual != test.allow {
			t.Errorf("Allow(%v, %v) expected %v, got %v", test.hostname, test.ip, test.allow, actual)
		}
	}
}
