package proxy

import (
	"encoding/json"
	"net/http"

	oapi "github.com/ollama/ollama/api"

	"modelmesh/pkg/mesh"
)

// apiListProcessHandler implements the ollama /api/ps API endpoint
// this simply returns the originally discovered ps information
func (p *Proxy) apiListProcessHandler(w http.ResponseWriter, r *http.Request) {
	if mesh.IsSource(r) {
		http.Error(w, "we should not see this rpc over the mesh network", http.StatusInternalServerError)
		return
	}

	resp := &oapi.ProcessResponse{}
	for _, r := range p.meshRoutes {
		resp.Models = append(resp.Models, r.Process)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// apiListTagsHandler handles the /api/tags endpoint, returning a list of model tags based on the request source.
func (p *Proxy) apiListTagsHandler(w http.ResponseWriter, r *http.Request) {
	if mesh.IsSource(r) {
		http.Error(w, "we should not see this rpc over the mesh network", http.StatusInternalServerError)
		return
	}

	resp := &oapi.ListResponse{}
	for _, r := range p.meshRoutes {
		resp.Models = append(resp.Models, r.Properties)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
