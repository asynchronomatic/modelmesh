package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewAllowListEmptyPath(t *testing.T) {
	l, err := NewAllowList("")
	if err != nil {
		t.Fatal(err)
	}
	if l.Has("anyone") {
		t.Fatal("empty allow list should deny")
	}
	if peers := l.Peers(); len(peers) != 0 {
		t.Fatalf("expected no peers, got %v", peers)
	}

	l.Add("peer-1")
	if !l.Has("peer-1") {
		t.Fatal("Add with empty path should still allow in memory")
	}
}

func TestNewAllowListMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.list")
	l, err := NewAllowList(path)
	if err != nil {
		t.Fatal(err)
	}
	if l.Has("peer-1") {
		t.Fatal("missing file should load empty list")
	}
	if peers := l.Peers(); len(peers) != 0 {
		t.Fatalf("expected no peers, got %v", peers)
	}
}

func TestNewAllowListOpenError(t *testing.T) {
	dir := t.TempDir()
	_, err := NewAllowList(dir)
	if err == nil {
		t.Fatal("expected error opening directory as allow list")
	}
}

func TestNewAllowListLoadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allow.list")
	content := "# comment\n\npeer-1\n  peer-2  \n# another\npeer-3\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	l, err := NewAllowList(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"peer-1", "peer-2", "peer-3"} {
		if !l.Has(id) {
			t.Fatalf("expected %s allowed", id)
		}
	}
	if l.Has("# comment") || l.Has("comment") || l.Has("") {
		t.Fatal("comments and blanks should not be treated as peers")
	}
	assertPeers(t, l, "peer-1", "peer-2", "peer-3")
}

func TestAllowListAddRemoveMemory(t *testing.T) {
	l, err := NewAllowList("")
	if err != nil {
		t.Fatal(err)
	}

	l.Add("peer-1")
	if !l.Has("peer-1") {
		t.Fatal("Add should allow peer-1")
	}
	l.Add("peer-1")
	assertPeers(t, l, "peer-1")

	l.Remove("peer-1")
	if l.Has("peer-1") {
		t.Fatal("Remove should deny peer-1")
	}
	l.Remove("peer-1")
	assertPeers(t, l)
}

func TestAllowListAddRemovePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allow.list")
	l, err := NewAllowList(path)
	if err != nil {
		t.Fatal(err)
	}

	l.Add("peer-a")
	l.Add("peer-b")
	assertPersisted(t, path, "peer-a", "peer-b")

	reloaded, err := NewAllowList(path)
	if err != nil {
		t.Fatal(err)
	}
	assertPeers(t, reloaded, "peer-a", "peer-b")

	l.Remove("peer-a")
	assertPersisted(t, path, "peer-b")
	if l.Has("peer-a") {
		t.Fatal("peer-a should be removed")
	}

	l.Remove("peer-b")
	assertPersisted(t, path)
	assertPeers(t, l)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("persist mode: got %o want 600", info.Mode().Perm())
	}
}

func TestPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.list")
	if err := persist(path, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "a\nb\n" {
		t.Fatalf("persist content: %q", b)
	}

	if err := persist(path, nil); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 0 {
		t.Fatalf("empty persist content: %q", b)
	}
}

func TestAllowListConcurrent(t *testing.T) {
	l, err := NewAllowList("")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("p-%d", i)
			l.Add(id)
			_ = l.Has(id)
			_ = l.Peers()
			l.Remove(id)
		}(i)
	}
	wg.Wait()
	assertPeers(t, l)
}

func assertPeers(t *testing.T, l *AllowList, want ...string) {
	t.Helper()
	got := l.Peers()
	if len(got) != len(want) {
		t.Fatalf("Peers() = %v, want %v", got, want)
	}
	set := make(map[string]struct{}, len(got))
	for _, p := range got {
		set[p] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatalf("Peers() missing %q: %v", w, got)
		}
	}
}

func assertPersisted(t *testing.T, path string, want ...string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) != len(want) {
		t.Fatalf("persisted %v, want %v", lines, want)
	}
	set := make(map[string]struct{}, len(lines))
	for _, p := range lines {
		set[p] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Fatalf("persisted missing %q: %v", w, lines)
		}
	}
}
