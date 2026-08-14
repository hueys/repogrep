// Package source defines the pluggable ingestion abstraction. Each backend
// (GitHub today; GitLab, a curated list, local disk, etc. later) implements
// Source and registers itself by name so the CLI and storage layer never
// need to know about backend specifics.
package source

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// RepoRecord is what a Source yields for a single repo. It is intentionally
// backend-agnostic: fields that don't apply to a given source are left zero.
type RepoRecord struct {
	SourceName string // e.g. "github"
	SourceID   string // stable external id within that source

	Owner    string
	Name     string
	FullName string // owner/name

	Description string
	URL         string
	Homepage    string

	PrimaryLanguage string
	Topics          []string

	Stars int
	Forks int

	IsArchived bool
	IsFork     bool

	License       string
	DefaultBranch string

	PushedAt      time.Time // last push/commit activity
	RepoCreatedAt time.Time
	RepoUpdatedAt time.Time // source's own "metadata last updated" timestamp

	README     string // decoded content; "" if none
	ReadmePath string

	Extra map[string]string // escape hatch for source-specific fields
}

// FetchOptions parameterizes a Fetch call.
type FetchOptions struct {
	Username    string // "" means "the source's default/authenticated identity"
	Limit       int    // 0 = unlimited
	Concurrency int    // worker pool size for any per-record fetches; 0 = source default
}

// FetchResult is one item streamed back from Fetch. Err is a per-record
// error (e.g. one repo's README failed to fetch) and does not mean the rest
// of the run should be abandoned; callers should log it and continue.
type FetchResult struct {
	Record RepoRecord
	Err    error
}

// Source is implemented once per ingestion backend. Fetch streams results on
// the returned channel so a caller can begin persisting before the whole run
// completes; the channel is closed when the run finishes (successfully or
// not). Fetch should respect ctx cancellation.
type Source interface {
	Name() string
	Fetch(ctx context.Context, opts FetchOptions) (<-chan FetchResult, error)
}

// Factory constructs a Source instance, performing any setup (e.g. auth
// discovery) that might fail.
type Factory func() (Source, error)

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

// Register makes a source factory available under name. Intended to be
// called from an implementing package's init().
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = f
}

// Get looks up a registered source factory by name.
func Get(name string) (Factory, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// Names returns all registered source names, sorted.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(factories))
	for n := range factories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// New constructs a registered source by name, or an error listing what's
// available if name isn't registered.
func New(name string) (Source, error) {
	f, ok := Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown source %q (available: %v)", name, Names())
	}
	return f()
}
