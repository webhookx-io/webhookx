package config

type AdminConfig struct {
	Listen         string `yaml:"listen"`
	DebugEndpoints bool   `yaml:"debug_endpoints" envconfig:"DEBUG_ENDPOINTS"`
	TLS            TLS    `yaml:"tls"`
}

func (cfg AdminConfig) Validate() error {
	return nil
}

func (cfg AdminConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}

type TLS struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

func (cfg TLS) Enabled() bool {
	return cfg.Cert != "" && cfg.Key != ""
}
