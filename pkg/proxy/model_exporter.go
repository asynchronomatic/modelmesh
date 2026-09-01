package proxy

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/ollama/ollama/api"

	"modelmesh/pkg/log"

	"modelmesh/pkg/core"
)

type ModelRoute struct {
	Name       string // Model Alias
	Model      string // Actual Model name
	Properties api.ListModelResponse
	Process    api.ProcessModelResponse
	Servers    map[string]string
}

type ModelExporter struct {
	providers    []core.Provider
	lock         sync.Mutex
	ActiveModels map[string]*ModelRoute
}

func (e *ModelExporter) List() []ModelRoute {
	e.lock.Lock()
	defer e.lock.Unlock()

	models := make([]ModelRoute, 0)

	for _, v := range e.ActiveModels {
		models = append(models, *v)
	}
	return models
}

func (e *ModelExporter) Refresh() {
	newTable := make(map[string]*ModelRoute)

	for _, provider := range e.providers {
		u, err := url.Parse(provider.BaseURL)
		if err != nil {
			log.Warnf("Failed to parse provider URL: %v", err)
			continue
		}

		client := api.NewClient(u, http.DefaultClient)
		ctx := context.Background()

		running, err := client.ListRunning(ctx) // GET /api/ps
		if err != nil {
			log.Warnf("Failed to list running models: %v", err)
			continue
		}

		for _, m := range running.Models {
			cfg, ok := newTable[m.Name]
			if !ok {
				cfg = &ModelRoute{
					Servers: make(map[string]string),
				}
			}
			cfg.Name = m.Name
			cfg.Model = m.Name
			cfg.Process = m
			//cfg.Servers = append(cfg.Servers, provider.BaseURL)
			cfg.Servers[provider.BaseURL] = provider.BaseURL
			newTable[m.Name] = cfg
		}

		available, err := client.List(ctx) // GET /api/tags
		if err != nil {
			log.Fatalf("Failed to list available models: %v", err)
		}

		for _, m := range available.Models {
			cfg, ok := newTable[m.Name]
			if !ok {
				continue
			}
			cfg.Properties = m
			newTable[m.Name] = cfg

			log.Infof("Model: %s -> %+v\n", m.Name, cfg.Servers)
		}
	}

	e.lock.Lock()
	defer e.lock.Unlock()
	e.ActiveModels = newTable
}

func NewModelExporter(providers []core.Provider) *ModelExporter {
	return &ModelExporter{
		providers:    providers,
		ActiveModels: make(map[string]*ModelRoute),
	}
}
