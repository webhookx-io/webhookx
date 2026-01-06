package modules

import (
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
)

type WorkerDeliverer struct {
	Timeout        int64     `yaml:"timeout" json:"timeout" default:"60000"`
	ACL            ACLConfig `yaml:"acl" json:"acl"`
	Proxy          string    `yaml:"proxy" json:"proxy"`
	ProxyTLSCert   string    `yaml:"proxy_tls_cert" json:"proxy_tls_cert" envconfig:"PROXY_TLS_CERT"`
	ProxyTLSKey    string    `yaml:"proxy_tls_key" json:"proxy_tls_key" envconfig:"PROXY_TLS_KEY"`
	ProxyTLSCaCert string    `yaml:"proxy_tls_ca_cert" json:"proxy_tls_ca_cert" envconfig:"PROXY_TLS_CA_CERT"`
	ProxyTLSVerify bool      `yaml:"proxy_tls_verify" json:"proxy_tls_verify" envconfig:"PROXY_TLS_VERIFY"`
}

func (cfg *WorkerDeliverer) Validate() error {
	if cfg.Timeout < 0 {
		return fmt.Errorf("deliverer.timeout cannot be negative")
	}
	if err := cfg.ACL.Validate(); err != nil {
		return err
	}
	if cfg.Proxy != "" {
		u, err := url.Parse(cfg.Proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy url: %s", err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid proxy url: '%s'", cfg.Proxy)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("proxy schema must be http or https")
		}
	}

	return nil
}

type Pool struct {
	Size        uint32 `yaml:"size" json:"size" default:"10000"`
	Concurrency uint32 `yaml:"concurrency" json:"concurrency"`
}

type WorkerConfig struct {
	BaseConfig
	Enabled   bool            `yaml:"enabled" json:"enabled" default:"true"`
	Deliverer WorkerDeliverer `yaml:"deliverer" json:"deliverer"`
	Pool      Pool            `yaml:"pool" json:"pool"`
}

func (cfg *WorkerConfig) Status() string {
	if cfg.Enabled {
		return "on"
	}
	return "off"
}

type ACLConfig struct {
	Deny []string `yaml:"deny" json:"deny" default:"[\"@default\"]"`
}

func (acl *ACLConfig) Validate() error {
	for _, rule := range acl.Deny {
		if err := validateRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func validateRule(rule string) error {
	groups := []string{"@default", "@private", "@loopback", "@linklocal", "@reserved"}
	if slices.Contains(groups, rule) {
		return nil
	}
	if _, err := netip.ParseAddr(rule); err == nil {
		return nil
	}
	if _, err := netip.ParsePrefix(rule); err == nil {
		return nil
	}
	r := regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)+$`)
	if matched := r.MatchString(rule); matched {
		return nil
	}
	return fmt.Errorf("invalid rule '%s': requires IP, CIDR, hostname, or pre-configured name", rule)
}

func (cfg *WorkerConfig) Validate() error {
	if err := cfg.Deliverer.Validate(); err != nil {
		return err
	}
	return nil
}
