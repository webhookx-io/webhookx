package providers

import (
	"context"
	"os"

	"github.com/webhookx-io/webhookx/pkg/secret"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
	"gopkg.in/yaml.v3"
)

type YAMLProvider struct {
	filename string
	content  []byte
	key      string
	manager  *secret.SecretManager
}

func NewYAMLProvider(filename string, content []byte) *YAMLProvider {
	return &YAMLProvider{
		filename: filename,
		content:  content,
	}
}

func (p *YAMLProvider) WithManager(manager *secret.SecretManager) *YAMLProvider {
	p.manager = manager
	return p
}

func (p *YAMLProvider) WithKey(key string) *YAMLProvider {
	p.key = key
	return p
}

func rewriteScalarNode(dst *yaml.Node, val string) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(val), &doc); err == nil && len(doc.Content) > 0 {
		if scalar := doc.Content[0]; scalar.Kind == yaml.ScalarNode {
			dst.Tag = scalar.Tag
			dst.Style = scalar.Style
			dst.Value = scalar.Value
		}
		return
	}
	dst.Tag = "!!str"
	dst.Style = 0
	dst.Value = val
}

func resolveReference(n *yaml.Node, manager *secret.SecretManager) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if reference.IsReference(n.Value) {
			ref, err := reference.Parse(n.Value)
			if err != nil {
				return err
			}
			val, err := manager.ResolveReference(context.TODO(), ref)
			if err != nil {
				return err
			}
			rewriteScalarNode(n, val)
		}
	case yaml.MappingNode:
		for i := 0; i < len(n.Content); i += 2 {
			if err := resolveReference(n.Content[i+1], manager); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		if n.Alias != nil {
			if err := resolveReference(n.Alias, manager); err != nil {
				return err
			}
		}
	default:
		for _, c := range n.Content {
			if err := resolveReference(c, manager); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *YAMLProvider) Load(cfg any) error {
	if p.filename == "" && p.content == nil {
		return nil
	}

	if p.filename != "" {
		b, err := os.ReadFile(p.filename)
		if err != nil {
			return err
		}
		p.content = b
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(p.content, &doc); err != nil {
		return err
	}

	if p.key != "" && len(doc.Content) > 0 {
		if node := findYaml(doc.Content[0], p.key); node != nil {
			doc = *node
		}
	}

	if p.manager != nil {
		if err := resolveReference(&doc, p.manager); err != nil {
			return err
		}
	}

	return doc.Decode(cfg)
}

func findYaml(n *yaml.Node, key string) *yaml.Node {
	if n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i]
		if k.Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}
