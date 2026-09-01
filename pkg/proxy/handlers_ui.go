package proxy

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ollama/ollama/types/model"

	"modelmesh/pkg/log"
	"modelmesh/web"
)

type UIModelProvider struct {
	Name   string `json:"name"`
	PeerID string `json:"peer_id"`
	Self   bool   `json:"self"`
}

type UIModel struct {
	Name         string            `json:"name"`
	Model        string            `json:"model"`
	Size         int64             `json:"size"`
	Digest       string            `json:"digest"`
	Family       string            `json:"family"`
	Families     []string          `json:"families,omitempty"`
	Params       string            `json:"parameter_size"`
	Quant        string            `json:"quantization"`
	Format       string            `json:"format"`
	Parent       string            `json:"parent_model"`
	Context      int               `json:"context_length"`
	VRAM         int64             `json:"size_vram"`
	ModifiedAt   time.Time         `json:"modified_at"`
	Loaded       bool              `json:"loaded"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Providers    []UIModelProvider `json:"providers"`
}

func capabilityStrings(caps []model.Capability) []string {
	out := make([]string, 0, len(caps))
	seen := make(map[string]struct{}, len(caps))
	for _, c := range caps {
		s := strings.TrimSpace(string(c))
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

type UIModelsResponse struct {
	Models []UIModel `json:"models"`
}

func findWebDir() (string, bool) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "web"))
	}
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			candidates = append(candidates, filepath.Join(dir, "web"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, d := range candidates {
		if st, err := os.Stat(filepath.Join(d, "index.html")); err == nil && !st.IsDir() {
			return d, true
		}
	}
	return "", false
}

func webDir() string {
	if d, ok := findWebDir(); ok {
		return d
	}
	return "web"
}

func uiFileSystem() http.FileSystem {
	if dir, ok := findWebDir(); ok {
		log.Debugf("serving UI from %s", dir)
		return http.Dir(dir)
	}
	log.Debugf("serving UI from embedded web/")
	return http.FS(web.FS)
}

func (p *Proxy) uiRootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
}

func (p *Proxy) uiHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/", http.StatusTemporaryRedirect)
}

type UIConfigResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (p *Proxy) apiUIConfigHandler(w http.ResponseWriter, r *http.Request) {
	path := "config.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&UIConfigResponse{
		Path:    path,
		Content: string(data),
	})
}

func (p *Proxy) apiUIModelsHandler(w http.ResponseWriter, r *http.Request) {
	peers, _ := p.mesh.GetPeerMap()

	p.lock.RLock()
	defer p.lock.RUnlock()

	resp := UIModelsResponse{
		Models: make([]UIModel, 0, len(p.meshRoutes)),
	}

	for _, route := range p.meshRoutes {
		ctxLen := route.Properties.Details.ContextLength
		if ctxLen == 0 {
			ctxLen = route.Process.ContextLength
		}
		m := UIModel{
			Name:         route.Name,
			Model:        route.Model,
			Size:         route.Properties.Size,
			Digest:       route.Properties.Digest,
			Family:       route.Properties.Details.Family,
			Families:     route.Properties.Details.Families,
			Params:       route.Properties.Details.ParameterSize,
			Quant:        route.Properties.Details.QuantizationLevel,
			Format:       route.Properties.Details.Format,
			Parent:       route.Properties.Details.ParentModel,
			Context:      ctxLen,
			VRAM:         route.Process.SizeVRAM,
			ModifiedAt:   route.Properties.ModifiedAt,
			Loaded:       route.Process.Name != "",
			Capabilities: capabilityStrings(route.Properties.Capabilities),
			Providers:    make([]UIModelProvider, 0, len(route.Servers)),
		}
		for peerID := range route.Servers {
			name := shortPeer(peerID)
			if peerID == p.mesh.ID && p.mesh.Name != "" {
				name = p.mesh.Name
			} else if n, ok := peers[peerID]; ok && n.Name != "" {
				name = n.Name
			}
			m.Providers = append(m.Providers, UIModelProvider{
				Name:   name,
				PeerID: peerID,
				Self:   peerID == p.mesh.ID,
			})
		}
		sort.Slice(m.Providers, func(i, j int) bool {
			if m.Providers[i].Self != m.Providers[j].Self {
				return m.Providers[i].Self
			}
			return m.Providers[i].Name < m.Providers[j].Name
		})
		resp.Models = append(resp.Models, m)
	}

	sort.Slice(resp.Models, func(i, j int) bool {
		return resp.Models[i].Name < resp.Models[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&resp)
}

func shortPeer(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:8] + "…"
}
