package config

type KafkaConfig struct {
	Topic   string   `yaml:"topic" json:"topic" default:"webhookx"`
	Address []string `yaml:"address" json:"address"`
}
