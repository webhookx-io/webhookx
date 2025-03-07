package config

import (
	"errors"
	"fmt"
	"slices"
)

type ProxyResponse struct {
	Code        uint   `yaml:"code" default:"200"`
	ContentType string `yaml:"contentType" default:"application/json"`
	Body        string `yaml:"body" default:"{\"message\": \"OK\"}"`
}

type QueueType string

const (
	QueueTypeOff   QueueType = "off"
	QueueTypeRedis QueueType = "redis"
)

type Queue struct {
	Type  QueueType   `yaml:"type" default:"redis"`
	Redis RedisConfig `yaml:"redis"`
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
	Listen             string        `yaml:"listen"`
	TLS                TLS           `yaml:"tls"`
	TimeoutRead        int64         `yaml:"timeout_read" default:"10" envconfig:"TIMEOUT_READ"`
	TimeoutWrite       int64         `yaml:"timeout_write" default:"10" envconfig:"TIMEOUT_WRITE"`
	MaxRequestBodySize int64         `yaml:"max_request_body_size" default:"1048576" envconfig:"MAX_REQUEST_BODY_SIZE"`
	Response           ProxyResponse `yaml:"response"`
	Queue              Queue         `yaml:"queue"`
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
