// Package model defines the canonical, storage-facing representation of a
// repo. It is the shared vocabulary between internal/store and internal/cli.
package model

import "time"

// Repo is a single repo record as persisted in the store. It is a superset
// of source.RepoRecord: it adds the fields the store itself owns (ID,
// FirstSeen, LastSynced).
type Repo struct {
	ID              int64
	Source          string
	SourceID        string
	Owner           string
	Name            string
	FullName        string
	Description     string
	URL             string
	Homepage        string
	PrimaryLanguage string
	Topics          []string
	Stars           int
	Forks           int
	IsArchived      bool
	IsFork          bool
	License         string
	DefaultBranch   string
	PushedAt        time.Time
	RepoCreatedAt   time.Time
	RepoUpdatedAt   time.Time
	README          string
	ReadmePath      string
	FirstSeen       time.Time
	LastSynced      time.Time
}
