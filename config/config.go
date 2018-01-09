package config

import (
	"github.com/go-redis/redis"
)

type Config struct {
	Listen        string
	Redis         redis.UniversalOptions `yaml:"redis"`
	SupervisorRPC string                 `yaml:"supervisor-rpc"`
}
