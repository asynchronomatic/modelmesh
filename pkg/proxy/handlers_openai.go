package proxy

import (
	"encoding/json"
	"net/http"

	"modelmesh/pkg/mesh"
)

type OpenaiModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenaiModelList struct {
	Object string        `json:"object"`
	Data   []OpenaiModel `json:"data"`
}

func (p *Proxy) openaiListModelsHandler(w http.ResponseWriter, r *http.Request) {
	if mesh.IsSource(r) {
		http.Error(w, "we should not see this rpc over the mesh network", http.StatusInternalServerError)
		return
	}

	resp := &OpenaiModelList{
		Object: "list",
	}

	for _, model := range p.localRoutes {
		resp.Data = append(resp.Data, OpenaiModel{
			ID:      model.Name,
			Object:  "model",
			Created: model.Properties.ModifiedAt.Unix(),
			OwnedBy: "ollama",
		})
	}

	if MeshModelPrefix != "" {
		for _, model := range p.meshRoutes {
			resp.Data = append(resp.Data, OpenaiModel{
				ID:      MeshModelPrefix + model.Name,
				Object:  "model",
				Created: model.Properties.ModifiedAt.Unix(),
				OwnedBy: "ollama",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
