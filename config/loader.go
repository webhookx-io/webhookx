package config

import (
	"strings"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/config/providers"
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

func newSecretManager(cfg modules.SecretConfig) (*secret.SecretManager, error) {
	manager := secret.NewManager(secret.Options{
		TTL: time.Second * time.Duration(cfg.TTL),
	})
	for _, p := range cfg.GetProviders() {
		name := string(p)
		err := manager.RegisterProvider(name, cfg.GetProviderConfiguration(name))
		if err != nil {
			return nil, err
		}
	}
	return manager, nil
}

func (l *Loader) Load() error {
	cfg := l.cfg
	if err := l.load("SECRET", &cfg.Secret); err != nil {
		return err
	}

	if cfg.Secret.Enabled() {
		secretManager, err := newSecretManager(cfg.Secret)
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

	if err := l.load("", cfg); err != nil {
		return err
	}

	return cfg.PostProcess()
}

func Load(filename string, cfg *Config) error {
	return NewLoader(cfg).WithEnvPrefix("WEBHOOKX").WithFilename(filename).Load()
}
