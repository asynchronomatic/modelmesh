package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/exp/maps"

	"modelmesh/pkg/core"
	"modelmesh/pkg/jsonclient"
	"modelmesh/pkg/log"

	"github.com/ollama/ollama/api"
)

type ModelRouter struct {
	node       core.PeerNode
	providers  []core.Provider // configured providers for rescanning
	lock       sync.Mutex
	MeshModels map[string]ModelRoute // MeshModelswill be forwarded out
}

// ListExportedModels returns a list of models exported from our instance
func (e *ModelRouter) ListExportedModels() []ModelRoute {
	e.lock.Lock()
	defer e.lock.Unlock()
	local := make([]ModelRoute, 0)

	for name := range e.MeshModels {
		model := e.MeshModels[name]
		if model.isLocal() && !model.isPrivate() {
			local = append(local, model)
		}
	}
	return local
}

// ListMeshModels lists all models discovered (local & peer)
func (e *ModelRouter) ListMeshModels() []ModelRoute {
	e.lock.Lock()
	defer e.lock.Unlock()
	return maps.Values(e.MeshModels)
}

func (e *ModelRouter) modelsFromWhitelist(provider *core.Provider) map[string]ModelRoute {
	modelTable := make(map[string]ModelRoute)

	for _, m := range provider.Models {
		cfg, ok := modelTable[m.Model]
		if !ok {
			cfg = MakeRoute(m.Model, m.Model, m.Capabilities)
		}

		cfg.providers = append(cfg.providers, ModelProvider{
			Private:  m.Private,
			Provider: provider.Type,
			BaseURL:  provider.BaseURL,
			Token:    provider.Token,
		})
		modelTable[m.Model] = cfg
	}
	return modelTable
}

func (e *ModelRouter) ollamaFetchModels(provider *core.Provider) (map[string]ModelRoute, error) {

	models := make(map[string]ModelRoute)

	u, err := url.Parse(provider.BaseURL)
	if err != nil {
		return nil, err
	}

	client := api.NewClient(u, http.DefaultClient)
	ctx := context.Background()

	resp, err := client.List(ctx) // GET /api/tags
	if err != nil {
		return nil, fmt.Errorf("failed to list running models: %v", err)
	}

	properties := make(map[string]api.ListModelResponse)
	for k := range resp.Models {
		m := resp.Models[k]
		properties[m.Name] = m
	}

	if provider.Discovery == "whitelist" {
		models = e.modelsFromWhitelist(provider)
	} else {
		running, err := client.ListRunning(ctx) // GET /api/ps
		if err != nil {
			return nil, fmt.Errorf("failed to list running models: %v", err)
		}

		for _, m := range running.Models {
			cfg, ok := models[m.Model]
			if !ok {
				cfg = MakeRoute(m.Model, m.Model, []string{})
			}

			cfg.providers = append(cfg.providers, ModelProvider{
				Private:  provider.Private,
				Provider: provider.Type,
				BaseURL:  provider.BaseURL,
				Token:    provider.Token,
			})
			models[m.Model] = cfg
		}
	}

	for k := range models {
		p, ok := properties[k]
		if !ok {
			delete(models, k)
			continue
		}

		m := models[k]
		for _, c := range p.Capabilities {
			m.Capabilities = append(m.Capabilities, string(c))
		}

		m.ContextLength = p.Details.ContextLength

	}

	return models, nil
}

func (e *ModelRouter) openaiFetchModels(provider *core.Provider) (map[string]ModelRoute, error) {
	client := jsonclient.NewClient(provider.BaseURL, provider.Token)

	resp := OpenaiModelList{}
	err := client.Get("/models", &resp)
	if err != nil {
		return nil, err
	}

	models := make(map[string]ModelRoute)

	for _, m := range resp.Data {
		cfg, ok := models[m.ID]
		if !ok {
			cfg = MakeRoute(m.ID, m.ID, []string{})
		}
		cfg.providers = append(cfg.providers, ModelProvider{
			Private:  false,
			Provider: provider.Type,
			BaseURL:  provider.BaseURL,
			Token:    provider.Token,
		})
		models[m.ID] = cfg
	}
	return models, nil
}

func (e *ModelRouter) testFetchModels(provider *core.Provider) (map[string]ModelRoute, error) {
	return e.modelsFromWhitelist(provider), nil
}

func (e *ModelRouter) ListModels() []string {
	models := make([]string, 0)
	for _, m := range e.MeshModels {
		models = append(models, m.Name)
	}
	return models
}

func (e *ModelRouter) AddPeerModels(node core.PeerNode, models map[string]ModelRoute) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for name := range models {
		route, ok := e.MeshModels[node.ID]
		if !ok {
			route = models[name]
		}

		route.AddPeer(node)
		e.MeshModels[name] = route
	}
}

func (e *ModelRouter) RemovePeer(node core.PeerNode) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, model := range e.MeshModels {
		model.RemovePeer(node)
	}
}

func (e *ModelRouter) Refresh() {
	for _, provider := range e.providers {
		var err error
		var routes map[string]ModelRoute

		switch provider.Type {
		case "ollama":
			routes, err = e.ollamaFetchModels(&provider)
			if err != nil {
				log.Warnf("%s", err)
				continue
			}
		case "openai":
			routes, err = e.openaiFetchModels(&provider)
			if err != nil {
				log.Warnf("%s", err)
				continue
			}

		case "test":
			routes, err = e.testFetchModels(&provider)
			if err != nil {
				log.Warnf("%s", err)
				continue
			}

		default:
			log.Warnf("Unsupported provider type: %s", provider.Type)
			continue
		}

		e.AddPeerModels(e.node, routes)
	}

	e.lock.Lock()
	defer e.lock.Unlock()
}

func (e *ModelRouter) GetModelRoute(model string) *ModelRoute {
	e.lock.Lock()
	defer e.lock.Unlock()
	if route, ok := e.MeshModels[model]; ok {
		return &route
	}
	return nil
}

func NewModelDiscovery(node core.PeerNode, providers []core.Provider) *ModelRouter {
	return &ModelRouter{
		node:       node,
		providers:  providers,
		MeshModels: make(map[string]ModelRoute),
	}
}
