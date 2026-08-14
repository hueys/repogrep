package store

import (
	"fmt"
	"strings"

	"github.com/hueys/repogrep/internal/model"
)

// SearchFilters narrows a full-text Search in addition to the query text.
type SearchFilters struct {
	Language        string
	Topic           string
	IncludeArchived bool // if false (default), archived repos are excluded
	Limit           int  // 0 = no limit
}

// Search runs a full-text query over full_name/description/readme/topics,
// ranked by FTS5's bm25 relevance score (best match first).
func (s *Store) Search(query string, f SearchFilters) ([]model.Repo, error) {
	match := ftsQuery(query)
	if match == "" {
		return nil, fmt.Errorf("empty search query")
	}

	//nolint:gosec // G202: prefixColumns only ever concatenates our own static column list, never user input; the query value is always bound via args
	sqlQuery := `
		SELECT ` + prefixColumns("r") + `
		FROM repos r
		JOIN repos_fts ON repos_fts.rowid = r.id
		WHERE repos_fts MATCH ?`
	args := []any{match}

	if !f.IncludeArchived {
		sqlQuery += ` AND r.is_archived = 0`
	}
	if f.Language != "" {
		sqlQuery += ` AND r.primary_language = ? COLLATE NOCASE`
		args = append(args, f.Language)
	}
	if f.Topic != "" {
		sqlQuery += ` AND r.id IN (SELECT repo_id FROM topics WHERE topic = ? COLLATE NOCASE)`
		args = append(args, f.Topic)
	}

	sqlQuery += ` ORDER BY bm25(repos_fts) ASC`
	if f.Limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, f.Limit)
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search repos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []model.Repo
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// prefixColumns qualifies each column in repoColumns with alias.
func prefixColumns(alias string) string {
	cols := strings.Split(strings.ReplaceAll(strings.TrimSpace(repoColumns), "\n", " "), ",")
	for i, c := range cols {
		cols[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

// ftsQuery turns free-form user input into a safe FTS5 MATCH expression:
// each whitespace-separated term is quoted as a literal phrase and joined
// with implicit AND, so punctuation/hyphens in the input (e.g. "rag-app")
// can't be misinterpreted as FTS5 query syntax.
func ftsQuery(input string) string {
	terms := strings.Fields(input)
	if len(terms) == 0 {
		return ""
	}
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	return strings.Join(quoted, " ")
}
