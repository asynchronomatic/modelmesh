package jsonkv

import (
	"errors"
	"path/filepath"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestPutGetDelete(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.jkv"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	in := sample{Name: "alpha", Count: 3}
	if err := s.Put("item", in); err != nil {
		t.Fatal(err)
	}

	var out sample
	if err := s.Get("item", &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("Get: %+v, want %+v", out, in)
	}

	if err := s.Put("item", sample{Name: "beta", Count: 9}); err != nil {
		t.Fatal(err)
	}
	if err := s.Get("item", &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "beta" || out.Count != 9 {
		t.Fatalf("overwrite Get: %+v", out)
	}

	if err := s.Delete("item"); err != nil {
		t.Fatal(err)
	}
	if err := s.Get("item", &out); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: %v, want ErrNotFound", err)
	}
}

func TestGetMissing(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "missing.jkv"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	var out sample
	if err := s.Get("nope", &out); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing Get: %v, want ErrNotFound", err)
	}
}

func TestReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist.jkv")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("k", sample{Name: "kept", Count: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	var out sample
	if err := s.Get("k", &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "kept" || out.Count != 1 {
		t.Fatalf("reopen Get: %+v", out)
	}
}
