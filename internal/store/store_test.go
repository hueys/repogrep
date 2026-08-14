package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/hueys/repogrep/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repogrep.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func sampleRepo() model.Repo {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return model.Repo{
		Source:          "github",
		SourceID:        "123",
		Owner:           "foo",
		Name:            "bar",
		FullName:        "foo/bar",
		Description:     "A neat retrieval augmented generation toolkit",
		URL:             "https://github.com/foo/bar",
		PrimaryLanguage: "Go",
		Topics:          []string{"rag", "machine-learning"},
		Stars:           42,
		README:          "# bar\n\nDoes cool RAG stuff with embeddings.",
		PushedAt:        now,
		RepoCreatedAt:   now,
		RepoUpdatedAt:   now,
		FirstSeen:       now,
		LastSynced:      now,
	}
}

func TestUpsertInsertThenUpdate(t *testing.T) {
	s := newTestStore(t)
	rec := sampleRepo()

	id1, inserted, err := s.UpsertRepo(rec)
	if err != nil {
		t.Fatalf("UpsertRepo (insert): %v", err)
	}
	if !inserted {
		t.Fatalf("expected inserted=true on first upsert")
	}

	got, err := s.GetByFullName("github", "foo/bar")
	if err != nil {
		t.Fatalf("GetByFullName: %v", err)
	}
	if got.Stars != 42 || len(got.Topics) != 2 {
		t.Fatalf("unexpected row after insert: %+v", got)
	}

	// Update: change stars/topics, bump last_synced, keep first_seen fixed.
	rec.Stars = 99
	rec.Topics = []string{"rag"}
	originalFirstSeen := rec.FirstSeen
	rec.FirstSeen = time.Time{} // should be ignored on conflict path
	rec.LastSynced = originalFirstSeen.Add(24 * time.Hour)

	id2, inserted, err := s.UpsertRepo(rec)
	if err != nil {
		t.Fatalf("UpsertRepo (update): %v", err)
	}
	if inserted {
		t.Fatalf("expected inserted=false on second upsert")
	}
	if id1 != id2 {
		t.Fatalf("expected same id across upserts, got %d then %d", id1, id2)
	}

	got, err = s.GetByFullName("github", "foo/bar")
	if err != nil {
		t.Fatalf("GetByFullName after update: %v", err)
	}
	if got.Stars != 99 {
		t.Fatalf("stars not updated: got %d", got.Stars)
	}
	if len(got.Topics) != 1 || got.Topics[0] != "rag" {
		t.Fatalf("topics not replaced: got %v", got.Topics)
	}
	if !got.FirstSeen.Equal(originalFirstSeen) {
		t.Fatalf("first_seen should be preserved across update, got %v want %v", got.FirstSeen, originalFirstSeen)
	}
	if !got.LastSynced.Equal(rec.LastSynced) {
		t.Fatalf("last_synced should be bumped, got %v want %v", got.LastSynced, rec.LastSynced)
	}

	all, err := s.List(ListFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 row after insert+update, got %d", len(all))
	}
}

func TestSearchFTSAndFilters(t *testing.T) {
	s := newTestStore(t)

	rag := sampleRepo()
	if _, _, err := s.UpsertRepo(rag); err != nil {
		t.Fatalf("upsert rag: %v", err)
	}

	other := sampleRepo()
	other.Owner, other.Name, other.FullName = "baz", "qux", "baz/qux"
	other.SourceID = "456"
	other.URL = "https://github.com/baz/qux"
	other.Description = "A CLI tool for managing todo lists"
	other.README = "# qux\n\nPlain old todo list manager, nothing fancy."
	other.PrimaryLanguage = "Python"
	other.Topics = []string{"cli", "todo"}
	if _, _, err := s.UpsertRepo(other); err != nil {
		t.Fatalf("upsert other: %v", err)
	}

	results, err := s.Search("embeddings", SearchFilters{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].FullName != "foo/bar" {
		t.Fatalf("expected only foo/bar to match 'embeddings', got %+v", results)
	}

	results, err = s.Search("todo", SearchFilters{Language: "Go"})
	if err != nil {
		t.Fatalf("Search with language filter: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected language filter to exclude the Python match, got %+v", results)
	}

	results, err = s.List(ListFilters{Topic: "rag"})
	if err != nil {
		t.Fatalf("List by topic: %v", err)
	}
	if len(results) != 1 || results[0].FullName != "foo/bar" {
		t.Fatalf("expected topic filter to find foo/bar, got %+v", results)
	}
}

func TestSearchSort(t *testing.T) {
	s := newTestStore(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Both repos match the same query term ("toolkit") so relevance alone
	// wouldn't reliably separate them; stars/pushed_at are set so
	// --sort stars/--sort updated have an unambiguous expected order.
	low := sampleRepo()
	low.Owner, low.Name, low.FullName = "low", "repo", "low/repo"
	low.SourceID = "10"
	low.Description = "a small toolkit"
	low.README = "toolkit for small things"
	low.Stars = 5
	low.PushedAt = now.AddDate(0, 0, -30)
	if _, _, err := s.UpsertRepo(low); err != nil {
		t.Fatalf("upsert low: %v", err)
	}

	high := sampleRepo()
	high.Owner, high.Name, high.FullName = "high", "repo", "high/repo"
	high.SourceID = "11"
	high.Description = "a big toolkit"
	high.README = "toolkit for big things"
	high.Stars = 500
	high.PushedAt = now
	if _, _, err := s.UpsertRepo(high); err != nil {
		t.Fatalf("upsert high: %v", err)
	}

	results, err := s.Search("toolkit", SearchFilters{Sort: "stars"})
	if err != nil {
		t.Fatalf("Search sort=stars: %v", err)
	}
	if len(results) != 2 || results[0].FullName != "high/repo" || results[1].FullName != "low/repo" {
		t.Fatalf("expected high/repo then low/repo sorted by stars desc, got %+v", results)
	}

	results, err = s.Search("toolkit", SearchFilters{Sort: "updated"})
	if err != nil {
		t.Fatalf("Search sort=updated: %v", err)
	}
	if len(results) != 2 || results[0].FullName != "high/repo" || results[1].FullName != "low/repo" {
		t.Fatalf("expected high/repo then low/repo sorted by pushed_at desc, got %+v", results)
	}
}

func TestSearchExcludesArchivedByDefault(t *testing.T) {
	s := newTestStore(t)

	rec := sampleRepo()
	rec.IsArchived = true
	if _, _, err := s.UpsertRepo(rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	results, err := s.Search("embeddings", SearchFilters{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected archived repo excluded by default, got %+v", results)
	}

	results, err = s.Search("embeddings", SearchFilters{IncludeArchived: true})
	if err != nil {
		t.Fatalf("Search with IncludeArchived: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected archived repo included, got %+v", results)
	}
}

func TestDeleteCascadesToFTSAndTopics(t *testing.T) {
	s := newTestStore(t)
	rec := sampleRepo()
	id, _, err := s.UpsertRepo(rec)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	n, err := s.Delete([]int64{id})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row deleted, got %d", n)
	}

	all, err := s.List(ListFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected no rows after delete, got %d", len(all))
	}

	// FTS row should be gone too, not just orphaned.
	var ftsCount int
	if err := s.db.QueryRow(`SELECT count(*) FROM repos_fts WHERE rowid = ?`, id).Scan(&ftsCount); err != nil {
		t.Fatalf("query repos_fts: %v", err)
	}
	if ftsCount != 0 {
		t.Fatalf("expected repos_fts row removed by delete trigger, found %d", ftsCount)
	}

	var topicCount int
	if err := s.db.QueryRow(`SELECT count(*) FROM topics WHERE repo_id = ?`, id).Scan(&topicCount); err != nil {
		t.Fatalf("query topics: %v", err)
	}
	if topicCount != 0 {
		t.Fatalf("expected topics cascade-deleted, found %d", topicCount)
	}
}

func TestPruneCandidates(t *testing.T) {
	s := newTestStore(t)
	realNow := Now
	fixedNow := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	Now = func() time.Time { return fixedNow }
	t.Cleanup(func() { Now = realNow })

	active := sampleRepo()
	active.Owner, active.Name, active.FullName = "active", "repo", "active/repo"
	active.SourceID = "1"
	active.PushedAt = fixedNow.AddDate(0, -1, 0) // 1 month ago
	if _, _, err := s.UpsertRepo(active); err != nil {
		t.Fatalf("upsert active: %v", err)
	}

	stale := sampleRepo()
	stale.Owner, stale.Name, stale.FullName = "stale", "repo", "stale/repo"
	stale.SourceID = "2"
	stale.PushedAt = fixedNow.AddDate(-2, 0, 0) // 2 years ago
	if _, _, err := s.UpsertRepo(stale); err != nil {
		t.Fatalf("upsert stale: %v", err)
	}

	archived := sampleRepo()
	archived.Owner, archived.Name, archived.FullName = "archived", "repo", "archived/repo"
	archived.SourceID = "3"
	archived.PushedAt = fixedNow.AddDate(0, -1, 0) // recent, but archived
	archived.IsArchived = true
	if _, _, err := s.UpsertRepo(archived); err != nil {
		t.Fatalf("upsert archived: %v", err)
	}

	candidates, err := s.PruneCandidates(12)
	if err != nil {
		t.Fatalf("PruneCandidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 prune candidates, got %d: %+v", len(candidates), candidates)
	}
	byName := map[string]PruneCandidate{}
	for _, c := range candidates {
		byName[c.Repo.FullName] = c
	}
	if _, ok := byName["active/repo"]; ok {
		t.Fatalf("active repo should not be a prune candidate")
	}
	if c, ok := byName["stale/repo"]; !ok || c.Reason != "inactive" {
		t.Fatalf("expected stale/repo flagged inactive, got %+v (ok=%v)", c, ok)
	}
	if c, ok := byName["archived/repo"]; !ok || c.Reason != "archived" {
		t.Fatalf("expected archived/repo flagged archived, got %+v (ok=%v)", c, ok)
	}
}
