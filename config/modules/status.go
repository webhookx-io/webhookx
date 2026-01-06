package modules

import (
	"fmt"
	"net"

	"github.com/webhookx-io/webhookx/utils"
)

type StatusConfig struct {
	BaseConfig
	Listen         string `yaml:"listen" json:"listen" default:"127.0.0.1:9602"`
	DebugEndpoints bool   `yaml:"debug_endpoints" json:"debug_endpoints" default:"true" envconfig:"DEBUG_ENDPOINTS"`
}

func (cfg StatusConfig) Validate() error {
	if cfg.IsEnabled() {
		_, _, err := net.SplitHostPort(cfg.Listen)
		if err != nil {
			return fmt.Errorf("invalid listen '%s': %s", cfg.Listen, err)
		}
	}
	return nil
}

func (cfg StatusConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}

func (cfg StatusConfig) URL() string {
	if !cfg.IsEnabled() {
		return "disabled"
	}
	return utils.ListenAddrToURL(false, cfg.Listen)
}
