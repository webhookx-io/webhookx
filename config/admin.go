package config

type AdminConfig struct {
	Listen string `yaml:"listen"`
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
