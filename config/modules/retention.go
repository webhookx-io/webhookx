package modules

type RetentionConfig struct {
	BaseConfig

	Enabled  bool   `yaml:"enabled" json:"enabled" default:"false"`
	Events   uint32 `yaml:"events" json:"events" default:"0"`
	Attempts uint32 `yaml:"attempts" json:"attempts" default:"0"`
}

func (cfg RetentionConfig) Validate() error {
	return nil
}
