package deliverer

import (
	"net/netip"
	"strings"
)

var presets = map[string][]string{
	"@private": {
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	},
	"@loopback": {
		"127.0.0.0/8",
		"::1/128",
	},
	"@linklocal": {
		"169.254.0.0/16",
		"fe80::/10",
	},
	"@reserved": {
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"fc00::/7",
	},
	"@default": {
		"@private",
		"@loopback",
		"@linklocal",
		"@reserved",
	},
}

type AclOptions struct {
	Rules []string
}

type ACL struct {
	IP     []netip.Addr
	CIDR   []netip.Prefix
	Domain []Domain
}

func expandPreset(rules []string) []string {
	var expanded []string
	for _, r := range rules {
		if set, ok := presets[r]; ok {
			expanded = append(expanded, expandPreset(set)...)
		} else {
			expanded = append(expanded, r)
		}
	}
	return expanded
}

func NewACL(opts AclOptions) *ACL {
	acl := &ACL{}

	rules := expandPreset(opts.Rules)
	for _, rule := range rules {
		if addr, err := netip.ParseAddr(rule); err == nil {
			acl.IP = append(acl.IP, addr)
			continue
		}
		if cidr, err := netip.ParsePrefix(rule); err == nil {
			acl.CIDR = append(acl.CIDR, cidr)
			continue
		}
		acl.Domain = append(acl.Domain, Domain(rule))
	}
	return acl
}

func (acl *ACL) Allow(host string, addr netip.Addr) bool {
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	if len(acl.IP) > 0 {
		for _, ip := range acl.IP {
			if ip == addr {
				return false
			}
		}
	}
	if len(acl.CIDR) > 0 {
		for _, cidr := range acl.CIDR {
			if cidr.Contains(addr) {
				return false
			}
		}
	}
	if len(acl.Domain) > 0 {
		for _, domain := range acl.Domain {
			if domain.Match(host) {
				return false
			}
		}
	}

	return true
}

type Domain string

func (d Domain) Match(host string) bool {
	if host == "" {
		return false
	}

	pattern := strings.ToLower(string(d))
	host = strings.ToLower(host)

	// exact match
	if !strings.HasPrefix(pattern, "*.") {
		return pattern == host
	}
	// wildcard match
	suffix := pattern[1:]
	return strings.HasSuffix(host, suffix)
}
