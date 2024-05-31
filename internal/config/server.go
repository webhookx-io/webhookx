package config

import (
	"fmt"
)

type ServerConfig struct {
	Port   int    `default:"8080"`
	Host   string `default:"127.0.0.1"`
	Daemon bool   `default:"false"`
}

func (cfg ServerConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}
