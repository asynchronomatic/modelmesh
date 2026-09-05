package core

import (
	"os"

	"github.com/goccy/go-yaml"

	"github.com/asynchronomatic/speakeasy/pkg/log"
)

var DefaultRelayPort = 4001
var DefaultAdminPort = 4002
var DefaultProxyListen = ":4080"

type ModelConfig struct {
	Model        string
	Private      bool
	Capabilities []string
	Tools        []string // any builtin tools
}

type Provider struct {
	ID        string
	Type      string `yaml:"type"`     // Provider name (ollama, openai, etc)
	BaseURL   string `yaml:"base_url"` // BaseURL contains the API endpoint for the given provider
	Token     string `yaml:"token"`    // Token or access key used to access the api
	Private   bool   `yaml:"private"`  // Private indicates that this provider should not be exposed to the mesh
	Discovery string `yaml:"model_discovery"`
	Models    []ModelConfig
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

type AdminConfig struct {
	Address       string `yaml:"address"`
	Secret        string `yaml:"secret"`
	AdminPort     int    `yaml:"admin_port"`
	RelayPort     int    `yaml:"relay_port"`
	PublicAddress string `yaml:"public_address"`
}

type MeshConfig struct {
	Name          string `yaml:"name"`
	Address       string `yaml:"address"`
	Secret        string `yaml:"secret"`
	MeshId        string `yaml:"mesh_id"`
	PublicAddress string `yaml:"public_address"`
	Port          int    `yaml:"port"`
	ForcePrivate  bool   `yaml:"force_private"`
	MDNSEnabled   bool   `yaml:"mdns_enabled"`
}

type Config struct {
	Proxy struct {
		Listen string `yaml:"listen"`
	} `yaml:"proxy"`
	Admin     AdminConfig `yaml:"admin"`
	Mesh      MeshConfig  `yaml:"mesh"`
	Providers []Provider  `yaml:"providers"`
}

func applyConfigDefaults(config *Config) {
	if config.Proxy.Listen == "" {
		config.Proxy.Listen = DefaultProxyListen
	}
	if config.Admin.RelayPort <= 0 {
		config.Admin.RelayPort = DefaultRelayPort
	}
	if config.Mesh.Port <= 0 {
		config.Mesh.Port = 0
	}
	if config.Admin.AdminPort <= 0 {
		config.Admin.AdminPort = DefaultAdminPort
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

	if config.Mesh.Address == "" {
		log.Fatalf("Mesh.Address must be set in config.yaml")
	}

	return config
}
