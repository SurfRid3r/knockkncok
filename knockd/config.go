
package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Iface       string   `toml:"iface"`
	AllowPorts  []int    `toml:"allow_ports"`
	BaseTTLMin  int      `toml:"base_ttl_min"`
	MaxTTLMin   int      `toml:"max_ttl_min"`
	DbFile      string   `toml:"db_file"`
	Key         string   `toml:"key"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, err
	}

	return &config, nil
}
