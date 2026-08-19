package modules

import (
	"fmt"
	"net/url"

	"github.com/webhookx-io/webhookx/utils"
)

type DashboardConfig struct {
	BaseConfig
	Listen       string `yaml:"listen" json:"listen" default:"127.0.0.1:9605"`
	TLS          TLS    `yaml:"tls" json:"tls"`
	AdminAddress string `yaml:"admin_address" json:"admin_address" default:"http://127.0.0.1:9601" envconfig:"ADMIN_ADDRESS"`
}

func (cfg DashboardConfig) Validate() error {
	if err := validateDashboardAddress("admin_address", cfg.AdminAddress); err != nil {
		return err
	}
	return nil
}

func validateDashboardAddress(name string, address string) error {
	if address == "" {
		return fmt.Errorf("%s is required", name)
	}
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("invalid %s %q: %s", name, address, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid %s %q: schema must be http or https", name, address)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid %s %q: host is required", name, address)
	}
	return nil
}

func (cfg DashboardConfig) URL() string {
	if !cfg.IsEnabled() {
		return "disabled"
	}
	return utils.ListenAddrToURL(cfg.TLS.Enabled(), cfg.Listen)
}

func (cfg DashboardConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}
