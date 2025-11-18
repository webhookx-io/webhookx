package secret

import (
	"context"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/webhookx-io/webhookx/pkg/secret/provider"
	"github.com/webhookx-io/webhookx/pkg/secret/provider/aws"
	"github.com/webhookx-io/webhookx/pkg/secret/provider/vault"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
	"go.uber.org/zap"
)

var (
	ErrUnsupportedProvider  = errors.New("unsupported provider")
	ErrInvalidJson          = errors.New("value is not a valid json")
	ErrJsonPropertyNotFound = errors.New("json property not found")
)

type ProviderType string

const (
	AwsProviderType   ProviderType = "aws"
	VaultProviderType ProviderType = "vault"
)

type Manager struct {
	log       *zap.SugaredLogger
	providers map[string]provider.Provider
}

func NewManager(log *zap.SugaredLogger, providers map[string]map[string]interface{}) *Manager {
	manager := &Manager{
		log:       log,
		providers: make(map[string]provider.Provider),
	}
	for k, v := range providers {
		err := manager.registerProvider(k, v)
		if err != nil {
			panic(err) // todo
		}
	}
	return manager
}

func (p *Manager) registerProvider(name string, cfg map[string]interface{}) error {
	switch ProviderType(name) {
	case AwsProviderType:
		provider, err := aws.NewProvider(cfg)
		if err != nil {
			return err
		}
		p.providers[name] = provider
	case VaultProviderType:
		provider, err := vault.NewProvider(cfg)
		if err != nil {
			return err
		}
		p.providers[name] = provider
	default:
		return errors.New("unknown provider " + name)
	}
	return nil
}

func (p *Manager) getProvider(name string) provider.Provider {
	return p.providers[name]
}

// ResolveReference returns resolved value of a reference
func (p *Manager) ResolveReference(ctx context.Context, ref *reference.Reference) (string, error) {
	provider := p.getProvider(ref.Provider)
	if provider == nil {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedProvider, ref.Reference)
	}

	value, err := provider.GetValue(ctx, ref.Name, ref.Properties)
	if err != nil {
		return "", fmt.Errorf("failed to resolve value of reference '%s': %s", ref.Reference, err)
	}

	if ref.JsonPointer != "" {
		if !gjson.Valid(value) {
			return "", ErrInvalidJson
		}
		result := gjson.Get(value, ref.JsonPointer)
		if !result.Exists() {
			return "", fmt.Errorf("%w: %s", ErrJsonPropertyNotFound, ref.JsonPointer)
		}
		value = result.String()
	}

	p.log.Warnf("resolved %s to '%s'", ref.Reference, value)
	fmt.Printf("resolved %s to '%s'\n", ref.Reference, value)

	return value, nil
}
