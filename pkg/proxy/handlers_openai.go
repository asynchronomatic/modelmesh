package proxy

import (
	"encoding/json"
	"net/http"

	"github.com/asynchronomatic/speakeasy/pkg/mesh"
	"github.com/asynchronomatic/speakeasy/pkg/proxy/modeldex"
)

// FIXME: alias until we restructure the code a bit
type OpenaiModel = modeldex.OpenaiModel
type OpenaiModelList = modeldex.OpenaiModelList

func (p *Proxy) openaiListModelsHandler(w http.ResponseWriter, r *http.Request) {
	if mesh.IsSource(r) {
		http.Error(w, "we should not see this rpc over the mesh network", http.StatusInternalServerError)
		return
	}

	resp := &OpenaiModelList{
		Object: "list",
	}

	for _, model := range p.modelRouter.ListMeshModels() {
		resp.Data = append(resp.Data, OpenaiModel{
			ID:     model.Name,
			Object: "model",
			//Created: model.Properties.ModifiedAt.Unix(),
			OwnedBy: "ollama",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
