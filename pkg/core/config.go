package core

import (
	"os"

	"github.com/goccy/go-yaml"

	"modelmesh/pkg/log"
)

var DefaultRelayPort = 4001
var DefaultAdminPort = 4002
var DefaultProxyListen = ":8080"

type Provider struct {
	ID        string
	Type      string
	BaseURL   string `yaml:"base_url"`
	Discovery string `yaml:"model_discovery"`
	Models    []struct {
		ID string
	}
}

type EnabledModel struct {
	Name            string
	Strategy        string
	SessionAffinity string
	Targets         []struct {
		Model   string
		BaseURL string `yaml:"base_url"`
		Weight  int
	}
}

type MeshConfig struct {
	Name         string `yaml:"name"`
	AdminAddress string `yaml:"admin_address"`
	AdminKey     string `yaml:"admin_secret"`
	AdminPort    int    `yaml:"admin_port"`
	RelayPort    int    `yaml:"relay_port"`

	PublicAddress string `yaml:"public_address"`

	AppPort      int  `yaml:"app_port"`
	ForcePrivate bool `yaml:"force_private"`
	MDNSEnabled  bool `yaml:"mdns_enabled"`
}

type Config struct {
	Proxy struct {
		Listen string `yaml:"listen"`
	}
	Mesh      MeshConfig
	Providers []Provider
}

func applyConfigDefaults(config *Config) {
	if config.Proxy.Listen == "" {
		config.Proxy.Listen = DefaultProxyListen
	}
	if config.Mesh.RelayPort <= 0 {
		config.Mesh.RelayPort = DefaultRelayPort
	}
	if config.Mesh.AppPort <= 0 {
		config.Mesh.AppPort = 0
	}
	if config.Mesh.AdminPort <= 0 {
		config.Mesh.AdminPort = DefaultAdminPort
	}
	if config.Mesh.Name == "" {
		config.Mesh.Name, _ = os.Hostname()
	}
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	applyConfigDefaults(config)
	return config, nil
}

func MustLoadConfig() *Config {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Could not load config.yaml  (Err:%v)\n", err)
	}

	if config.Mesh.AdminAddress == "" {
		log.Fatalf("Mesh.Address must be set in config.yaml")
	}

	return config
}
