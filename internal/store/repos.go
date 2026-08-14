package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hueys/repogrep/internal/model"
)

const repoColumns = `
	id, source, source_id, owner, name, full_name, description, url, homepage,
	primary_language, topics_text, stars, forks, is_archived, is_fork,
	license, default_branch, pushed_at, repo_created_at, repo_updated_at,
	readme, readme_path, first_seen, last_synced
`

// scanRepo scans one row shaped like repoColumns into a model.Repo.
func scanRepo(row interface{ Scan(...any) error }) (model.Repo, error) {
	var (
		r                                          model.Repo
		description, homepage, primaryLanguage     sql.NullString
		license, defaultBranch, readme, readmePath sql.NullString
		topicsText                                 string
		isArchived, isFork                         int
		pushedAt, repoCreatedAt, repoUpdatedAt     sql.NullTime
	)
	err := row.Scan(
		&r.ID, &r.Source, &r.SourceID, &r.Owner, &r.Name, &r.FullName, &description, &r.URL, &homepage,
		&primaryLanguage, &topicsText, &r.Stars, &r.Forks, &isArchived, &isFork,
		&license, &defaultBranch, &pushedAt, &repoCreatedAt, &repoUpdatedAt,
		&readme, &readmePath, &r.FirstSeen, &r.LastSynced,
	)
	if err != nil {
		return model.Repo{}, err
	}
	r.Description = description.String
	r.Homepage = homepage.String
	r.PrimaryLanguage = primaryLanguage.String
	r.License = license.String
	r.DefaultBranch = defaultBranch.String
	r.README = readme.String
	r.ReadmePath = readmePath.String
	r.PushedAt = pushedAt.Time
	r.RepoCreatedAt = repoCreatedAt.Time
	r.RepoUpdatedAt = repoUpdatedAt.Time
	r.IsArchived = isArchived != 0
	r.IsFork = isFork != 0
	r.Topics = splitTopics(topicsText)
	return r, nil
}

// UpsertRepo inserts rec, or updates the existing row for the same
// (source, owner, name), preserving the original first_seen. Returns the
// row's id and whether this call inserted a new row (vs. updated one).
func (s *Store) UpsertRepo(rec model.Repo) (id int64, inserted bool, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.QueryRow(`SELECT id FROM repos WHERE source = ? AND owner = ? AND name = ?`,
		rec.Source, rec.Owner, rec.Name).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		inserted = true
	case err != nil:
		return 0, false, fmt.Errorf("check existing repo: %w", err)
	}

	topicsText := strings.Join(rec.Topics, " ")

	_, err = tx.Exec(`
		INSERT INTO repos (
			source, source_id, owner, name, full_name, description, url, homepage,
			primary_language, topics_text, stars, forks, is_archived, is_fork,
			license, default_branch, pushed_at, repo_created_at, repo_updated_at,
			readme, readme_path, first_seen, last_synced
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(source, owner, name) DO UPDATE SET
			source_id = excluded.source_id,
			full_name = excluded.full_name,
			description = excluded.description,
			url = excluded.url,
			homepage = excluded.homepage,
			primary_language = excluded.primary_language,
			topics_text = excluded.topics_text,
			stars = excluded.stars,
			forks = excluded.forks,
			is_archived = excluded.is_archived,
			is_fork = excluded.is_fork,
			license = excluded.license,
			default_branch = excluded.default_branch,
			pushed_at = excluded.pushed_at,
			repo_created_at = excluded.repo_created_at,
			repo_updated_at = excluded.repo_updated_at,
			readme = excluded.readme,
			readme_path = excluded.readme_path,
			last_synced = excluded.last_synced
	`,
		rec.Source, rec.SourceID, rec.Owner, rec.Name, rec.FullName,
		nullableString(rec.Description), rec.URL, nullableString(rec.Homepage),
		nullableString(rec.PrimaryLanguage), topicsText, rec.Stars, rec.Forks,
		boolToInt(rec.IsArchived), boolToInt(rec.IsFork),
		nullableString(rec.License), nullableString(rec.DefaultBranch),
		nullableTime(rec.PushedAt), nullableTime(rec.RepoCreatedAt), nullableTime(rec.RepoUpdatedAt),
		nullableString(rec.README), nullableString(rec.ReadmePath),
		rec.FirstSeen, rec.LastSynced,
	)
	if err != nil {
		return 0, false, fmt.Errorf("upsert repo: %w", err)
	}

	if inserted {
		if err := tx.QueryRow(`SELECT id FROM repos WHERE source = ? AND owner = ? AND name = ?`,
			rec.Source, rec.Owner, rec.Name).Scan(&id); err != nil {
			return 0, false, fmt.Errorf("read back inserted id: %w", err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM topics WHERE repo_id = ?`, id); err != nil {
		return 0, false, fmt.Errorf("clear topics: %w", err)
	}
	for _, topic := range rec.Topics {
		if topic == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO topics (repo_id, topic) VALUES (?, ?)`, id, topic); err != nil {
			return 0, false, fmt.Errorf("insert topic %q: %w", topic, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, false, fmt.Errorf("commit: %w", err)
	}
	return id, inserted, nil
}

// GetByFullName returns the repo with the given source and owner/name
// ("owner/name"), or sql.ErrNoRows if not found.
func (s *Store) GetByFullName(source, fullName string) (model.Repo, error) {
	row := s.db.QueryRow(`SELECT `+repoColumns+` FROM repos WHERE source = ? AND full_name = ?`, source, fullName)
	return scanRepo(row)
}

// FindByFullName looks up a repo by owner/name across all sources. It
// returns sql.ErrNoRows if none match, or an error listing the matching
// sources if the same full_name exists under more than one (e.g. once
// non-GitHub sources exist) — callers can disambiguate with GetByFullName.
func (s *Store) FindByFullName(fullName string) (model.Repo, error) {
	rows, err := s.db.Query(`SELECT `+repoColumns+` FROM repos WHERE full_name = ?`, fullName)
	if err != nil {
		return model.Repo{}, fmt.Errorf("find repo: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var matches []model.Repo
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return model.Repo{}, fmt.Errorf("scan repo: %w", err)
		}
		matches = append(matches, r)
	}
	if err := rows.Err(); err != nil {
		return model.Repo{}, err
	}

	switch len(matches) {
	case 0:
		return model.Repo{}, sql.ErrNoRows
	case 1:
		return matches[0], nil
	default:
		sources := make([]string, len(matches))
		for i, m := range matches {
			sources[i] = m.Source
		}
		return model.Repo{}, fmt.Errorf("%q exists in multiple sources %v; use GetByFullName with a specific source", fullName, sources)
	}
}

// ListFilters narrows the result set of List.
type ListFilters struct {
	Source       string // exact match, "" = any
	Language     string // exact match (case-insensitive), "" = any
	Topic        string // exact match against a single topic, "" = any
	ArchivedOnly bool   // if true, only archived repos
	Sort         string // "stars", "updated" (pushed_at desc), "" = full_name asc
	Limit        int    // 0 = no limit
}

// List returns repos matching filters.
func (s *Store) List(f ListFilters) ([]model.Repo, error) {
	query := `SELECT ` + repoColumns + ` FROM repos WHERE 1=1`
	var args []any

	if f.Source != "" {
		query += ` AND source = ?`
		args = append(args, f.Source)
	}
	if f.Language != "" {
		query += ` AND primary_language = ? COLLATE NOCASE`
		args = append(args, f.Language)
	}
	if f.Topic != "" {
		query += ` AND id IN (SELECT repo_id FROM topics WHERE topic = ? COLLATE NOCASE)`
		args = append(args, f.Topic)
	}
	if f.ArchivedOnly {
		query += ` AND is_archived = 1`
	}

	switch f.Sort {
	case "stars":
		query += ` ORDER BY stars DESC`
	case "updated":
		query += ` ORDER BY pushed_at DESC`
	default:
		query += ` ORDER BY full_name COLLATE NOCASE ASC`
	}

	if f.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, f.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
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

// Delete removes the repos with the given ids (cascades to topics via FK,
// and to repos_fts via the repos_ad trigger). Returns the number removed.
func (s *Store) Delete(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	res, err := s.db.Exec(`DELETE FROM repos WHERE id IN (`+placeholders+`)`, args...) //nolint:gosec // G202: placeholders is just "?," repeated per id, all values still bound via args
	if err != nil {
		return 0, fmt.Errorf("delete repos: %w", err)
	}
	return res.RowsAffected()
}

// ListSources returns the distinct source names already present in the
// store (used by `update` to know what to re-import).
func (s *Store) ListSources() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT source FROM repos ORDER BY source`)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// Now is a var so tests can override it; production code always uses the
// real clock.
var Now = time.Now
