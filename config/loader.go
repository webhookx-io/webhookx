package config

import (
	"strings"

	"github.com/webhookx-io/webhookx/config/providers"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/secret"
)

// Loader is configuration loader
type Loader struct {
	cfg         *Config
	envPrefix   string
	filename    string
	fileContent []byte
	manager     *secret.SecretManager
}

func NewLoader(cfg *Config) *Loader {
	return &Loader{cfg: cfg}
}

func (l *Loader) WithEnvPrefix(prefix string) *Loader {
	l.envPrefix = prefix
	return l
}

func (l *Loader) WithFilename(filename string) *Loader {
	l.filename = filename
	return l
}

func (l *Loader) WithFileContent(content []byte) *Loader {
	l.fileContent = content
	return l
}

func (l *Loader) load(module string, value any) (err error) {
	err = providers.NewYAMLProvider(l.filename, l.fileContent).
		WithKey(strings.ToLower(module)).
		WithManager(l.manager).
		Load(value)
	if err != nil {
		return err
	}

	if l.envPrefix != "" {
		envPrefix := l.envPrefix
		if module != "" {
			envPrefix = l.envPrefix + "_" + module
		}
		err = providers.NewEnvProvider(envPrefix).
			WithManager(l.manager).
			Load(value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) Load() error {
	cfg := l.cfg

	if license.GetLicenser().Allow("secret") {
		if err := l.load("SECRET", &cfg.Secret); err != nil {
			return err
		}
		if cfg.Secret.Enabled() {
			secretManager, err := secret.NewManagerFromConfig(cfg.Secret)
			if err != nil {
				return err
			}
			l.manager = secretManager
			if err := l.load("LOG", &cfg.Log); err != nil {
				return err
			}
			logger, err := log.NewZapLogger(&cfg.Log)
			if err != nil {
				return err
			}
			secretManager.WithLogger(logger.Named("core"))
		}
	}

	if err := l.load("", cfg); err != nil {
		return err
	}

	return cfg.PostProcess()
}

func Load(filename string, cfg *Config) error {
	return NewLoader(cfg).WithEnvPrefix("WEBHOOKX").WithFilename(filename).Load()
}
