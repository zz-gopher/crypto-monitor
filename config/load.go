package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load 从指定路径读取 config.yaml 并解析为 Root
func Load(path string) (*Root, error) {
	if path == "" {
		return nil, errors.New("config path is empty")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get abs path: %w", err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", abs, err)
	}

	var cfg Root
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal %s: %w", abs, err)
	}

	return &cfg, nil
}
