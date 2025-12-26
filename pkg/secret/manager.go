package secret

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/tidwall/gjson"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/pkg/secret/provider"
	"github.com/webhookx-io/webhookx/pkg/secret/provider/aws"
	"github.com/webhookx-io/webhookx/pkg/secret/provider/vault"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
	"go.uber.org/zap"
)

type SecretManager struct {
	opts      Options
	log       *zap.SugaredLogger
	providers map[string]provider.Provider
	cache     *expirable.LRU[string, string]
}

type Options struct {
	TTL time.Duration
}

func NewManager(opts Options) *SecretManager {
	manager := &SecretManager{
		opts:      opts,
		log:       zap.NewNop().Sugar(),
		providers: make(map[string]provider.Provider),
	}
	if opts.TTL > 0 {
		manager.cache = expirable.NewLRU[string, string](64, nil, opts.TTL)
	}
	return manager
}

func NewManagerFromConfig(cfg modules.SecretConfig) (*SecretManager, error) {
	manager := NewManager(Options{
		TTL: time.Second * time.Duration(cfg.TTL),
	})

	for _, name := range cfg.GetProviders() {
		var prov provider.Provider
		var err error
		switch name {
		case modules.ProviderAWS:
			prov, err = aws.NewProvider(cfg.GetProviderConfiguration(name))
		case modules.ProviderVault:
			prov, err = vault.NewProvider(cfg.GetProviderConfiguration(name))
		}
		if err != nil {
			return nil, err
		}
		manager.AddProvider(name, prov)
	}
	return manager, nil
}

func (p *SecretManager) WithLogger(log *zap.SugaredLogger) *SecretManager {
	p.log = log
	return p
}

func (p *SecretManager) AddProvider(name string, prov provider.Provider) {
	p.providers[name] = prov
}

func referenceKey(ref *reference.Reference) string {
	return ref.Provider + "/" + ref.Name
}

// ResolveReference returns resolved value of a reference
func (p *SecretManager) ResolveReference(ctx context.Context, ref *reference.Reference) (value string, err error) {
	p.log.Infof("resolving secret reference %s", ref)
	var cached bool
	if p.opts.TTL > 0 {
		value, cached = p.cache.Get(referenceKey(ref))
	}

	if !cached {
		prov := p.providers[ref.Provider]
		if prov == nil {
			return "", fmt.Errorf("failed to resolve reference value '%s': provider '%s' is not supported", ref.Reference, ref.Provider)
		}
		p.log.Debugf("fetching secret '%s' from %s", ref.Name, ref.Provider)
		value, err = prov.GetValue(ctx, ref.Name, ref.Properties)
		if err != nil {
			return "", fmt.Errorf("failed to resolve reference value '%s': %s", ref.Reference, err)
		}

		if p.opts.TTL > 0 {
			p.cache.Add(referenceKey(ref), value)
		}
	}

	if ref.JsonPointer != "" {
		if !gjson.Valid(value) {
			return "", fmt.Errorf("failed to resolve reference value '%s': value is not a valid JSON string", ref.Reference)
		}
		result := gjson.Get(value, ref.JsonPointer)
		if !result.Exists() {
			return "", fmt.Errorf("failed to resolve reference value '%s': no value for json path '%s'", ref.Reference, ref.JsonPointer)
		}
		value = result.String()
	}

	return value, nil
}
