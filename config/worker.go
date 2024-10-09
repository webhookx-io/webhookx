package config

type WorkerDeliverer struct {
	Timeout int64 `yaml:"timeout" default:"60000"`
}

type Pool struct {
	Size        uint32 `yaml:"size" default:"10000"`
	Concurrency uint32 `yaml:"concurrency"`
}

type WorkerConfig struct {
	Enabled   bool            `yaml:"enabled" default:"false"`
	Deliverer WorkerDeliverer `yaml:"deliverer"`
	Pool      Pool            `yaml:"pool"`
}

func (cfg *WorkerConfig) Validate() error {
	return nil
}
