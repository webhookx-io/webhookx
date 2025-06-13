package config

type WorkerDeliverer struct {
	Timeout int64 `yaml:"timeout" json:"timeout" default:"60000"`
}

type Pool struct {
	Size        uint32 `yaml:"size" json:"size" default:"10000"`
	Concurrency uint32 `yaml:"concurrency" json:"concurrency"`
}

type WorkerConfig struct {
	Enabled   bool            `yaml:"enabled" json:"enabled" default:"false"`
	Deliverer WorkerDeliverer `yaml:"deliverer" json:"deliverer"`
	Pool      Pool            `yaml:"pool" json:"pool"`
}

func (cfg *WorkerConfig) Validate() error {
	return nil
}
