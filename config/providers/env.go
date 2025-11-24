package providers

import (
	"context"

	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"github.com/webhookx-io/webhookx/pkg/secret"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
)

type EnvProvider struct {
	prefix  string
	manager *secret.SecretManager
}

func (p *EnvProvider) WithManager(manager *secret.SecretManager) *EnvProvider {
	p.manager = manager
	return p
}

func (p *EnvProvider) Load(cfg any) error {
	var reader = envconfig.EnvironmentReader
	if p.manager != nil {
		reader = func(key string) (string, bool, error) {
			value, ok, _ := envconfig.EnvironmentReader.Read(key)
			if ok && reference.IsReference(value) {
				ref, err := reference.Parse(value)
				if err != nil {
					return "", false, err
				}
				resolved, err := p.manager.ResolveReference(context.TODO(), ref)
				if err != nil {
					return "", false, err
				}
				value = resolved
			}
			return value, ok, nil
		}
	}

	return envconfig.ProcessWithReader(p.prefix, cfg, reader)
}

func NewEnvProvider(prefix string) *EnvProvider {
	return &EnvProvider{prefix: prefix}
}
