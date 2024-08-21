package config

type WorkerConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

func (cfg *WorkerConfig) Validate() error {
	return nil
}
