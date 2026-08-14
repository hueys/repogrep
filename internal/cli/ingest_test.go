package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/hueys/repogrep/internal/source"
	"github.com/hueys/repogrep/internal/store"
)

var errBoom = errors.New("boom")

// fakeResults turns a slice of records into a closed, pre-filled channel,
// simulating what a real source.Source.Fetch would stream.
func fakeResults(recs ...source.RepoRecord) <-chan source.FetchResult {
	out := make(chan source.FetchResult, len(recs))
	for _, r := range recs {
		out <- source.FetchResult{Record: r}
	}
	close(out)
	return out
}

// TestIngestEndToEnd exercises import (via ingest) -> search against a real
// temp-file store, with no network involved. This is also an architecture
// smoke test: it proves the CLI layer only depends on the source.Source
// vocabulary, never on the GitHub package directly.
func TestIngestEndToEnd(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repogrep.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	recs := []source.RepoRecord{
		{
			SourceName: "github", SourceID: "1", Owner: "foo", Name: "rag-tool",
			FullName: "foo/rag-tool", URL: "https://github.com/foo/rag-tool",
			PrimaryLanguage: "Go", Topics: []string{"rag", "search"},
			Description: "Retrieval augmented generation toolkit",
			README:      "Does embeddings-based retrieval over your docs.",
		},
		{
			SourceName: "github", SourceID: "2", Owner: "bar", Name: "todo-cli",
			FullName: "bar/todo-cli", URL: "https://github.com/bar/todo-cli",
			PrimaryLanguage: "Python", Topics: []string{"cli"},
			Description: "A todo list manager",
			README:      "Plain todo list manager, nothing fancy.",
		},
	}

	added, updated, errored := ingest(st, fakeResults(recs...), false)
	if added != 2 || updated != 0 || errored != 0 {
		t.Fatalf("unexpected ingest counts: added=%d updated=%d errored=%d", added, updated, errored)
	}

	// Re-ingesting the same records should upsert, not duplicate.
	added, updated, errored = ingest(st, fakeResults(recs...), false)
	if added != 0 || updated != 2 || errored != 0 {
		t.Fatalf("expected re-ingest to update in place, got added=%d updated=%d errored=%d", added, updated, errored)
	}

	all, err := st.List(store.ListFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 repos stored, got %d", len(all))
	}

	results, err := st.Search("embeddings", store.SearchFilters{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].FullName != "foo/rag-tool" {
		t.Fatalf("expected search for 'embeddings' to find foo/rag-tool, got %+v", results)
	}

	results, err = st.Search("todo", store.SearchFilters{Language: "Go"})
	if err != nil {
		t.Fatalf("Search with language filter: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected language filter to exclude bar/todo-cli, got %+v", results)
	}

	repo, err := st.FindByFullName("bar/todo-cli")
	if err != nil {
		t.Fatalf("FindByFullName: %v", err)
	}
	if repo.Description != "A todo list manager" {
		t.Fatalf("unexpected repo detail: %+v", repo)
	}

	// ingest reports per-record errors without aborting the whole run.
	mixed := make(chan source.FetchResult, 2)
	mixed <- source.FetchResult{Record: source.RepoRecord{SourceName: "github", SourceID: "3", Owner: "ok", Name: "one", FullName: "ok/one", URL: "https://x"}}
	mixed <- source.FetchResult{Err: errBoom}
	close(mixed)
	added, updated, errored = ingest(st, mixed, false)
	if added != 1 || errored != 1 {
		t.Fatalf("expected one success and one error, got added=%d updated=%d errored=%d", added, updated, errored)
	}
}
