package config

type StatusConfig struct {
	Listen         string `yaml:"listen" json:"listen" default:"127.0.0.1:8082"`
	DebugEndpoints bool   `yaml:"debug_endpoints" json:"debug_endpoints" default:"true" envconfig:"DEBUG_ENDPOINTS"`
}

func (cfg StatusConfig) Validate() error {
	return nil
}

func (cfg StatusConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}
