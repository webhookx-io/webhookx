package modules

import (
	"fmt"
	"slices"

	"github.com/webhookx-io/webhookx/pkg/secret/provider/vault"
	"github.com/webhookx-io/webhookx/utils"
)

type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderVault Provider = "vault"
)

type SecretConfig struct {
	BaseConfig
	Providers []Provider    `json:"providers" yaml:"providers" default:"[\"@default\"]"`
	TTL       uint32        `json:"ttl" yaml:"ttl"`
	Aws       AwsProvider   `json:"aws" yaml:"aws"`
	Vault     VaultProvider `json:"vault" yaml:"vault"`
}

func (cfg *SecretConfig) Validate() error {
	for _, name := range cfg.Providers {
		if !slices.Contains([]Provider{"@default", ProviderAWS, ProviderVault}, name) {
			return fmt.Errorf("invalid provider: %s", name)
		}
	}
	if err := cfg.Aws.Validate(); err != nil {
		return err
	}
	if err := cfg.Vault.Validate(); err != nil {
		return err
	}
	return nil
}

func (cfg *SecretConfig) Enabled() bool {
	return len(cfg.Providers) > 0
}

func (cfg *SecretConfig) GetProviders() []Provider {
	names := make([]Provider, 0)
	for _, p := range cfg.Providers {
		if p == "@default" {
			names = append(names, ProviderAWS, ProviderVault)
		} else {
			names = append(names, p)
		}
	}
	return names
}

func (cfg *SecretConfig) GetProviderConfiguration(name string) map[string]interface{} {
	switch Provider(name) {
	case ProviderAWS:
		return utils.Must(utils.StructToMap(cfg.Aws))
	case ProviderVault:
		return utils.Must(utils.StructToMap(cfg.Vault))
	default:
		return nil
	}
}

type AwsProvider struct {
	Region string `json:"region" yaml:"region"`
	URL    string `json:"url" yaml:"url"`
}

func (cfg *AwsProvider) Validate() error {
	return nil
}

type VaultProvider struct {
	Address    string      `json:"address" yaml:"address" default:"http://127.0.0.1:8200"`
	MountPath  string      `json:"mount_path" yaml:"mount_path" default:"secret" split_words:"true"`
	Namespace  string      `json:"namespace" yaml:"namespace"`
	AuthMethod string      `json:"auth_method" yaml:"auth_method" default:"token" split_words:"true"`
	AuthN      vault.AuthN `json:"authn" yaml:"authn"`
}

func (cfg *VaultProvider) Validate() error {
	if !slices.Contains([]string{"token", "approle", "kubernetes"}, cfg.AuthMethod) {
		return fmt.Errorf("invalid auth_method: %s", cfg.AuthMethod)
	}
	return nil
}
