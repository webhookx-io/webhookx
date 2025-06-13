package config

import (
	"errors"
	"fmt"
	"slices"
)

type ProxyResponse struct {
	Code        uint   `yaml:"code" json:"code" default:"200"`
	ContentType string `yaml:"content_type" json:"content_type" default:"application/json" envconfig:"CONTENT_TYPE"`
	Body        string `yaml:"body" json:"body" default:"{\"message\": \"OK\"}"`
}

type QueueType string

const (
	QueueTypeOff   QueueType = "off"
	QueueTypeRedis QueueType = "redis"
)

type Queue struct {
	Type  QueueType   `yaml:"type" json:"type" default:"redis"`
	Redis RedisConfig `yaml:"redis" json:"redis"`
}

func (cfg Queue) Validate() error {
	if !slices.Contains([]QueueType{QueueTypeRedis, QueueTypeOff}, cfg.Type) {
		return fmt.Errorf("unknown type: %s", cfg.Type)
	}
	if cfg.Type == QueueTypeRedis {
		if err := cfg.Redis.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ProxyConfig struct {
	Listen             string        `yaml:"listen" json:"listen"`
	TLS                TLS           `yaml:"tls" json:"tls"`
	TimeoutRead        int64         `yaml:"timeout_read" json:"timeout_read" default:"10" envconfig:"TIMEOUT_READ"`
	TimeoutWrite       int64         `yaml:"timeout_write" json:"timeout_write" default:"10" envconfig:"TIMEOUT_WRITE"`
	MaxRequestBodySize int64         `yaml:"max_request_body_size" json:"max_request_body_size" default:"1048576" envconfig:"MAX_REQUEST_BODY_SIZE"`
	Response           ProxyResponse `yaml:"response" json:"response"`
	Queue              Queue         `yaml:"queue" json:"queue"`
}

func (cfg ProxyConfig) Validate() error {
	if cfg.MaxRequestBodySize < 0 {
		return errors.New("max_request_body_size cannot be negative value")
	}
	if cfg.TimeoutRead < 0 {
		return errors.New("timeout_read cannot be negative value")
	}
	if cfg.TimeoutWrite < 0 {
		return errors.New("timeout_write cannot be negative value")
	}
	if err := cfg.Queue.Validate(); err != nil {
		return errors.New("invalid queue: " + err.Error())
	}
	return nil
}

func (cfg ProxyConfig) IsEnabled() bool {
	if cfg.Listen == "" || cfg.Listen == "off" {
		return false
	}
	return true
}
