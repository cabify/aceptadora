package aceptadora

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

// YAML defines the aceptadora yaml config
// This enumerates the services aceptadora can run, their images, volumes to be mounted, ports to be mapped, and env configs
type YAML struct {
	Services map[string]Service `yaml:"services"`
}

// Service describes a service aceptadora can run
type Service struct {
	Image   string   `yaml:"image"`
	Network string   `yaml:"network"`
	Binds   []string `yaml:"binds"`
	Command []string `yaml:"command"`
	EnvFile []string `yaml:"env_file"`
	Ports   []string `yaml:"ports"`

	IgnoreLogs bool `yaml:"ignore_logs"`
}

// LoadYAML reads the aceptadora.yml config, expanding the env var references to their values.
func LoadYAML(filename string) (YAML, error) {
	cfg := YAML{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return cfg, fmt.Errorf("can't read file: %w", err)
	}
	// expand environment variables
	data = []byte(os.ExpandEnv(string(data)))
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("can't read yaml: %w", err)
	}
	return cfg, nil
}
