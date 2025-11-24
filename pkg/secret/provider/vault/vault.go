package vault

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/mitchellh/mapstructure"
	"github.com/webhookx-io/webhookx/pkg/secret/provider"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrSecretNoData   = errors.New("secret no data")
)

type VaultProvider struct {
	mountPath string
	client    *api.Client
	cfg       interface{}
}

func NewProvider(cfg map[string]interface{}) (provider.Provider, error) {
	config := api.DefaultConfig()
	config.Address = cfg["address"].(string)
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	method := cfg["auth_method"].(string)
	authn := cfg["authn"].(map[string]interface{})
	if err := setupClientAuth(client, method, authn); err != nil {
		return nil, err
	}

	p := &VaultProvider{
		mountPath: cfg["mount_path"].(string),
		client:    client,
		cfg:       cfg,
	}
	return p, nil
}

func setupClientAuth(client *api.Client, method string, cfg map[string]interface{}) error {
	var auth AuthN
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &auth,
	})
	if err != nil {
		return err
	}
	if err := decoder.Decode(cfg); err != nil {
		return err
	}
	switch method {
	case "token":
		client.SetToken(auth.Token.Token)
	case "approle":
		opts := make([]approle.LoginOption, 0)
		if auth.AppRole.ResponseWrapping {
			opts = append(opts, approle.WithWrappingToken())
		}
		appRoleAuth, err := approle.NewAppRoleAuth(
			auth.AppRole.RoleID,
			&approle.SecretID{FromString: auth.AppRole.SecretID},
			opts...,
		)
		if err != nil {
			return err
		}
		_, err = client.Auth().Login(context.TODO(), appRoleAuth)
		if err != nil {
			return err
		}
	case "kubernetes":
		opts := make([]kubernetes.LoginOption, 0)
		if auth.Kubernetes.TokenPath != "" {
			opts = append(opts, kubernetes.WithServiceAccountTokenPath(auth.Kubernetes.TokenPath))
		}
		auth, err := kubernetes.NewKubernetesAuth(
			auth.Kubernetes.Role,
			opts...,
		)
		if err != nil {
			return err
		}
		_, err = client.Auth().Login(context.TODO(), auth)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *VaultProvider) GetValue(ctx context.Context, key string, properties map[string]string) (string, error) {
	secret, err := p.client.KVv2(p.mountPath).Get(ctx, key)
	if err != nil {
		if errors.Is(err, api.ErrSecretNotFound) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	if secret == nil {
		return "", ErrSecretNotFound
	}
	if secret.Data == nil {
		return "", ErrSecretNoData
	}
	value, err := json.Marshal(secret.Data)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
