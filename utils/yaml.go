package utils

import "gopkg.in/yaml.v3"

func FindYaml(n *yaml.Node, key string) *yaml.Node {
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
