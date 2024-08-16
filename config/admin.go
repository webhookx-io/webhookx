package config

type AdminConfig struct {
	Listen string `yaml:"listen" default:"127.0.0.1:8080"`
}

func (cfg AdminConfig) Validate() error {
	return nil
}
