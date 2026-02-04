package helper

import (
	"github.com/webhookx-io/webhookx/config"
)

type LoadConfigOptions struct {
	Envs       map[string]string
	File       string
	ExcludeEnv bool
}

func LoadConfig(opts LoadConfigOptions) (*config.Config, error) {
	cfg := config.New()

	loader := config.NewLoader(cfg).
		WithFilename(opts.File)

	if !opts.ExcludeEnv {
		loader = loader.
			WithEnvPrefix("WEBHOOKX").
			WithEnv(opts.Envs)
	}

	if err := loader.Load(); err != nil {
		return nil, err
	}

	return cfg, nil
}
