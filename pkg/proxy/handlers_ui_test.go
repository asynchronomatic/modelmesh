package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/types/model"

	"modelmesh/web"
)

func TestWebDirHasIndex(t *testing.T) {
	dir := webDir()
	index := filepath.Join(dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		t.Fatalf("expected %s: %v", index, err)
	}
}

func TestEmbeddedWebHasIndex(t *testing.T) {
	f, err := web.FS.Open("index.html")
	if err != nil {
		t.Fatalf("embedded index.html: %v", err)
	}
	f.Close()
}

func TestUIStaticAssets(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /ui/", http.StripPrefix("/ui/", http.FileServer(uiFileSystem())))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	for _, path := range []string{"/ui/", "/ui/css/styles.css", "/ui/js/app.js"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, res.StatusCode)
		}
		res.Body.Close()
	}

	res, err := http.Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	buf := make([]byte, 4096)
	n, _ := res.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "Mesh") || !strings.Contains(body, "Models") {
		t.Fatalf("index missing Mesh/Models panels: %s", body)
	}
}

func TestUIModelsJSONShape(t *testing.T) {
	resp := UIModelsResponse{
		Models: []UIModel{{
			Name:         "llama3.2:latest",
			Model:        "llama3.2:latest",
			Capabilities: []string{"completion", "tools", "vision"},
			Providers: []UIModelProvider{{
				Name:   "local",
				PeerID: "12D3KooWtest",
				Self:   true,
			}},
		}},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{`"models"`, `"providers"`, `"peer_id"`, `"self":true`, `"capabilities"`, `"completion"`, `"vision"`} {
		if !strings.Contains(s, needle) {
			t.Fatalf("missing %s in %s", needle, s)
		}
	}
}

func TestCapabilityStrings(t *testing.T) {
	got := capabilityStrings([]model.Capability{
		"completion",
		" tools ",
		"",
		"completion",
		"vision",
	})
	want := []string{"completion", "tools", "vision"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
