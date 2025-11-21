package config

import (
	"github.com/webhookx-io/webhookx/config/core"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/config/providers"
	"github.com/webhookx-io/webhookx/pkg/secret"
	"go.uber.org/zap"
)

type Loader struct {
	cfg         core.Config
	filename    string
	fileContent []byte
}

func NewLoader(cfg core.Config) *Loader {
	return &Loader{cfg: cfg}
}

func (l *Loader) WithFilename(filename string) *Loader {
	l.filename = filename
	return l
}

func (l *Loader) WithFileContent(content []byte) *Loader {
	l.fileContent = content
	return l
}

func (l *Loader) Load() error {
	var secretCfg = l.cfg.(interface{ GetSecret() *modules.SecretConfig }).GetSecret()

	if err := providers.NewYAMLProvider(l.filename, l.fileContent).WithKey("secret").Load(secretCfg); err != nil {
		return err
	}
	if err := providers.NewEnvProvider("WEBHOOKX_SECRET").Load(secretCfg); err != nil {
		return err
	}

	var manager *secret.Manager
	if secretCfg.Enabled() {
		providers := make(map[string]map[string]interface{})
		for _, p := range secretCfg.GetProviders() {
			name := string(p)
			providers[name] = secretCfg.GetProviderConfiguration(name)
		}
		var err error
		manager, err = secret.NewManager(zap.S(), providers)
		if err != nil {
			return err
		}
	}

	list := make([]providers.ConfigProvider, 0)
	list = append(list, providers.NewYAMLProvider(l.filename, l.fileContent).WithManager(manager))
	list = append(list, providers.NewEnvProvider("WEBHOOKX").WithManager(manager))

	for _, provider := range list {
		err := provider.Load(l.cfg)
		if err != nil {
			return err
		}
	}

	return l.cfg.PostProcess()
	// todo l.cfg.Validate() ?
}

func Load(filename string, cfg core.Config) error {
	return NewLoader(cfg).WithFilename(filename).Load()
}
