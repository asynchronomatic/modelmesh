package proxy

import (
	"encoding/json"
	"io"
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
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)
	for _, needle := range []string{"Mesh", "Models", "<th>Owner</th>", "<th>Context</th>", "<th>Visibility</th>", "<th>Capabilities</th>"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("index missing %s", needle)
		}
	}
	for _, old := range []string{"<th>Size</th>", "<th>Family</th>", "<th>Parameters</th>"} {
		if strings.Contains(body, old) {
			t.Fatalf("index still has old model column %s", old)
		}
	}
}

func TestUIModelsJSONShape(t *testing.T) {
	resp := UIModelsResponse{
		Models: []UIModel{{
			Name:          "llama3.2:latest",
			Model:         "llama3.2:latest",
			Private:       true,
			Owner:         "alice",
			ContextLength: 8192,
			Capabilities:  []string{"completion", "tools", "vision"},
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
	for _, needle := range []string{
		`"models"`, `"providers"`, `"peer_id"`, `"self":true`,
		`"capabilities"`, `"completion"`, `"vision"`,
		`"private":true`, `"owner":"alice"`, `"context_length":8192`,
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("missing %s in %s", needle, s)
		}
	}
	for _, old := range []string{`"digest"`, `"parameter_size"`, `"quantization"`, `"size_vram"`, `"loaded"`} {
		if strings.Contains(s, old) {
			t.Fatalf("unexpected legacy field %s in %s", old, s)
		}
	}
}

func TestAppJSUsesNewUIModelFields(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(webDir(), "js", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{"m.private", "m.owner", "m.context_length", "badge-private", "badge-shared"} {
		if !strings.Contains(s, needle) {
			t.Fatalf("app.js missing %s", needle)
		}
	}
	for _, old := range []string{"m.loaded", "m.parameter_size", "m.size_vram", "m.digest", "formatBytes", "m.family", "m.quantization"} {
		if strings.Contains(s, old) {
			t.Fatalf("app.js still uses legacy UIModel field %s", old)
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
