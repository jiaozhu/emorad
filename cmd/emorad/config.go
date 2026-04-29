package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type emoradConfig struct {
	Filter struct {
		CommonJarPrefixes []string `yaml:"common_jar_prefixes"`
		AppJarHints       []string `yaml:"app_jar_hints"`
		JarInclude        []string `yaml:"jar_include"`
	} `yaml:"filter"`
}

func loadConfig(configPath string) (*emoradConfig, string, error) {
	candidates := make([]string, 0, 3)
	if strings.TrimSpace(configPath) != "" {
		candidates = append(candidates, configPath)
	} else {
		candidates = append(candidates, ".emorad.yaml")
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			candidates = append(candidates, filepath.Join(home, ".emorad", "config.yaml"))
		}
	}

	for _, p := range candidates {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, "", err
		}
		var cfg emoradConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, "", err
		}
		return &cfg, p, nil
	}

	return nil, "", nil
}

