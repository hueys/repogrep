package store

import (
	"database/sql"
	"strings"
	"time"
)

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullableTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// splitTopics reconstructs a topics slice from the space-joined
// topics_text column, which mirrors the normalized topics table and is
// what read paths use to avoid an extra join per repo.
func splitTopics(topicsText string) []string {
	fields := strings.Fields(topicsText)
	if len(fields) == 0 {
		return nil
	}
	return fields
}
