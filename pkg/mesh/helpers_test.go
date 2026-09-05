package mesh

import (
	"path/filepath"
	"testing"
)

func TestNodeIDFromKey(t *testing.T) {
	key, err := LoadOrCreateKey(filepath.Join(t.TempDir(), "node.key"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := NodeIDFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty node id")
	}

	again, err := NodeIDFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if again != id {
		t.Fatalf("got %q want %q", again, id)
	}
}
