package modules

import (
	"github.com/webhookx-io/webhookx/utils"
)

type AdminConfig struct {
	BaseConfig
	Listen         string `yaml:"listen" json:"listen" default:"127.0.0.1:9601"`
	DebugEndpoints bool   `yaml:"debug_endpoints" json:"debug_endpoints" envconfig:"DEBUG_ENDPOINTS"`
	TLS            TLS    `yaml:"tls" json:"tls"`
}

func (cfg AdminConfig) Validate() error {
	return nil
}

func (cfg AdminConfig) URL() string {
	if !cfg.IsEnabled() {
		return "disabled"
	}
	return utils.ListenAddrToURL(cfg.TLS.Enabled(), cfg.Listen)
}

func (cfg AdminConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}

type TLS struct {
	Cert string `yaml:"cert" json:"cert"`
	Key  string `yaml:"key" json:"key"`
}

func (cfg TLS) Enabled() bool {
	return cfg.Cert != "" && cfg.Key != ""
}
