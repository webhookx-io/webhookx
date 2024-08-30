package config

type WorkerDeliverer struct {
	Timeout int64 `yaml:"timeout" default:"60000"`
}

type WorkerConfig struct {
	Enabled   bool            `yaml:"enabled" default:"false"`
	Deliverer WorkerDeliverer `yaml:"deliverer"`
}

func (cfg *WorkerConfig) Validate() error {
	return nil
}
