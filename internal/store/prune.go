package store

import (
	"fmt"
	"time"

	"github.com/hueys/repogrep/internal/model"
)

// PruneCandidate is a repo flagged by PruneCandidates, with the reason it
// qualified.
type PruneCandidate struct {
	Repo   model.Repo
	Reason string // "archived", "inactive", or "archived, inactive"
}

// PruneCandidates returns repos that are archived and/or have had no push
// activity in inactiveMonths, ordered by full_name. It never deletes
// anything; callers decide whether to act on the result.
func (s *Store) PruneCandidates(inactiveMonths int) ([]PruneCandidate, error) {
	cutoff := Now().AddDate(0, -inactiveMonths, 0)

	rows, err := s.db.Query(`
		SELECT `+repoColumns+`
		FROM repos
		WHERE is_archived = 1 OR pushed_at < ? OR pushed_at IS NULL
		ORDER BY full_name COLLATE NOCASE ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query prune candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []PruneCandidate
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		out = append(out, PruneCandidate{Repo: r, Reason: pruneReason(r, cutoff)})
	}
	return out, rows.Err()
}

func pruneReason(r model.Repo, cutoff time.Time) string {
	archived := r.IsArchived
	inactive := r.PushedAt.IsZero() || r.PushedAt.Before(cutoff)
	switch {
	case archived && inactive:
		return "archived, inactive"
	case archived:
		return "archived"
	default:
		return "inactive"
	}
}
