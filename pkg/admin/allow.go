package admin

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"modelmesh/pkg/log"
)

type AllowList struct {
	mu    sync.RWMutex
	allow map[string]struct{}
	path  string
}

func NewAllowList(path string) (*AllowList, error) {
	l := &AllowList{allow: map[string]struct{}{}, path: path}
	if path == "" {
		return l, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return l, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		l.allow[line] = struct{}{}
	}
	return l, sc.Err()
}

func (l *AllowList) Has(id string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.allow[id]
	if !ok {
		log.Warnf("acl denied for %s", id)
	}
	return ok
}

func (l *AllowList) Add(id string) {
	l.mu.Lock()
	l.allow[id] = struct{}{}
	path := l.path
	snapshot := make([]string, 0, len(l.allow))
	for id := range l.allow {
		snapshot = append(snapshot, id)
	}
	l.mu.Unlock()
	if path != "" {
		_ = persist(path, snapshot)
	}
}

func (l *AllowList) Remove(id string) {
	l.mu.Lock()
	delete(l.allow, id)
	path := l.path
	snapshot := make([]string, 0, len(l.allow))
	for id := range l.allow {
		snapshot = append(snapshot, id)
	}
	l.mu.Unlock()
	if path != "" {
		_ = persist(path, snapshot)
	}
}

func (l *AllowList) Peers() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.allow))
	for id := range l.allow {
		out = append(out, id)
	}
	return out
}

func persist(path string, ids []string) error {
	var b strings.Builder
	for _, id := range ids {
		b.WriteString(id)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}
