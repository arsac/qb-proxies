package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Feeds []Feed `yaml:"feeds"`
}

type Feed struct {
	Name           string            `yaml:"name"`
	Path           string            `yaml:"path"`
	Upstream       string            `yaml:"upstream"`
	Transformations []Transformation `yaml:"transformations"`
}

type Transformation struct {
	Field      string `yaml:"field"`
	Expression string `yaml:"expression"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	paths := make(map[string]bool)
	for i, feed := range c.Feeds {
		if feed.Name == "" {
			return fmt.Errorf("feed %d: name is required", i)
		}
		if feed.Path == "" {
			return fmt.Errorf("feed %d: path is required", i)
		}
		if feed.Upstream == "" {
			return fmt.Errorf("feed %d: upstream is required", i)
		}
		if paths[feed.Path] {
			return fmt.Errorf("feed %d: duplicate path %s", i, feed.Path)
		}
		paths[feed.Path] = true
	}
	return nil
}
