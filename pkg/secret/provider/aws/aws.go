package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/webhookx-io/webhookx/pkg/secret/provider"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
)

type AwsProvider struct {
	cfg    interface{}
	client *secretsmanager.Client
}

func NewProvider(cfg map[string]interface{}) (provider.Provider, error) {
	opts := make([]func(*config.LoadOptions) error, 0)
	if region := cfg["region"].(string); region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	if url := cfg["url"].(string); url != "" {
		opts = append(opts, config.WithBaseEndpoint(url))
	}
	awsconfig, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	p := &AwsProvider{
		cfg: cfg,
	}
	p.client = secretsmanager.NewFromConfig(awsconfig, func(options *secretsmanager.Options) {})

	return p, nil
}

func (p *AwsProvider) GetValue(ctx context.Context, key string, properties map[string]string) (string, error) {
	result, err := p.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(key)})
	if err != nil {
		var awsErr *types.ResourceNotFoundException
		if errors.As(err, &awsErr) {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return *result.SecretString, nil
}
